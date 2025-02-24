package core

import (
	"errors"
	"fmt"
	//"strconv"
	"strings"
	"encoding/json"
	"truenas/truenas_incus_ctl/truenas_api"
)

type ApiJobResult struct {
	JobID  int64
	Method string
	State  string
	Result interface{}
	Error  interface{}
}

type RealSession struct {
	HostUrl string
	ApiKey string
	ShouldWait bool
	client *truenas_api.Client
	subscribedToJobs bool
	resultsQueue *SimpleQueue[ApiJobResult]
	jobsList []int64
}

func (s *RealSession) Login() error {
	if s.client != nil {
		// TODO: Clear resultsQueue before calling close here, since we want to log in again immediately after
		_ = s.Close(nil)
	}

	if s.HostUrl == "" || s.ApiKey == "" {
		return errors.New("--url and --api-key were not provided")
	}

	if s.resultsQueue == nil {
		s.resultsQueue = MakeSimpleQueue[ApiJobResult]()
	}

	client, err := truenas_api.NewClientWithCallback(
		s.HostUrl,
		strings.HasPrefix(s.HostUrl, "wss://"),
		func(waitingJobId int64, innerJobId int64, params map[string]interface{}) {
			s.HandleJobUpdate(waitingJobId, innerJobId, params)
		},
	)
	if err != nil {
		return errors.New("Failed to create client: " + err.Error())
	}

	err = client.Login("", "", s.ApiKey)
	if err != nil {
		client.Close()
		return errors.New("Client login failed: " + err.Error())
	}

	s.client = client
	return nil
}

func (s *RealSession) CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
	return s.client.Call(method, timeoutSeconds, params)
}

func (s *RealSession) CallAsyncRaw(method string, params interface{}, awaitThisJob bool) (int64, error) {
	awaitThisJob = awaitThisJob || s.ShouldWait

	if awaitThisJob && !s.subscribedToJobs {
		// For every async call that we call "core.job_wait" on, we'll be notified whenever the original call is updated or completes.
		// In order to get those notifications, we have to subscribe to "core.get_jobs".
		if err := s.client.SubscribeToJobs(); err != nil {
			return -1, err
		}
		s.subscribedToJobs = true
	}

	mainJob, err := s.client.CallWithJob(method, params, nil)
	if err != nil {
		FlushString("Main call error: " + err.Error() + "\n")
		return mainJob.ID, err
	}

	if awaitThisJob {
		// This is to ensure we get notified when mainJob completes.
		_, err := s.client.CallWithJob("core.job_wait", []interface{}{mainJob.ID}, nil)
		if err != nil {
			return mainJob.ID, err
		}

		if s.jobsList == nil {
			s.jobsList = make([]int64, 0)
		}
		s.jobsList = append(s.jobsList, mainJob.ID)
	}

	return mainJob.ID, nil
}

func (s *RealSession) WaitForJob(jobId int64) (json.RawMessage, error) {
	//fmt.Println([]interface{}{"Waiting for job", jobId}...)
	idx := -1
	for i, job := range s.jobsList {
		if job == jobId {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("Job ID %d was not submitted during this session", jobId)
	}

	var res interface{}
	var err error

	irrelevantList := make([]ApiJobResult, 0)
	for true {
		jr := s.resultsQueue.Take()
		if jr.JobID == jobId {
			res = jr.Result
			err = jr.GetError()
			break
		} else {
			irrelevantList = append(irrelevantList, jr)
		}
	}

	for _, jr := range irrelevantList {
		s.resultsQueue.Add(jr)
	}
	s.jobsList[idx] = s.jobsList[len(s.jobsList)-1]
	s.jobsList = s.jobsList[:len(s.jobsList)-1]

	if err != nil || res == nil {
		//fmt.Println([]interface{}{"Job", jobId, "returned with error", err}...)
		return nil, err
	}

	data, err := json.Marshal(res)
	//fmt.Println([]interface{}{"Job", jobId, "returned with result", string(data)}...)
	return data, err
}

func (s *RealSession) Close(internalError error) error {
	if s.client == nil {
		return internalError
	}

	errorList := make([]error, 0)
	if internalError != nil {
		errorList = append(errorList, internalError)
	}

	if s.ShouldWait {
		for len(s.jobsList) > 0 {
			jr := s.resultsQueue.Take()
			//jr.Print()
			for i := 0; i < len(s.jobsList); i++ {
				if s.jobsList[i] == jr.JobID {
					if err := jr.GetError(); err != nil {
						errorList = append(errorList, err)
					}
					// remove it
					s.jobsList[i] = s.jobsList[len(s.jobsList)-1]
					s.jobsList = s.jobsList[:len(s.jobsList)-1]
				}
			}
		}
	}

	err := s.client.Close()
	s.client = nil

	if err != nil {
		errorList = append(errorList, err)
	}

	return MakeErrorFromList(errorList)
}

// This is called from the listen thread in truenas_api
func (s *RealSession) HandleJobUpdate(waitingJobId int64, innerJobId int64, params map[string]interface{}) {
	st, _ := params["state"].(string)
	state := strings.ToUpper(st)

	if state == "SUCCESS" || state == "FAILED" {
		method, _ := params["params"].(string)
		res, _ := params["result"]
		err, _ := params["error"]

		jr := ApiJobResult{
			JobID:  innerJobId,
			Method: method,
			State:  state,
			Result: res,
			Error:  err,
		}
		s.resultsQueue.Add(jr)
	}
}

func (jr *ApiJobResult) GetError() error {
	if jr == nil {
		return nil
	}
	if jr.Error != nil {
		return errors.New(fmt.Sprint(jr.Error))
	}
	if jr.Result == nil {
		return nil
	}

	arrayOfResults := make([]map[string]interface{}, 0)
	if arr, ok := jr.Result.([]interface{}); ok {
		for _, elem := range arr {
			if res, ok := elem.(map[string]interface{}); ok {
				arrayOfResults = append(arrayOfResults, res)
			} else {
				arrayOfResults = append(arrayOfResults, nil)
			}
		}
	} else if res, ok := jr.Result.(map[string]interface{}); ok {
		arrayOfResults = append(arrayOfResults, res)
	}

	errorList := make([]error, 0)
	for i, res := range arrayOfResults {
		if res == nil {
			continue
		}
		if value, exists := res["error"]; exists {
			if str, ok := value.(string); ok && str != "" {
				errorList = append(errorList, fmt.Errorf("%d: %s", i, str))
			} else if value != nil {
				errorList = append(errorList, fmt.Errorf("%d: %v", i, value))
			}
		}
	}

	return MakeErrorFromList(errorList)
}

func (jr *ApiJobResult) Print() {
	if jr == nil {
		fmt.Println("nil")
		return
	}
	fmt.Printf("Job: %d, State: %s, Result: %v, Error: %v\n", jr.JobID, jr.State, jr.Result, jr.Error)
}

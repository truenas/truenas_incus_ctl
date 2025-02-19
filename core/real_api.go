package core

import (
	"errors"
	"fmt"
	"strings"
	"encoding/json"
	"truenas/truenas_incus_ctl/truenas_api"
)

type RealSession struct {
	HostUrl string
	ApiKey string
	ShouldWait bool
	client *truenas_api.Client
}

var realApiJobsList []*truenas_api.Job
var realApiSubscriptions map[string]bool

func (s *RealSession) Login() error {
	if s.client != nil {
		_ = s.Close()
	}

	if s.HostUrl == "" || s.ApiKey == "" {
		return errors.New("--url and --api-key were not provided")
	}

	client, err := truenas_api.NewClient(s.HostUrl, strings.HasPrefix(s.HostUrl, "wss://"))
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

func (s *RealSession) CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	return s.client.Call(method, timeoutStr, params)
}

func (s *RealSession) CallAsyncRaw(method string, params interface{}, callback func(progress float64, state string, desc string)) error {
	if s.ShouldWait {
		if realApiSubscriptions == nil {
			realApiSubscriptions = make(map[string]bool)
		}
		subscribeStr := "sharing.nfs.update"
		if _, exists := realApiSubscriptions[subscribeStr]; !exists {
			realApiSubscriptions[subscribeStr] = true
			event := []interface{}{subscribeStr}
			out, err := s.CallRaw("core.subscribe", "10s", event)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
		}
	}

	job, err := s.client.CallWithJob(method, params, callback)
	if err != nil {
		return err
	}
	fmt.Printf("job=%d\n", job.ID)
	if s.ShouldWait {
		if realApiJobsList == nil {
			realApiJobsList = make([]*truenas_api.Job, 0)
		}
		realApiJobsList = append(realApiJobsList, job)
	}
	return nil
}

func (s *RealSession) Close() error {
	if s.client == nil {
		return nil
	}

	if s.ShouldWait {
		fmt.Println("Waiting for", len(realApiJobsList), "jobs to finish")
	}

	var err error
	for _, job := range realApiJobsList {
		if job != nil {
			// TODO: Also wait on either timeout or SIGKILL
			fmt.Printf("%d\t%s\t%s", job.ID, job.State, job.Method)
			status := <- job.DoneCh
			fmt.Printf("\t%s\n", status)
		}
	}

	err = s.client.Close()
	s.client = nil
	return err
}

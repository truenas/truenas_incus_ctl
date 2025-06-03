package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type ClientSession struct {
	HostName string
	ApiKey string
	SocketPath string
	IsDebug bool
	AllowInsecure bool
	client *http.Client
	timeout time.Duration
	jobsList []int64
	mapSkipWaitOnClose map[int64]bool
}

func (s *ClientSession) IsLoggedIn() bool {
	return s.client != nil
}

func (s *ClientSession) GetHostName() string {
	return GetHostNameFromApiUrl(s.HostName)
}
func (s *ClientSession) GetUrl() string {
	return GetApiUrlFromHostName(s.HostName)
}

func (s *ClientSession) Login() error {
	var t1 time.Time
	if s.IsDebug {
		t1 = time.Now()
	}

	if s.jobsList == nil {
		s.jobsList = make([]int64, 0)
	}

	var errBuilder strings.Builder
	if s.HostName == "" {
		errBuilder.WriteString("Hostname was not provided\n")
	}
	if s.ApiKey == "" {
		errBuilder.WriteString("API key was not provided\n")
	}
	if s.SocketPath == "" {
		errBuilder.WriteString("Socket path was not provided\n")
	}
	if errBuilder.Len() > 0 {
		return fmt.Errorf(errBuilder.String())
	}

	s.timeout = time.Duration(180) * time.Second

	st, err := os.Stat(s.SocketPath)
	if err != nil {
		if err = launchDaemonAndAwaitSocket(s.SocketPath, s.timeout, nil); err != nil {
			return fmt.Errorf("launchDaemonAndAwaitSocket: %v", err)
		}
		st, err = os.Stat(s.SocketPath)
	}

	if err != nil {
		return err
	}
	if (st.Mode() & fs.ModeSocket) == 0 {
		return fmt.Errorf("%s was not a socket", s.SocketPath)
	}

	if s.client == nil {
		s.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return net.Dial("unix", s.SocketPath)
				},
			},
		}
	}

	if s.IsDebug {
		fmt.Println("tncdaemon connection time:", time.Now().Sub(t1).String())
	}

	return nil
}

func (s *ClientSession) CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
	var t1 time.Time
	if s.IsDebug {
		t1 = time.Now()
	}

	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request, _ := http.NewRequest("POST", "http://unix/tnc-daemon", bytes.NewReader(paramsData))
	request.Header.Set("TNC-Host-Url", s.GetUrl())
	request.Header.Set("TNC-Api-Key", s.ApiKey)
	request.Header.Set("TNC-Allow-Insecure", fmt.Sprint(s.AllowInsecure))
	request.Header.Set("TNC-Call-Method", method)
	if timeoutSeconds > 0 {
		request.Header.Set("TNC-Timeout", fmt.Sprintf("%ds", timeoutSeconds))
	}

call:
	response, err := s.client.Do(request)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		errMsg := err.Error()
		if strings.Contains(errMsg, ": dial unix") {
			os.Remove(s.SocketPath)
			err = s.Login()
			if err == nil {
				goto call
			}
		}
		return nil, err
	}

	data, err := io.ReadAll(response.Body)
	response.Body.Close()

	if s.IsDebug {
		fmt.Println(method + ":", time.Now().Sub(t1).String())
	}

	if err != nil {
		return data, err
	}
	if response.StatusCode >= 400 {
		return nil, errors.New("Error: " + string(data))
	}
	return data, err
}

func (s *ClientSession) CallAsyncRaw(method string, params interface{}) (int64, error) {
	data, err := s.CallRaw(method, 0, params)
	if err != nil {
		return -1, err
	}
	jobId, _ := GetJobNumber(data)
	s.jobsList = append(s.jobsList, jobId)
	return jobId, nil
}

func (s *ClientSession) WaitForJob(jobId int64) (json.RawMessage, error) {
	return s.CallRaw("tnc_daemon.await_job", 0, []interface{} {jobId})
}

func (s *ClientSession) SkipWaitingJobOnClose(jobId int64) {
	s.mapSkipWaitOnClose[jobId] = true
}

func (s *ClientSession) Close(internalError error) error {
	if s.client == nil {
		return internalError
	}

	errorList := make([]error, 0)
	if internalError != nil {
		errorList = append(errorList, internalError)
	}

	for _, jobId := range s.jobsList {
		if jobId < 0 {
			continue
		}
		if shouldSkip, _ := s.mapSkipWaitOnClose[jobId]; shouldSkip {
			continue
		}
		data, err := s.CallRaw("tnc_daemon.await_job", 0, []interface{} {jobId})
		if err != nil {
			errorList = append(errorList, err)
		} else if data != nil {
			_, errs := GetResultsAndErrorsFromApiResponseRaw(data)
			for _, e := range errs {
				errorList = append(errorList, errors.New(ExtractApiErrorJsonGivenError(e)))
			}
		}
	}

	s.client = nil
	return MakeErrorFromList(errorList)
}

func launchDaemonAndAwaitSocket(socketPath string, daemonTimeout time.Duration, optWarningBuilder *strings.Builder) error {
	thisExec, err := os.Executable()
	if err != nil {
		return err
	}

	lastSlash := strings.LastIndex(socketPath, "/")
	if lastSlash == -1 {
		socketPath = "./" + socketPath
		lastSlash = 1
	}

	doneCh := make(chan error)
	go func() {
		defer close(doneCh)
		sockFolder := path.Dir(socketPath)
		sockName := socketPath[lastSlash+1:]

		WaitForFilesToAppear(sockFolder, func(fname string, isCreated bool) bool {
			return isCreated && fname != "" && strings.Contains(fname, sockName)
		})
	}()

	cmd := []string { "daemon" }
	if daemonTimeout >= time.Second {
		cmd = append(cmd, "-t", daemonTimeout.String())
	}
	cmd = append(cmd, socketPath)

	if err = exec.Command(thisExec, cmd...).Start(); err != nil {
		return fmt.Errorf("Failed to launch daemon: %v", err)
	}

	tmDuration := time.Duration(500) * time.Millisecond
	timeoutCh := time.After(tmDuration)

	select {
	case err = <- doneCh:
		break
	case <- timeoutCh:
		if optWarningBuilder != nil {
			optWarningBuilder.WriteString("Waiting for socket creation timed out after ")
			optWarningBuilder.WriteString(tmDuration.String())
			optWarningBuilder.WriteString("\n")
		}
		err = nil
	}

	if err != nil {
		if optWarningBuilder != nil {
			optWarningBuilder.WriteString("launchDaemonAndAwaitSocket inotify warning: ")
			optWarningBuilder.WriteString(err.Error())
			optWarningBuilder.WriteString("\n")
		}
		_ = <- timeoutCh
	}

	return nil
}

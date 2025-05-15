package core

import (
	"encoding/json"
	"errors"
	//"fmt"
)

type Session interface {
	Login() error
	IsLoggedIn() bool
	GetHostName() string
	GetUrl() string
	CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error)
	CallAsyncRaw(method string, params interface{}) (int64, error)
	WaitForJob(jobId int64) (json.RawMessage, error)
	SkipWaitingJobOnClose(jobId int64)
	Close(error) error
}

func MaybeLogin(s Session) error {
	if s.IsLoggedIn() {
		return nil
	}
	return s.Login()
}

func ApiCall(s Session, method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
	if err := MaybeLogin(s); err != nil {
		return nil, err
	}
	out, err := s.CallRaw(method, timeoutSeconds, params)
	if err != nil {
		return out, err
	}
	if errMsg := ExtractApiError(out); errMsg != "" {
		return out, errors.New(errMsg)
	}
	return out, nil
}

func ApiCallAsync(s Session, method string, params interface{}, awaitThisJob bool) (int64, error) {
	if err := MaybeLogin(s); err != nil {
		return -1, err
	}
	jobId, err := s.CallAsyncRaw(method, params)
	if err != nil && jobId > 0 && !awaitThisJob {
		s.SkipWaitingJobOnClose(jobId)
	}
	return jobId, err
}

package core

import (
	"encoding/json"
	"errors"
	//"fmt"
)

type Session interface {
	Login() error
	CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error)
	CallAsyncRaw(method string, params interface{}, awaitThisJob bool) (int64, error)
	WaitForJob(jobId int64) (json.RawMessage, error)
	Close(error) error
}

func ApiCall(s Session, method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
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
	return s.CallAsyncRaw(method, params, awaitThisJob)
}

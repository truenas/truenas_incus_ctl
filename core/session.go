package core

import (
	"encoding/json"
	"errors"
)

type Session interface {
	Login() error
	CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error)
	CallStringRaw(method string, timeoutStr string, paramsStr string) (json.RawMessage, error)
	Close() error
}

func ApiCall(s Session, method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	out, err := s.CallRaw(method, timeoutStr, params)
	if err != nil {
		return out, err
	}
	if errMsg := ExtractApiError(out); errMsg != "" {
		return out, errors.New(errMsg)
	}
	return out, nil
}

func ApiCallString(s Session, method string, timeoutStr string, paramsStr string) (json.RawMessage, error) {
	out, err := s.CallStringRaw(method, timeoutStr, paramsStr)
	if err != nil {
		return out, err
	}
	if errMsg := ExtractApiError(out); errMsg != "" {
		return out, errors.New(errMsg)
	}
	return out, nil
}

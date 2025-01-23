package core

import (
	"encoding/json"
)

type Session interface {
	Login() error
	Call(method string, timeoutStr string, params interface{}) (json.RawMessage, error)
	CallString(method string, timeoutStr string, paramsStr string) (json.RawMessage, error)
	Close() error
}

func GetApi() Session {
	if (false) {
		return &RealSession{}
	}
	return &MockSession{}
}

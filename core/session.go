package core

import (
    "encoding/json"
)

type Session interface {
    Login() error
    Call(method string, timeoutStr string, params interface{}) (json.RawMessage, error)
    CallStrings(method string, timeoutStr string, params []string) (json.RawMessage, error)
    Close() error
}

func GetApi() Session {
    if (true) {
        return &RealSession{}
    }
    return &MockSession{}
}

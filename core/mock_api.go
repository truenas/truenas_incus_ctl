package core

import (
    "fmt"
    "errors"
    "strings"
    "encoding/json"
)

type MockSession struct {
    closed bool
}

func (s *MockSession) Login() error {
    s.closed = false
    return nil
}

func (s *MockSession) Close() error {
    s.closed = true
    return nil
}

func (s *MockSession) Call(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
    if s.closed {
        return nil, errors.New("API connection closed")
    }
    switch (method) {
        case "pool.dataset.create":
            return mockDatasetCreate(params)
        case "pool.dataset.delete":
            return mockDatasetDelete(params)
        case "pool.dataset.query":
            return mockDatasetQuery(params)
        default:
            return nil, errors.New("Unrecognised command " + method)
    }
}

func (s *MockSession) CallStrings(method string, timeoutStr string, params []string) (json.RawMessage, error) {
    var paramsUnmarsalled interface{}
    if err := json.Unmarshal([]byte("[" + strings.Join(params, ",") + "]"), &paramsUnmarsalled); err != nil {
		return nil, fmt.Errorf("failed to parse params string: %w", err)
	}
	return s.Call(method, timeoutStr, paramsUnmarsalled)
}

func mockDatasetCreate(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetCreate() not yet implemented")
}

func mockDatasetDelete(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetDelete() not yet implemented")
}

func mockDatasetQuery(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetQuery() not yet implemented")
}

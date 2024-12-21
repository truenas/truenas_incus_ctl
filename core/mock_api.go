package core

import (
    "fmt"
    "errors"
    //"strings"
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

func (s *MockSession) CallString(method string, timeoutStr string, paramsStr string) (json.RawMessage, error) {
    var paramsUnmarsalled interface{}
    if err := json.Unmarshal([]byte(paramsStr), &paramsUnmarsalled); err != nil {
		return nil, fmt.Errorf("failed to parse params string: %w", err)
	}
	return s.Call(method, timeoutStr, paramsUnmarsalled)
}

func (s *MockSession) Call(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
    if s.closed {
        return nil, errors.New("API connection closed")
    }
    switch (method) {
        case "pool.dataset.create":
            return mockDatasetCreate(params)
        case "zfs.dataset.create":
            return mockDatasetCreate(params)
        case "pool.dataset.delete":
            return mockDatasetDelete(params)
        case "zfs.dataset.delete":
            return mockDatasetDelete(params)
        case "pool.dataset.query":
            return mockDatasetQuery(params)
        case "zfs.dataset.query":
            return mockDatasetQuery(params)
        default:
            return nil, errors.New("Unrecognised command " + method)
    }
}

func mockDatasetCreate(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetCreate() not yet implemented")
}

func mockDatasetDelete(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetDelete() not yet implemented")
}

type typeQueryParams struct {
    datasetName string
    properties []string
    isFlat bool
    withChildren bool
    withUser bool
}

func getQueryParams(paramsList []interface{}) (typeQueryParams, error) {
    q := typeQueryParams{}
    cur := 0
    if cur >= len(paramsList) {
        return q, nil
    }
    if filterParamOuter, ok := paramsList[cur].([]interface{}); ok {
        if len(filterParamOuter) < 1 {
            return q, errors.New("Could not find dataset name in name filter")
        }
        if filterParam, ok := filterParamOuter[0].([]interface{}); ok {
            if len(filterParam) >= 3 {
                if idString, ok := filterParam[2].(string); ok {
                    q.datasetName = idString
                } else if idArray, ok := filterParam[2].([]interface{}); ok {
                    if idString, ok := idArray[0].(string); ok {
                        q.datasetName = idString
                    }
                }
            }
            if q.datasetName == "" {
                return q, errors.New("Could not find dataset name in name filter")
            }
            cur++
        }
    }
    if cur >= len(paramsList) {
        return q, nil
    }
    if propsParam, ok := paramsList[cur].(map[string]interface{}); ok {
        var extraMap map[string]interface{}
        if extra, ok := propsParam["extra"]; ok {
            extraMap, ok = extra.(map[string]interface{})
        }
        if extraMap == nil {
            return q, errors.New("Could not find query options in query filter")
        }
        if value, ok := extraMap["flat"]; ok {
            q.isFlat, ok = value.(bool)
        }
        if value, ok := extraMap["retrieve_children"]; ok {
            q.withChildren, ok = value.(bool)
        }
        if value, ok := extraMap["user_properties"]; ok {
            q.withUser, ok = value.(bool)
        }
        if value, ok := extraMap["properties"]; ok {
            if props, ok := value.([]interface{}); ok {
                for _, elem := range(props) {
                    str := ""
                    if str, ok = elem.(string); !ok {
                        str = fmt.Sprint(elem)
                    }
                    q.properties = append(q.properties, str)
                }
            }
        }
    }
    return q, nil
}

func mockDatasetQuery(params interface{}) (json.RawMessage, error) {
    q := typeQueryParams{}
    if paramsList, ok := params.([]interface{}); ok {
        var err error
        q, err = getQueryParams(paramsList)
        if err != nil {
            return nil, err
        }
    }

    fmt.Println("datasetName:", q.datasetName)
    fmt.Println("properties:", q.properties)
    fmt.Println("isFlat:", q.isFlat)
    fmt.Println("withChildren:", q.withChildren)
    fmt.Println("withUser:", q.withUser)

    return nil, errors.New("yep")
}

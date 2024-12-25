package core

import (
    "fmt"
    "errors"
    "strings"
    "encoding/json"
)

type MockDataset struct {
    name string
    poolName string
    properties map[string]string
}

type MockSession struct {
    closed bool
    datasets map[string]MockDataset
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
            return s.mockDatasetCreate(params)
        case "zfs.dataset.create":
            return s.mockDatasetCreate(params)
        case "pool.dataset.delete":
            return s.mockDatasetDelete(params)
        case "zfs.dataset.delete":
            return s.mockDatasetDelete(params)
        case "pool.dataset.query":
            return s.mockDatasetQuery(params)
        case "zfs.dataset.query":
            return s.mockDatasetQuery(params)
        default:
            return nil, errors.New("Unrecognised command " + method)
    }
}

type typeDatasetParams struct {
    datasetName string
    properties []string
    isFlat bool
    withChildren bool
    withUser bool
}

func getDatasetParams(paramsList []interface{}) (typeDatasetParams, error) {
    dp := typeDatasetParams{}
    cur := 0
    if cur >= len(paramsList) {
        return dp, nil
    }
    if filterParamOuter, ok := paramsList[cur].([]interface{}); ok {
        if len(filterParamOuter) < 1 {
            return dp, errors.New("Could not find dataset name in name filter")
        }
        if filterParam, ok := filterParamOuter[0].([]interface{}); ok {
            if len(filterParam) >= 3 {
                if idString, ok := filterParam[2].(string); ok {
                    dp.datasetName = idString
                } else if idArray, ok := filterParam[2].([]interface{}); ok {
                    if idString, ok := idArray[0].(string); ok {
                        dp.datasetName = idString
                    }
                }
            }
            if dp.datasetName == "" {
                return dp, errors.New("Could not find dataset name in name filter")
            }
            cur++
        }
    }
    if cur >= len(paramsList) {
        return dp, nil
    }
    if propsParam, ok := paramsList[cur].(map[string]interface{}); ok {
        var extraMap map[string]interface{}
        if extra, ok := propsParam["extra"]; ok {
            extraMap, ok = extra.(map[string]interface{})
        }
        if extraMap == nil {
            return dp, errors.New("Could not find dataset options in the parameters")
        }
        if value, ok := extraMap["flat"]; ok {
            dp.isFlat, ok = value.(bool)
        }
        if value, ok := extraMap["retrieve_children"]; ok {
            dp.withChildren, ok = value.(bool)
        }
        if value, ok := extraMap["user_properties"]; ok {
            dp.withUser, ok = value.(bool)
        }
        if value, ok := extraMap["properties"]; ok {
            if props, ok := value.([]interface{}); ok {
                for _, elem := range(props) {
                    str := ""
                    if str, ok = elem.(string); !ok {
                        str = fmt.Sprint(elem)
                    }
                    dp.properties = append(dp.properties, str)
                }
            }
        }
    }
    return dp, nil
}

func (s *MockSession) mockDatasetCreate(params interface{}) (json.RawMessage, error) {
    dp := typeDatasetParams{}
    if paramsList, ok := params.([]interface{}); ok {
        var err error
        dp, err = getDatasetParams(paramsList)
        if err != nil {
            return nil, err
        }
    }

    if dp.datasetName == "" {
        return nil, errors.New("No dataset name was provided")
    }
    if _, exists := s.datasets[dp.datasetName]; exists {
        return nil, errors.New("Dataset already exists")
    }

    newDataset := MockDataset{}
    newDataset.properties = make(map[string]string)
    newDataset.name = dp.datasetName
    newDataset.poolName = "foo"

    s.datasets[dp.datasetName] = newDataset

    var output strings.Builder
    writeDatasetInfo(&output, &newDataset, &dp)
    outStr := output.String()
    fmt.Println(outStr)
    return []byte(outStr), nil
}

func (s *MockSession) mockDatasetDelete(params interface{}) (json.RawMessage, error) {
    return nil, errors.New("mockDatasetDelete() not yet implemented")
}

func writeDatasetInfo(output *strings.Builder, dataset *MockDataset, dp *typeDatasetParams) {
    output.WriteString("{ \"id\": \"")
    output.WriteString(dataset.name)
    output.WriteString("\", \"type\": \"FILESYSTEM\", \"name\": \"")
    output.WriteString(dataset.name)
    output.WriteString("\", \"pool\": \"")
    output.WriteString(dataset.poolName)
    output.WriteString("\", ")
    if len(dp.properties) > 0 {
        output.WriteString("\"properties\": {")
        for idx, prop := range(dp.properties) {
            if idx > 0 {
                output.WriteString(",\n")
            }
            value, _ := dataset.properties[prop]
            output.WriteString("\"")
            output.WriteString(prop)
            output.WriteString("\": {\"value\": \"")
            output.WriteString(value)
            output.WriteString("\", \"rawvalue\": \"")
            output.WriteString(value)
            output.WriteString("\", \"source\": \"LOCAL\", \"parsed\": \"")
            output.WriteString(value)
            output.WriteString("\"}")
        }
        output.WriteString("},\n")
    }
    output.WriteString("\"comments\": { \"value\": \"\", \"rawvalue\": \"\", \"source\": \"LOCAL\", \"parsed\": \"\" }, \"user_properties\": {} }")
}

func (s *MockSession) mockDatasetQuery(params interface{}) (json.RawMessage, error) {
    dp := typeDatasetParams{}
    if paramsList, ok := params.([]interface{}); ok {
        var err error
        dp, err = getDatasetParams(paramsList)
        if err != nil {
            return nil, err
        }
    }

    /*
    {
        "jsonrpc": "2.0",
        "result": [
            
        ],
        "id": 2
    }
    */

    var output strings.Builder
    output.WriteString("{\"jsonrpc\": \"2.0\", \"result\": [")

    if dp.datasetName != "" {
        if dataset, exists := s.datasets[dp.datasetName]; exists {
            writeDatasetInfo(&output, &dataset, &dp)
        }
    } else {
        idx := 0
        for _, dataset := range(s.datasets) {
            if idx > 0 {
                output.WriteString(", ")
            }
            writeDatasetInfo(&output, &dataset, &dp)
            idx++
        }
    }

    fmt.Println("datasetName:", dp.datasetName)
    fmt.Println("properties:", dp.properties)
    fmt.Println("isFlat:", dp.isFlat)
    fmt.Println("withChildren:", dp.withChildren)
    fmt.Println("withUser:", dp.withUser)

    output.WriteString("], \"id\": 2}")
    outStr := output.String()
    fmt.Println(outStr)
    return []byte(outStr), nil
}

package core

import (
    "os"
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

func getPoolNameFromDataset(datasetName string) string {
    firstSlash := strings.Index(datasetName, "/")
    var poolName string
    if firstSlash > 0 && firstSlash < len(datasetName) - 1 {
        poolName = datasetName[0:firstSlash]
    }
    return poolName
}

type MockDataset struct {
    name string
    properties map[string]string
    userProps map[string]string
}

func loadMockDatasets() map[string]MockDataset {
    var datasets map[string]MockDataset
    content, err := os.ReadFile("datasets.tsv")
    if err != nil || len(content) == 0 {
        return datasets
    }

    lines := strings.Split(string(content), "\n")
    for _, l := range(lines) {
        values := strings.Split(l, "\t")
        if len(l) < 1 {
            continue
        }
        d := MockDataset{}
        d.name = values[0]
        for j := 1; j < len(values) - 1; j += 2 {
            key := values[j]
            value := values[j+1]
            if strings.HasPrefix(key, "user:") {
                if d.userProps == nil {
                    d.userProps = make(map[string]string)
                }
                d.userProps[key[5:]] = value
            } else {
                if d.properties == nil {
                    d.properties = make(map[string]string)
                }
                d.properties[key] = value
            }
        }
        if datasets == nil {
            datasets = make(map[string]MockDataset)
        }
        datasets[d.name] = d
    }

    return datasets
}

func saveMockDatasets(datasets *map[string]MockDataset) {
    var output strings.Builder
    idx := 0
    for _, d := range(*datasets) {
        output.WriteString(d.name)
        for key, value := range(d.properties) {
            output.WriteString("\t")
            output.WriteString(key)
            output.WriteString("\t")
            output.WriteString(value)
        }
        for key, value := range(d.userProps) {
            output.WriteString("\tuser:")
            output.WriteString(key)
            output.WriteString("\t")
            output.WriteString(value)
        }
        output.WriteString("\n")
        idx++
    }

    if idx > 0 {
        _ = os.WriteFile("datasets.tsv", []byte(output.String()), 0666)
    }
}

type typeCreateDatasetParams struct {
    datasetName string
    comments string
    properties map[string]string
    userProps map[string]string
}

func getCreateDatasetParams(params interface{}) (typeCreateDatasetParams, error) {
    cdp := typeCreateDatasetParams{}
    paramsMap, ok := params.(map[string]interface{})
    if !ok {
        return cdp, errors.New("Parameters for 'create' must be in the form of a map")
    }

    if value, ok := paramsMap["name"]; ok {
        if cdp.datasetName, ok = value.(string); !ok {
            return cdp, errors.New("Name was not a string")
        }
        if strings.Index(cdp.datasetName, "/") <= 0 {
            return cdp, errors.New("Dataset name must contain its pool name before a slash, eg. puddle/example")
        }
    }
    if value, ok := paramsMap["comments"]; ok {
        if cdp.comments, ok = value.(string); !ok {
            return cdp, errors.New("Comments was not a string")
        }
    }
    if value, ok := paramsMap["properties"]; ok {
        var inMap map[string]interface{}
        if inMap, ok = value.(map[string]interface{}); !ok {
            return cdp, errors.New("Properties was not a map/dictionary")
        }
        for key, pv := range(inMap) {
            pvStr, ok := pv.(string)
            if !ok {
                return cdp, errors.New("Property '" + key + "' was not a string")
            }
            if cdp.properties == nil {
                cdp.properties = make(map[string]string)
            }
            cdp.properties[key] = pvStr
        }
    }
    if value, ok := paramsMap["user_properties"]; ok {
        var inMapList []interface{}
        if inMapList, ok = value.([]interface{}); !ok {
            return cdp, errors.New("User Properties was not an array of map/dictionary")
        }
        for _, elem := range(inMapList) {
            inMap, ok := elem.(map[string]interface{})
            if !ok {
                return cdp, errors.New("User Properties was not entirely an array of map/dictionary")
            }

            var uKeyStr string
            uKey, keyExists := inMap["key"]
            if !keyExists {
                return cdp, errors.New("User Property did not contain a 'key'")
            }
            if uKeyStr, ok = uKey.(string); !ok {
                return cdp, errors.New("User Property key was not a string")
            }

            var uValueStr string
            uValue, valueExists := inMap["value"]
            if !valueExists {
                return cdp, errors.New("User Property '" + uKeyStr + "' did not contain a value")
            }
            if uValueStr, ok = uValue.(string); !ok {
                return cdp, errors.New("User Property '" + uKeyStr + "' value was not a string")
            }

            if cdp.userProps == nil {
                cdp.userProps = make(map[string]string)
            }
            cdp.userProps[uKeyStr] = uValueStr
        }
    }
    return cdp, nil
}

func (s *MockSession) mockDatasetCreate(params interface{}) (json.RawMessage, error) {
    cdp, err := getCreateDatasetParams(params)
    if err != nil {
        return nil, err
    }

    if cdp.datasetName == "" {
        return nil, errors.New("No dataset name was provided")
    }

    datasets := loadMockDatasets()

    if datasets != nil {
        if _, exists := datasets[cdp.datasetName]; exists {
            return nil, errors.New("Dataset already exists")
        }
    } else {
        datasets = make(map[string]MockDataset)
    }

    newDataset := MockDataset{}
    newDataset.properties = make(map[string]string)
    newDataset.name = cdp.datasetName

    datasets[cdp.datasetName] = newDataset
    saveMockDatasets(&datasets)

    propertyKeys := make([]string, 0, len(cdp.properties))
    for key, _ := range(cdp.properties) {
        propertyKeys = append(propertyKeys, key)
    }

    var output strings.Builder
    writeDatasetInfo(&output, &newDataset, propertyKeys, nil)
    return []byte(output.String()), nil
}

func (s *MockSession) mockDatasetDelete(params interface{}) (json.RawMessage, error) {
    datasetName, ok := params.(string)
    if !ok {
        return nil, errors.New("dataset delete requires a string, representing the name of the dataset to delete")
    }

    datasets := loadMockDatasets()

    if datasets == nil {
        return nil, errors.New("Dataset does not exist")
    }
    if _, exists := datasets[datasetName]; !exists {
        return nil, errors.New("Dataset does not exist")
    }

    delete(datasets, datasetName)
    saveMockDatasets(&datasets)

    return []byte("True"), nil
}

type typeQueryDatasetParams struct {
    datasetName string
    properties []string
    isFlat bool
    withChildren bool
    withUser bool
}

func getQueryDatasetParams(paramsList []interface{}) (typeQueryDatasetParams, error) {
    qdp := typeQueryDatasetParams{}
    cur := 0
    if cur >= len(paramsList) {
        return qdp, nil
    }
    if filterParamOuter, ok := paramsList[cur].([]interface{}); ok {
        if len(filterParamOuter) < 1 {
            return qdp, errors.New("Could not find dataset name in name filter")
        }
        if filterParam, ok := filterParamOuter[0].([]interface{}); ok {
            if len(filterParam) >= 3 {
                if idString, ok := filterParam[2].(string); ok {
                    qdp.datasetName = idString
                } else if idArray, ok := filterParam[2].([]interface{}); ok {
                    if idString, ok := idArray[0].(string); ok {
                        qdp.datasetName = idString
                    }
                }
            }
            if qdp.datasetName == "" {
                return qdp, errors.New("Could not find dataset name in name filter")
            }
            cur++
        }
    }
    if cur >= len(paramsList) {
        return qdp, nil
    }
    if propsParam, ok := paramsList[cur].(map[string]interface{}); ok {
        var extraMap map[string]interface{}
        if extra, ok := propsParam["extra"]; ok {
            extraMap, ok = extra.(map[string]interface{})
        }
        if extraMap == nil {
            return qdp, errors.New("Could not find dataset options in the parameters")
        }
        if value, ok := extraMap["flat"]; ok {
            qdp.isFlat, ok = value.(bool)
        }
        if value, ok := extraMap["retrieve_children"]; ok {
            qdp.withChildren, ok = value.(bool)
        }
        if value, ok := extraMap["user_properties"]; ok {
            qdp.withUser, ok = value.(bool)
        }
        if value, ok := extraMap["properties"]; ok {
            if props, ok := value.([]interface{}); ok {
                for _, elem := range(props) {
                    str := ""
                    if str, ok = elem.(string); !ok {
                        str = fmt.Sprint(elem)
                    }
                    qdp.properties = append(qdp.properties, str)
                }
            }
        }
    }
    return qdp, nil
}

func writeDatasetInfo(output *strings.Builder, dataset *MockDataset, propertiesKeys []string, userPropsKeys []string) {
    poolName := getPoolNameFromDataset(dataset.name)
    output.WriteString("{ \"id\":\"")
    output.WriteString(dataset.name)
    output.WriteString("\", \"type\":\"FILESYSTEM\", \"name\":\"")
    output.WriteString(dataset.name)
    output.WriteString("\", \"pool\":\"")
    output.WriteString(poolName)
    output.WriteString("\", ")
    if propertiesKeys != nil {
        output.WriteString("\"properties\":{")
        for idx, prop := range(propertiesKeys) {
            if idx > 0 {
                output.WriteString(",\n")
            }
            value, _ := dataset.properties[prop]
            output.WriteString("\"")
            output.WriteString(prop)
            output.WriteString("\":{\"value\":\"")
            output.WriteString(value)
            output.WriteString("\", \"rawvalue\":\"")
            output.WriteString(value)
            output.WriteString("\", \"source\":\"LOCAL\", \"parsed\":\"")
            output.WriteString(value)
            output.WriteString("\"}")
        }
        output.WriteString("},\n")
    }
    output.WriteString("\"comments\":{ \"value\":\"\", \"rawvalue\":\"\", \"source\":\"LOCAL\", \"parsed\":\"\" }, \"user_properties\":{} }")
}

func (s *MockSession) mockDatasetQuery(params interface{}) (json.RawMessage, error) {
    qdp := typeQueryDatasetParams{}
    if paramsList, ok := params.([]interface{}); ok {
        var err error
        qdp, err = getQueryDatasetParams(paramsList)
        if err != nil {
            return nil, err
        }
    }

    datasets := loadMockDatasets()

    var output strings.Builder
    output.WriteString("{\"jsonrpc\":\"2.0\", \"result\":[")

    if datasets != nil {
        if qdp.datasetName != "" {
            if dataset, exists := datasets[qdp.datasetName]; exists {
                writeDatasetInfo(&output, &dataset, qdp.properties, nil)
            }
        } else {
            idx := 0
            for _, dataset := range(datasets) {
                if idx > 0 {
                    output.WriteString(", ")
                }
                writeDatasetInfo(&output, &dataset, qdp.properties, nil)
                idx++
            }
        }
    }

    output.WriteString("], \"id\":2}")
    outStr := output.String()
    return []byte(outStr), nil
}

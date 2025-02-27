package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type MockSession struct {
	DatasetSource ReadAllWriteAll
	closed bool
}

func (s *MockSession) Login() error {
	s.closed = false
	return nil
}

func (s *MockSession) Close(internalError error) error {
	s.closed = true
	return nil
}

func (s *MockSession) CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
	if s.closed {
		return nil, errors.New("API connection closed")
	}

	// se: I think this is a valid assertion, and should help catch logic issues.
	_, ok := params.([]interface{})
	if !ok {
		return nil, errors.New("params for Call must be in the form of an array")
	}

	switch method {
	case "pool.dataset.create":
		return s.mockDatasetCreate(params)
	case "zfs.dataset.create":
		return s.mockDatasetCreate(params)
	case "pool.dataset.update":
		return s.mockDatasetUpdate(params)
	case "zfs.dataset.update":
		return s.mockDatasetUpdate(params)
	case "pool.dataset.delete":
		return s.mockDatasetDelete(params)
	case "zfs.dataset.delete":
		return s.mockDatasetDelete(params)
	case "pool.dataset.query":
		return s.mockDatasetQuery(params)
	case "zfs.dataset.query":
		return s.mockDatasetQuery(params)
	case "zfs.dataset.rename":
		return s.mockDatasetRename(params)
	default:
		return nil, errors.New("Unrecognised command " + method)
	}
}

func (s *MockSession) CallAsyncRaw(method string, params interface{}, awaitThisJob bool) (int64, error) {
	return -1, errors.New("async calls not supported in mock")
}

func (s *MockSession) WaitForJob(jobId int64) (json.RawMessage, error) {
	return nil, errors.New("jobs are not supported in mock")
}

func getPoolNameFromDataset(datasetName string) string {
	firstSlash := strings.Index(datasetName, "/")
	var poolName string
	if firstSlash > 0 && firstSlash < len(datasetName)-1 {
		poolName = datasetName[0:firstSlash]
	}
	return poolName
}

type MockDataset struct {
	name       string
	properties map[string]string
	userProps  map[string]string
}

func loadMockDatasets(source ReadAllWriteAll) map[string]MockDataset {
	var datasets map[string]MockDataset
	content, err := source.ReadAll()
	if err != nil || len(content) == 0 {
		return datasets
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		values := strings.Split(l, "\t")
		if len(l) < 1 {
			continue
		}
		d := MockDataset{}
		d.name = values[0]
		for j := 1; j < len(values)-1; j += 2 {
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

func saveMockDatasets(source ReadAllWriteAll, datasets *map[string]MockDataset) {
	var output strings.Builder
	idx := 0
	for _, d := range *datasets {
		output.WriteString(d.name)
		for key, value := range d.properties {
			output.WriteString("\t")
			output.WriteString(key)
			output.WriteString("\t")
			output.WriteString(value)
		}
		for key, value := range d.userProps {
			output.WriteString("\tuser:")
			output.WriteString(key)
			output.WriteString("\t")
			output.WriteString(value)
		}
		output.WriteString("\n")
		idx++
	}

	if idx > 0 {
		_ = source.WriteAll([]byte(output.String()))
	}
}

type typeCreateUpdateDatasetParams struct {
	datasetName string
	comments    string
	properties  map[string]string
	userProps   map[string]string
}

func getParamArray(params interface{}, minArgs int) ([]interface{}, error) {
	paramArray, ok := params.([]interface{})

	if !ok {
		return nil, errors.New("params must be supplied as an array")
	}

	if len(paramArray) < minArgs {
		formatted := fmt.Sprintf("params array must contain least %d entries", minArgs)
		return nil, errors.New(formatted)
	}

	return paramArray, nil
}

func getCreateUpdateDatasetParams(params interface{}) (typeCreateUpdateDatasetParams, error) {
	cdp := typeCreateUpdateDatasetParams{}

	paramArray, err := getParamArray(params, 1)
	if err != nil {
		return cdp, err
	}

	paramsMap, ok := paramArray[0].(map[string]interface{})
	if !ok {
		return cdp, errors.New("parameters for 'create/update' must be in the form of a map")
	}

	if value, ok := paramsMap["name"]; ok {
		if cdp.datasetName, ok = value.(string); !ok {
			return cdp, errors.New("name was not a string")
		}
		if strings.Index(cdp.datasetName, "/") <= 0 {
			return cdp, errors.New("dataset name must contain its pool name before a slash, eg. puddle/example")
		}
	}
	if value, ok := paramsMap["comments"]; ok {
		if cdp.comments, ok = value.(string); !ok {
			return cdp, errors.New("comments was not a string")
		}
	}
	if value, ok := paramsMap["properties"]; ok {
		var inMap map[string]interface{}
		if inMap, ok = value.(map[string]interface{}); !ok {
			return cdp, errors.New("properties was not a map/dictionary")
		}
		for key, pv := range inMap {
			pvStr, ok := pv.(string)
			if ok {
				pvStr = "\"" + pvStr + "\""
			} else {
				pvStr = fmt.Sprintf("%v", pv)
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
			return cdp, errors.New("user properties was not an array of map/dictionary")
		}
		for _, elem := range inMapList {
			inMap, ok := elem.(map[string]interface{})
			if !ok {
				return cdp, errors.New("user properties was not entirely an array of map/dictionary")
			}

			var uKeyStr string
			uKey, keyExists := inMap["key"]
			if !keyExists {
				return cdp, errors.New("user property did not contain a 'key'")
			}
			if uKeyStr, ok = uKey.(string); !ok {
				return cdp, errors.New("user property key was not a string")
			}

			var uValueStr string
			uValue, valueExists := inMap["value"]
			if !valueExists {
				return cdp, errors.New("user property '" + uKeyStr + "' did not contain a value")
			}
			if uValueStr, ok = uValue.(string); !ok {
				uValueStr = "\"" + uValueStr + "\""
			} else {
				uValueStr = fmt.Sprintf("%v", uValue)
			}

			if cdp.userProps == nil {
				cdp.userProps = make(map[string]string)
			}
			cdp.userProps[uKeyStr] = uValueStr
		}
	}
	return cdp, nil
}

func editDatasetProperties(cdp *typeCreateUpdateDatasetParams, dataset *MockDataset) ([]string, []string) {
	var propertyKeys []string
	nProps := len(cdp.properties)
	if nProps > 0 {
		propertyKeys = make([]string, 0, nProps)
		if dataset.properties == nil {
			dataset.properties = make(map[string]string)
		}
		for key, value := range cdp.properties {
			dataset.properties[key] = value
			propertyKeys = append(propertyKeys, key)
		}
		slices.Sort(propertyKeys)
	}

	var userPropKeys []string
	nUserProps := len(cdp.userProps)
	if nProps > 0 {
		userPropKeys = make([]string, 0, nUserProps)
		if dataset.userProps == nil {
			dataset.userProps = make(map[string]string)
		}
		for key, value := range cdp.userProps {
			dataset.userProps[key] = value
			userPropKeys = append(userPropKeys, key)
		}
		slices.Sort(userPropKeys)
	}

	return propertyKeys, userPropKeys
}

func (s *MockSession) mockDatasetCreate(params interface{}) (json.RawMessage, error) {
	cdp, err := getCreateUpdateDatasetParams(params)
	if err != nil {
		return nil, err
	}

	if cdp.datasetName == "" {
		return nil, errors.New("No dataset name was provided")
	}

	shouldCreateParents := false
	if parentsValue, ok := cdp.properties["create_ancestors"]; ok {
		shouldCreateParents = parentsValue == "true"
		delete(cdp.properties, "create_ancestors")
	}

	datasets := loadMockDatasets(s.DatasetSource)

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

	parts := strings.Split(cdp.datasetName, "/")
	for i := len(parts) - 2; i >= 1; i-- {
		// wasteful but easy
		parentName := strings.Join(parts[0:i+1], "/")
		if _, exists := datasets[parentName]; !exists {
			if shouldCreateParents {
				parent := MockDataset{}
				parent.name = parentName
				datasets[parentName] = parent
			} else {
				return nil, errors.New("Parent dataset \"" + parentName + "\" does not exist")
			}
		}
	}

	propertyKeys, userPropKeys := editDatasetProperties(&cdp, &newDataset)

	datasets[cdp.datasetName] = newDataset
	saveMockDatasets(s.DatasetSource, &datasets)

	var output strings.Builder
	writeDatasetInfo(&output, &newDataset, propertyKeys, userPropKeys)
	return []byte(output.String()), nil
}

func (s *MockSession) mockDatasetUpdate(params interface{}) (json.RawMessage, error) {
	udp, err := getCreateUpdateDatasetParams(params)
	if err != nil {
		return nil, err
	}

	if udp.datasetName == "" {
		return nil, errors.New("No dataset name was provided")
	}

	shouldUpdateParents := false
	if parentsValue, ok := udp.properties["create_ancestors"]; ok {
		shouldUpdateParents = parentsValue == "true"
		delete(udp.properties, "create_ancestors")
	}
	shouldUpdateParents = shouldUpdateParents

	datasets := loadMockDatasets(s.DatasetSource)
	if datasets == nil {
		return nil, errors.New("dataset does not exist")
	}
	dataset, exists := datasets[udp.datasetName]
	if !exists {
		return nil, errors.New("dataset does not exist")
	}

	propertyKeys, userPropKeys := editDatasetProperties(&udp, &dataset)
	datasets[udp.datasetName] = dataset
	saveMockDatasets(s.DatasetSource, &datasets)

	var output strings.Builder
	writeDatasetInfo(&output, &dataset, propertyKeys, userPropKeys)
	return []byte(output.String()), nil
}

func (s *MockSession) mockDatasetRename(params interface{}) (json.RawMessage, error) {
	paramsList, err := getParamArray(params, 2)
	if err != nil {
		return nil, err
	}

	var oldName string
	var newName string

	if value, ok := paramsList[0].(string); ok {
		oldName = value
	} else {
		return nil, errors.New("Dataset name (first param) was not a string")
	}

	if value, ok := paramsList[1].(map[string]interface{}); ok {
		if inner, ok := value["new_name"]; ok {
			if newName, ok = inner.(string); !ok {
				return nil, errors.New("New dataset name (second param) was not a string")
			}
		}
	} else if value, ok := paramsList[1].(string); ok {
		newName = value
	}

	if oldName == "" {
		return nil, errors.New("Dataset name (first param) was empty")
	}
	if newName == "" {
		return nil, errors.New("New dataset name (second param) was empty")
	}
	if newName == oldName {
		return nil, errors.New("New dataset name (second param) matches old dataset name (first param)")
	}

	datasets := loadMockDatasets(s.DatasetSource)
	if datasets == nil {
		return nil, errors.New("dataset does not exist")
	}
	dataset, exists := datasets[oldName]
	if !exists {
		return nil, errors.New("dataset does not exist")
	}

	delete(datasets, oldName)
	dataset.name = newName
	datasets[newName] = dataset

	saveMockDatasets(s.DatasetSource, &datasets)

	return []byte("True"), nil
}

func (s *MockSession) mockDatasetDelete(params interface{}) (json.RawMessage, error) {
	paramArray, err := getParamArray(params, 1)
	if err != nil {
		return nil, err
	}

	datasetName, ok := paramArray[0].(string)
	if !ok {
		return nil, errors.New("dataset delete requires a string, representing the name of the dataset to delete")
	}

	datasets := loadMockDatasets(s.DatasetSource)

	if datasets == nil {
		return nil, errors.New("dataset does not exist")
	}
	if _, exists := datasets[datasetName]; !exists {
		return nil, errors.New("dataset does not exist")
	}

	delete(datasets, datasetName)
	saveMockDatasets(s.DatasetSource, &datasets)

	return []byte("True"), nil
}

type typeQueryDatasetParams struct {
	datasetName       string
	properties        []string
	isFlat            bool
	withChildren      bool
	withUser          bool
	shouldGetAllProps bool
}

func getQueryDatasetParams(paramsList []interface{}) (typeQueryDatasetParams, error) {
	qdp := typeQueryDatasetParams{}
	cur := 0
	if cur >= len(paramsList) {
		return qdp, nil
	}
	if filterParamOuter, ok := paramsList[cur].([]interface{}); ok {
		// len == 0 is valid, and implies ALL datasets
		if len(filterParamOuter) > 0 {
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
		} else {
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
				for _, elem := range props {
					str := ""
					if str, ok = elem.(string); !ok {
						str = fmt.Sprint(elem)
					}
					qdp.properties = append(qdp.properties, str)
				}
			} else if value == nil {
				qdp.shouldGetAllProps = true
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
		isFirstProp := true
		for _, prop := range propertiesKeys {
			value, exists := dataset.properties[prop]
			if !exists {
				continue
			}
			if !isFirstProp {
				output.WriteString(",\n")
			}
			output.WriteString("\"")
			output.WriteString(prop)
			output.WriteString("\":{\"value\":")
			output.WriteString(value)
			output.WriteString(", \"rawvalue\":")
			output.WriteString(value)
			output.WriteString(", \"source\":\"LOCAL\", \"parsed\":")
			output.WriteString(value)
			output.WriteString("}")
			isFirstProp = false
		}
		output.WriteString("},\n")
	}
	output.WriteString("\"comments\":{\"value\":\"\", \"rawvalue\":\"\", \"source\":\"LOCAL\", \"parsed\":\"\"}, \"user_properties\":{")
	if userPropsKeys != nil {
		isFirstProp := true
		for _, prop := range userPropsKeys {
			value, exists := dataset.userProps[prop]
			if !exists {
				continue
			}
			if !isFirstProp {
				output.WriteString(",\n")
			}
			output.WriteString("\"")
			output.WriteString(prop)
			output.WriteString("\":{\"value\":")
			output.WriteString(value)
			output.WriteString(", \"rawvalue\":\"")
			output.WriteString(value)
			output.WriteString(", \"source\":\"LOCAL\", \"parsed\":")
			output.WriteString(value)
			output.WriteString("}")
			isFirstProp = false
		}
	}
	output.WriteString("} }")
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

	datasets := loadMockDatasets(s.DatasetSource)

	var output strings.Builder
	output.WriteString("{\"jsonrpc\":\"2.0\", \"result\":[")

	if datasets != nil {
		if qdp.datasetName != "" {
			if dataset, exists := datasets[qdp.datasetName]; exists {
				var properties []string
				if qdp.shouldGetAllProps {
					properties = GetKeysSorted(dataset.properties)
				} else {
					properties = qdp.properties
				}
				writeDatasetInfo(&output, &dataset, properties, nil)
			}
		} else {
			datasetNames := GetKeysSorted(datasets)
			for idx, name := range datasetNames {
				dataset := datasets[name]
				if idx > 0 {
					output.WriteString(", ")
				}
				var properties []string
				if qdp.shouldGetAllProps {
					properties = GetKeysSorted(dataset.properties)
				} else {
					properties = qdp.properties
				}
				writeDatasetInfo(&output, &dataset, properties, nil)
				idx++
			}
		}
	}

	output.WriteString("], \"id\":2}")
	outStr := output.String()
	return []byte(outStr), nil
}

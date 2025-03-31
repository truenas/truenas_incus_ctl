package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"
)

type typeQueryParams struct {
	valueOrder         []string
	shouldSkipKeyBuild bool
	shouldGetAllProps  bool
	shouldGetUserProps bool
	shouldRecurse      bool
}

type typeQueryResponse struct {
	resultsMap map[string]map[string]interface{}
	intKeys []int
	strKeys []string
}

func BuildNameStrAndPropertiesJson(options FlagMap, nameStr string) []interface{} {
	outMap := make(map[string]interface{})
	for key, value := range options.usedFlags {
		parsed, _ := ParseStringAndValidate(key, value, nil)
		outMap[key] = parsed
	}

	return []interface{}{nameStr, outMap}
}

func QueryApi(api core.Session, category string, entries, entryTypes, propsList []string, params typeQueryParams) (typeQueryResponse, error) {
	response := typeQueryResponse{}
	endpoint := category + ".query"
	isNfs := endpoint == "sharing.nfs.query"

	if len(entryTypes) != len(entries) {
		return response, fmt.Errorf("length mismatch between entries and entry types: %d != %d", len(entries), len(entryTypes))
	}

	filter, err := makeQueryFilter(entries, entryTypes, params)
	if err != nil {
		return response, err
	}

	query := []interface{}{filter}
	if !isNfs {
		query = append(query, makeQueryOptions(propsList, params, strings.Contains(endpoint, "snapshot")))
	}

	DebugJson(query)

	data, err := core.ApiCall(api, endpoint, 20, query)
	if err != nil {
		return response, err
	}

	//os.Stdout.WriteString(string(data))
	//fmt.Println("\n")

	var jsonResponse interface{}
	if err = json.Unmarshal(data, &jsonResponse); err != nil {
		return response, fmt.Errorf("response error: %v", err)
	}

	responseMap, ok := jsonResponse.(map[string]interface{})
	if !ok {
		return response, errors.New("API response was not a JSON object")
	}

	resultsList, errMsg := core.ExtractJsonArrayOfMaps(responseMap, "result")
	if errMsg != "" {
		return response, errors.New("API response results: " + errMsg)
	}
	if len(resultsList) == 0 {
		DebugString("resultsList was empty")
		return response, nil
	}

	outputMap := make(map[string]map[string]interface{})
	outputMapIntKeys := make([]int, 0, 0)
	outputMapStrKeys := make([]string, 0, 0)

	// Do not refactor this loop condition into a range!
	// This loop modifies the size of resultsList as it iterates.
	for i := 0; i < len(resultsList); i++ {
		children, _ := core.ExtractJsonArrayOfMaps(resultsList[i], "children")
		if len(children) > 0 {
			resultsList = append(resultsList, children...)
		}

		var primary string
		var primaryValue interface{}
		if primaryValue, ok = resultsList[i]["id"]; ok {
			if primaryStr, ok := primaryValue.(string); ok {
				primary = primaryStr
			} else {
				primary = fmt.Sprint(primaryValue)
			}
		}
		if len(primary) == 0 {
			continue
		}
		if _, exists := outputMap[primary]; exists {
			continue
		}

		dict := make(map[string]interface{})
		dict["id"] = primaryValue

		if isNfs {
			dict["type"] = "NFS"
		}

		insertProperties(dict, resultsList[i], []string{"id", "children", "properties"}, params.valueOrder)
		if innerProps, exists := resultsList[i]["properties"]; exists {
			if innerPropsMap, ok := innerProps.(map[string]interface{}); ok {
				insertProperties(dict, innerPropsMap, nil, params.valueOrder)
			}
		}
		if innerProps, exists := resultsList[i]["user_properties"]; exists {
			if innerPropsMap, ok := innerProps.(map[string]interface{}); ok {
				insertProperties(dict, innerPropsMap, nil, params.valueOrder)
			}
		}

		outputMap[primary] = dict
		if !params.shouldSkipKeyBuild {
			if primaryInt, errNotNumber := strconv.Atoi(primary); errNotNumber == nil {
				outputMapIntKeys = append(outputMapIntKeys, primaryInt)
			} else {
				outputMapStrKeys = append(outputMapStrKeys, primary)
			}
		}
	}

	response = typeQueryResponse {
		resultsMap: outputMap,
		intKeys: outputMapIntKeys,
		strKeys: outputMapStrKeys,
	}
	return response, nil
}

func GetListFromQueryResponse(response *typeQueryResponse) []map[string]interface{} {
	if response == nil {
		return nil
	}

	slices.Sort(response.intKeys)

	slices.SortStableFunc(response.strKeys, func(a, b string) int {
		atPosA := strings.Index(a, "@")
		if atPosA < 0 {
			return strings.Compare(a, b)
		}
		atPosB := strings.Index(b, "@")
		if atPosB < 0 {
			return strings.Compare(a, b)
		}
		if atPosA != atPosB {
			return strings.Compare(a, b)
		}
		nameCompare := strings.Compare(a[0:atPosA], b[0:atPosB])
		if nameCompare != 0 {
			return nameCompare
		}
		var txgA int64
		if res, exists := response.resultsMap[a]; exists {
			txgA = core.GetIntegerFromJsonObjectOr(res, "createtxg", 0)
		}
		if txgA == 0 {
			return 0
		}
		var txgB int64
		if res, exists := response.resultsMap[b]; exists {
			txgB = core.GetIntegerFromJsonObjectOr(res, "createtxg", 0)
		}
		if txgB == 0 || txgA == txgB {
			return 0
		}
		if txgA < txgB {
			return -1
		}
		return 1
	})

	nKeys := len(response.intKeys) + len(response.strKeys)
	resultsList := make([]map[string]interface{}, nKeys, nKeys)

	for i, _ := range response.intKeys {
		resultsList[i] = response.resultsMap[strconv.Itoa(response.intKeys[i])]
	}
	for i, _ := range response.strKeys {
		resultsList[len(response.intKeys)+i] = response.resultsMap[response.strKeys[i]]
	}

	return resultsList
}

func makeQueryFilter(entries, entryTypes []string, params typeQueryParams) ([]interface{}, error) {
	for i, e := range entries {
		if e == "" {
			return nil, fmt.Errorf("Cannot query based on empty %s", entryTypes[i])
		}
	}

	filter := make([]interface{}, 0)

	// first arg = query-filter
	if len(entries) == 1 {
		filter = append(filter, makeIndividualFilter(entryTypes[0], []string{entries[0]}, params.shouldRecurse))
	} else if len(entries) > 1 {
		typeEntriesMap := make(map[string][]string)
		uniqTypes := make([]string, 0, 0)
		for i := 0; i < len(entries); i++ {
			if _, exists := typeEntriesMap[entryTypes[i]]; !exists {
				typeEntriesMap[entryTypes[i]] = make([]string, 0, 0)
				uniqTypes = append(uniqTypes, entryTypes[i])
			}
			typeEntriesMap[entryTypes[i]] = append(typeEntriesMap[entryTypes[i]], entries[i])
		}

		filterList := make([][]interface{}, len(uniqTypes))
		for i := 0; i < len(uniqTypes); i++ {
			filterList[i] = makeIndividualFilter(uniqTypes[i], typeEntriesMap[uniqTypes[i]], params.shouldRecurse)
		}

		filter = append(filter, constructORChain(filterList))
	}

	return filter, nil
}

func makeIndividualFilter(key string, array []string, isRecursive bool) []interface{} {
	if isRecursive && (key == "dataset" /* || key == "pool"*/) {
		return constructORChain(makeRecursivePathsFilterList(key, array))
	}
	arr := make([]interface{}, len(array), len(array))
	for i := 0; i < len(array); i++ {
		if n, errNotNumber := strconv.Atoi(array[i]); errNotNumber == nil {
			arr[i] = n
		} else {
			arr[i] = array[i]
		}
	}
	return []interface{}{key, "in", arr}
}

func makeRecursivePathsFilterList(key string, paths []string) [][]interface{} {
	filterList := make([][]interface{}, 0)
	for i := 0; i < len(paths); i++ {
		filterList = append(filterList, []interface{}{key, "=", paths[i]})
		filterList = append(filterList, []interface{}{key, "^", paths[i] + "/"})
	}
	return filterList
}

func constructORChain(filterList [][]interface{}) []interface{} {
	nFilters := len(filterList)
	if nFilters == 0 {
		return nil
	}
	top := [][]interface{}{filterList[0]}
	for i := 1; i < nFilters; i++ {
		top = append(top, filterList[i])
		inner := []interface{}{"OR", top}
		top = [][]interface{}{inner}
	}
	return top[0]
}

func makeQueryOptions(propsList []string, params typeQueryParams, isSnapshot bool) map[string]interface{} {
	// second arg = query-options
	options := make(map[string]interface{})
	options["flat"] = false
	options["retrieve_children"] = params.shouldRecurse
	if params.shouldGetAllProps {
		var nothing interface{}
		options["properties"] = nothing
	} else {
		if propsList == nil {
			propsList = make([]string, 0)
		}
		if isSnapshot {
			propsList = core.AppendIfMissing(propsList, "createtxg")
		}
		options["properties"] = propsList
	}
	options["user_properties"] = params.shouldGetUserProps
	return map[string]interface{}{"extra": options}
}

func insertProperties(dstMap, srcMap map[string]interface{}, excludeKeys []string, valueOrder []string) {
	for key, value := range srcMap {
		if _, exists := dstMap[key]; exists {
			continue
		}
		shouldSkip := false
		for _, ex := range excludeKeys {
			if key == ex {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		DebugJson(value)

		var elem interface{}
		if valueMap, ok := value.(map[string]interface{}); ok {
			for _, t := range valueOrder {
				if actualValue, ok := valueMap[t]; ok {
					elem = actualValue
					break
				}
			}

		} else {
			elem = value
		}

		if elem != nil {
			if elemFloat, ok := elem.(float64); ok {
				if elemFloat == math.Floor(elemFloat) {
					elem = int64(elemFloat)
				}
			}
			dstMap[key] = elem
		}
	}
}

func BuildValueOrder(parsed bool) []string {
	if parsed {
		return []string{"parsed", "value", "rawvalue"}
	}
	return []string{"value", "rawvalue", "parsed"}
}

func LowerCaseValuesFromEnums(results []map[string]interface{}, enums map[string][]string) {
	for i, _ := range results {
		for key, _ := range enums {
			if value, exists := results[i][key]; exists {
				if valueStr, ok := value.(string); ok {
					results[i][key] = strings.ToLower(valueStr)
				}
			}
		}
	}
}

func LookupNfsIdByPath(api core.Session, sharePath string, optShareProperties map[string]string) (string, bool, error) {
	if sharePath == "" {
		return "", false, errors.New("Error looking up NFS share: no path was specified")
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(false),
		shouldGetAllProps:  optShareProperties != nil,
		shouldGetUserProps: optShareProperties != nil,
		shouldRecurse:      false,
	}

	response, err := QueryApi(api, "sharing.nfs", []string{sharePath}, []string{"path"}, []string{"id", "path"}, extras)
	if err != nil {
		return "", false, errors.New("API error: " + fmt.Sprint(err))
	}

	shares := GetListFromQueryResponse(&response)
	if len(shares) == 0 {
		return "", false, nil
	}

	var idStr string
	if value, exists := shares[0]["id"]; exists {
		if valueStr, ok := value.(string); ok {
			if _, errNotNumber := strconv.Atoi(valueStr); errNotNumber == nil {
				idStr = valueStr
			}
		} else {
			idStr = fmt.Sprint(value)
		}
	}
	if idStr == "" {
		return "", false, nil
	}

	if optShareProperties != nil {
		for key, value := range shares[0] {
			if valueStr, ok := value.(string); ok {
				optShareProperties[key] = valueStr
			} else {
				optShareProperties[key] = fmt.Sprint(value)
			}
		}
	}

	return idStr, true, nil
}

func ConvertParamsStringToKvArray(fullParamsStr string) []string {
	if fullParamsStr == "" {
		return nil
	}

	kvArray := make([]string, 0)
	params := strings.Split(fullParamsStr, ",")
	for _, parameter := range params {
		parts := strings.Split(parameter, "=")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		var value string
		if len(parts) == 1 || parts[1] == "" {
			value = "true"
		} else {
			value = parts[1]
		}
		kvArray = append(kvArray, parts[0], value)
	}

	return kvArray
}

func WriteKvArrayToMap(dstMap map[string]interface{}, kvArray []string, enumsList map[string][]string) error {
	for i := 0; i < len(kvArray); i += 2 {
		key := kvArray[i]
		value, err := ParseStringAndValidate(key, kvArray[i+1], enumsList)
		if err != nil {
			return err
		}
		dstMap[key] = value
	}
	return nil
}

func ParseStringAndValidate(optKey, value string, optEnumsList map[string][]string) (interface{}, error) {
	if value == "true" || value == "false" {
		return value == "true", nil
	} else if value == "null" {
		return nil, nil
	} else if intValue, errNotInteger := strconv.Atoi(value); errNotInteger == nil {
		return intValue, nil
	} else if floatValue, errNotFloat := strconv.ParseFloat(value, 64); errNotFloat == nil {
		return floatValue, nil
	} else {
		if optKey != "" && optEnumsList != nil {
			if acceptable, exists := optEnumsList[optKey]; exists {
				found := false
				valueUpper := strings.ToUpper(value)
				for i := 0; i < len(acceptable); i++ {
					if valueUpper == strings.ToUpper(acceptable[i]) {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("Could not find value %s in enum %s %v", value, optKey, acceptable)
				}
				return valueUpper, nil
			}
		}
	}
	return value, nil
}

func MaybeCopyProperty(dstMap map[string]interface{}, srcMap map[string]string, key string) {
	if valueStr, exists := srcMap[key]; exists {
		dstMap[key], _ = ParseStringAndValidate(key, valueStr, nil)
	}
}

func EnumerateOutputProperties(properties map[string]string) []string {
	propsStr, exists := properties["output"]
	if !exists || propsStr == "" {
		return nil
	}

	var propsList []string
	if len(propsStr) > 0 {
		propsList = strings.Split(propsStr, ",")
	}
	return propsList
}

func MakePropertyColumns(required []string, additional []string) []string {
	columnSet := make(map[string]bool)
	uniqAdditional := make([]string, 0, 0)

	for _, c := range required {
		columnSet[c] = true
	}
	for _, c := range additional {
		if _, exists := columnSet[c]; !exists {
			uniqAdditional = append(uniqAdditional, c)
		}
		columnSet[c] = true
	}

	slices.Sort(uniqAdditional)

	if len(required) > 0 {
		return append(required, uniqAdditional...)
	}
	return uniqAdditional
}

func GetUsedPropertyColumns[T any](data []map[string]T, required []string) []string {
	columnsMap := make(map[string]bool)
	columnsList := make([]string, 0)

	for _, c := range required {
		columnsMap[c] = true
	}

	for _, d := range data {
		for key, _ := range d {
			if _, exists := columnsMap[key]; !exists {
				columnsMap[key] = true
				columnsList = append(columnsList, key)
			}
		}
	}

	slices.Sort(columnsList)
	return append(required, columnsList...)
}

func GetTableFormat(properties map[string]string) (string, error) {
	isJson := core.IsValueTrue(properties, "json")
	isCompact := core.IsValueTrue(properties, "no_headers")
	if isJson && isCompact {
		return "", errors.New("--json and --no_headers cannot be used together")
	} else if isJson {
		return "json", nil
	} else if isCompact {
		return "compact", nil
	}

	return properties["format"], nil
}

func MaybeBulkApiCall(api core.Session, endpoint string, timeoutSeconds int64, params interface{}, remapList map[string][]interface{}, shouldWaitNow bool) (json.RawMessage, error) {
	allParams := make([][]interface{}, 0)
	for key, valueList := range remapList {
		for i, value := range valueList {
			if len(allParams) <= i {
				allParams = append(allParams, core.DeepCopy(params).([]interface{}))
			}
			_, isObjFirst := allParams[i][0].(map[string]interface{})
			if key == "" {
				if isObjFirst {
					allParams[i] = append([]interface{}{value}, allParams[i]...)
				} else {
					allParams[i][0] = value
				}
			} else {
				objIdx := 1
				if isObjFirst {
					objIdx = 0
				}
				allParams[i][objIdx].(map[string]interface{})[key] = value
			}
		}
	}

	nParams := len(allParams)
	if nParams == 0 {
		return nil, errors.New("MaybeBulkApiCall: Nothing to do")
	} else if nParams == 1 {
		DebugJson(allParams[0])
		return core.ApiCall(api, endpoint, timeoutSeconds, allParams[0])
	}

	methodAndParams := make([]interface{}, 0)
	methodAndParams = append(methodAndParams, endpoint)
	methodAndParams = append(methodAndParams, allParams)

	DebugJson(methodAndParams)
	jobId, err := core.ApiCallAsync(api, "core.bulk", methodAndParams, shouldWaitNow)
	if !shouldWaitNow || err != nil || jobId < 0 {
		return nil, err
	}

	return api.WaitForJob(jobId)
}

func MaybeBulkApiCallArray(api core.Session, endpoint string, timeoutSeconds int64, paramsArray []interface{}, shouldWaitNow bool) (json.RawMessage, error) {
	nCalls := len(paramsArray)
	if nCalls == 0 {
		return nil, errors.New("MaybeBulkApiCallArray: Nothing to do")
	}
	if nCalls == 1 {
		DebugJson(paramsArray[0])
		return core.ApiCall(api, endpoint, timeoutSeconds, paramsArray[0])
	}

	methodAndParams := make([]interface{}, 0)
	methodAndParams = append(methodAndParams, endpoint)
	methodAndParams = append(methodAndParams, paramsArray)

	DebugJson(methodAndParams)
	jobId, err := core.ApiCallAsync(api, "core.bulk", methodAndParams, shouldWaitNow)
	if !shouldWaitNow || err != nil || jobId < 0 {
		return nil, err
	}

	return api.WaitForJob(jobId)
}

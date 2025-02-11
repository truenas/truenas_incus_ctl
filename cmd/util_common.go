package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	//"os"
	//"log"
	"slices"
	"strconv"
	"strings"
	"truenas/admin-tool/core"
)

type typeRetrieveParams struct {
	valueOrder         []string
	shouldGetAllProps  bool
	shouldGetUserProps bool
	shouldRecurse      bool
}

func BuildNameStrAndPropertiesJson(options FlagMap, nameStr string) []interface{} {
	outMap := make(map[string]interface{})
	for key, value := range options.usedFlags {
		parsed, _ := ParseStringAndValidate(key, value, nil)
		outMap[key] = parsed
	}

	return []interface{} {nameStr, outMap}
}

func QueryApi(api core.Session, endpointType string, entries, entryTypes, propsList []string, params typeRetrieveParams) ([]map[string]interface{}, error) {
	var endpoint string
	switch endpointType {
	case "dataset":
		endpoint = "pool.dataset.query"
	case "snapshot":
		endpoint = "zfs.snapshot.query"
	case "nfs":
		endpoint = "sharing.nfs.query"
	default:
		return nil, fmt.Errorf("Unrecognised retrieve format \"" + endpointType + "\"")
	}

	if len(entryTypes) != len(entries) {
		return nil, errors.New(fmt.Sprint("Length mismatch between entries and entry types:", len(entries), "!=", len(entryTypes)))
	}

	query := []interface{} {makeQueryFilter(entries, entryTypes, params)}
	if endpointType != "nfs" {
		query = append(query, makeQueryOptions(propsList, params))
	}

	DebugJson(query)

	data, err := core.ApiCall(api, endpoint, "20s", query)
	if err != nil {
		return nil, err
	}

	//os.Stdout.WriteString(string(data))
	//fmt.Println("\n")

	var response interface{}
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("response error: %v", err)
	}

	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, errors.New("API response was not a JSON object")
	}

	resultsList, errMsg := core.ExtractJsonArrayOfMaps(responseMap, "result")
	if errMsg != "" {
		return nil, errors.New("API response results: " + errMsg)
	}
	if len(resultsList) == 0 {
		DebugString("resultsList was empty")
		return nil, nil
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
		if primaryValue, ok := resultsList[i]["id"]; ok {
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
		dict["id"] = primary

		if endpointType == "nfs" {
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
		if primaryInt, errNotNumber := strconv.Atoi(primary); errNotNumber == nil {
			outputMapIntKeys = append(outputMapIntKeys, primaryInt)
		} else {
			outputMapStrKeys = append(outputMapStrKeys, primary)
		}
	}

	slices.Sort(outputMapIntKeys)
	slices.Sort(outputMapStrKeys)
	nKeys := len(outputMapIntKeys) + len(outputMapStrKeys)

	outputList := make([]map[string]interface{}, nKeys, nKeys)
	for i, _ := range outputMapIntKeys {
		outputList[i] = outputMap[strconv.Itoa(outputMapIntKeys[i])]
	}
	for i, _ := range outputMapStrKeys {
		outputList[len(outputMapIntKeys) + i] = outputMap[outputMapStrKeys[i]]
	}

	return outputList, nil
}

func makeQueryFilter(entries, entryTypes []string, params typeRetrieveParams) []interface{} {
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

	return filter
}

func makeIndividualFilter(key string, array []string, isRecursive bool) []interface{} {
	if isRecursive && (key == "dataset" /* || key == "pool"*/) {
		return constructORChain(makeRecursivePathsFilterList(key, array))
	}
	return []interface{} {key, "in", array}
}

func makeRecursivePathsFilterList(key string, paths []string) [][]interface{} {
	filterList := make([][]interface{}, 0)
	for i := 0; i < len(paths); i++ {
		filterList = append(filterList, []interface{} {key, "=", paths[i]})
		filterList = append(filterList, []interface{} {key, "^", paths[i] + "/"})
	}
	return filterList
}

func constructORChain(filterList [][]interface{}) []interface{} {
	nFilters := len(filterList)
	if nFilters == 0 {
		return nil
	}
	top := [][]interface{} {filterList[0]}
	for i := 1; i < nFilters; i++ {
		top = append(top, filterList[i])
		inner := []interface{} {"OR",top}
		top = [][]interface{} {inner}
	}
	return top[0]
}

func makeQueryOptions(propsList []string, params typeRetrieveParams) map[string]interface{} {
	// second arg = query-options
	options := make(map[string]interface{})
	options["flat"] = false
	options["retrieve_children"] = params.shouldRecurse
	if params.shouldGetAllProps {
		var nothing interface{}
		options["properties"] = nothing
	} else {
		options["properties"] = propsList
	}
	options["user_properties"] = params.shouldGetUserProps
	return map[string]interface{} {"extra": options}
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

func BuildValueOrder(parseable bool) []string {
	if parseable {
		return []string{"value", "rawvalue", "parsed"}
	}
	return []string{"parsed", "value", "rawvalue"}
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

	extras := typeRetrieveParams{
		valueOrder:         BuildValueOrder(false),
		shouldGetAllProps:  optShareProperties != nil,
		shouldGetUserProps: optShareProperties != nil,
		shouldRecurse:      false,
	}

	shares, err := QueryApi(api, "nfs", []string{sharePath}, []string{"path"}, []string{"id", "path"}, extras)
	if err != nil {
		return "", false, errors.New("API error: " + fmt.Sprint(err))
	}
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

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
	retrieveType       string
	shouldGetAllProps  bool
	shouldGetUserProps bool
	shouldRecurse      bool
}

func BuildNameStrAndPropertiesJson(options FlagMap, nameStr string) string {
	var builder strings.Builder
	builder.WriteString("[")
	core.WriteEncloseAndEscape(&builder, nameStr, "\"")
	builder.WriteString(",{")

	nProps := 0
	for key, value := range options.usedFlags {
		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(&builder, key, "\"")
		builder.WriteString(":")
		if t, _ := options.allTypes[key]; t == "string" {
			core.WriteEncloseAndEscape(&builder, value, "\"")
		} else {
			builder.WriteString(value)
		}
		nProps++
	}

	builder.WriteString("}]")
	return builder.String()
}

func QueryApi(api core.Session, entries, entryTypes, propsList []string, params typeRetrieveParams) ([]map[string]interface{}, error) {
	var endpoint string
	switch params.retrieveType {
	case "dataset":
		endpoint = "pool.dataset.query"
	case "snapshot":
		endpoint = "zfs.snapshot.query"
	case "nfs":
		endpoint = "sharing.nfs.query"
	default:
		return nil, fmt.Errorf("Unrecognised retrieve format \"" + params.retrieveType + "\"")
	}

	if len(entryTypes) != len(entries) {
		return nil, fmt.Errorf("Length mismatch between entries and entry types:", len(entries), "!=", len(entryTypes))
	}

	var builder strings.Builder
	builder.WriteString("[")

	writeQueryFilter(&builder, entries, entryTypes, params)
	if params.retrieveType != "nfs" {
		builder.WriteString(", ")
		writeQueryOptions(&builder, propsList, params)
	}

	builder.WriteString("]")

	query := builder.String()
	DebugString(query)

	data, err := core.ApiCallString(api, endpoint, "20s", query)
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

		insertProperties(dict, resultsList[i], []string{"id", "children", "properties"})
		if innerProps, exists := resultsList[i]["properties"]; exists {
			if innerPropsMap, ok := innerProps.(map[string]interface{}); ok {
				insertProperties(dict, innerPropsMap, nil)
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

func writeQueryFilter(builder *strings.Builder, entries, entryTypes []string, params typeRetrieveParams) {
	builder.WriteString("[")

	// first arg = query-filter
	if len(entries) == 1 {
		writeIndividualFilter(builder, entryTypes[0], []string{entries[0]}, params.shouldRecurse)
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

		for i := 0; i < len(uniqTypes) - 1; i++ {
			builder.WriteString("[\"OR\",[")
		}

		writeIndividualFilter(builder, uniqTypes[0], typeEntriesMap[uniqTypes[0]], params.shouldRecurse)
		for i := 1; i < len(uniqTypes); i++ {
			builder.WriteString(",")
			writeIndividualFilter(builder, uniqTypes[i], typeEntriesMap[uniqTypes[i]], params.shouldRecurse)
			builder.WriteString("]]")
		}
	}
	builder.WriteString("]")
}

func writeIndividualFilter(builder *strings.Builder, key string, array []string, isRecursive bool) {
	if isRecursive && (key == "dataset" /* || key == "pool"*/) {
		writeKeyAndRecursivePaths(builder, key, array)
	} else {
		writeKeyAndArray(builder, key, array)
	}
}

func writeKeyAndArray(builder *strings.Builder, key string, array []string) {
	builder.WriteString("[")
	core.WriteEncloseAndEscape(builder, key, "\"")
	builder.WriteString(",\"in\",[")
	for j, elem := range array {
		if j > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(builder, elem, "\"")
	}
	builder.WriteString("]]")
}

func writeKeyAndRecursivePaths(builder *strings.Builder, key string, array []string) {
	nFilters := len(array) * 2
	for i := 0; i < nFilters - 1; i++ {
		builder.WriteString("[\"OR\",[")
	}

	writeRecursivePathFilter(builder, key, array[0], false)
	for i := 1; i < nFilters; i++ {
		builder.WriteString(",")
		writeRecursivePathFilter(builder, key, array[i / 2], (i % 2) == 1)
		builder.WriteString("]]")
	}
}

func writeRecursivePathFilter(builder *strings.Builder, key, path string, isStartsWith bool) {
	builder.WriteString("[")
	core.WriteEncloseAndEscape(builder, key, "\"")
	if isStartsWith {
		builder.WriteString(",\"^\",")
		core.WriteEncloseAndEscape(builder, path, "\"")
	} else {
		builder.WriteString(",\"=\",")
		core.WriteEncloseAndEscape(builder, path + "/", "\"")
	}
	builder.WriteString("]")
}

func writeQueryOptions(builder *strings.Builder, propsList []string, params typeRetrieveParams) {
	// second arg = query-options
	builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":")
	builder.WriteString(fmt.Sprint(params.shouldRecurse))
	builder.WriteString(", \"properties\":")
	if params.shouldGetAllProps {
		builder.WriteString("null")
	} else {
		builder.WriteString("[")
		if len(propsList) > 0 {
			core.WriteEncloseAndEscape(builder, propsList[0], "\"")
			for i := 1; i < len(propsList); i++ {
				builder.WriteString(",")
				core.WriteEncloseAndEscape(builder, propsList[i], "\"")
			}
		}
		builder.WriteString("]")
	}
	builder.WriteString(", \"user_properties\":")
	builder.WriteString(fmt.Sprint(params.shouldGetUserProps))
	builder.WriteString(" }} ")
}

func insertProperties(dstMap, srcMap map[string]interface{}, excludeKeys []string) {
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
			if actualValue, ok := valueMap["parsed"]; ok {
				elem = actualValue
			} else if actualValue, ok := valueMap["value"]; ok {
				elem = actualValue
			} else if actualValue, ok := valueMap["rawvalue"]; ok {
				elem = actualValue
			} else {
				continue
			}
		} else {
			elem = value
		}

		if elemFloat, ok := elem.(float64); ok {
			if elemFloat == math.Floor(elemFloat) {
				elem = int64(elemFloat)
			}
		}
		dstMap[key] = elem
	}
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

func LookupNfsIdByPath(api core.Session, sharePath string) (string, bool, error) {
	if sharePath == "" {
		return "", false, errors.New("Error looking up NFS share: no path was specified")
	}

	extras := typeRetrieveParams{
		retrieveType:       "nfs",
		shouldGetAllProps:  false,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}

	shares, err := QueryApi(api, []string{sharePath}, []string{"path"}, []string{"id", "path"}, extras)
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

	return idStr, true, nil
}

func EnumerateOutputProperties(properties map[string]string) []string {
	propsStr, exists := properties["output"]
	if !exists || propsStr == "" {
		return nil
	}

	var propsList []string
	if len(propsStr) > 0 {
		propsList = strings.Split(propsStr, ",")
		/*
			for j := 0; j < len(propsList); j++ {
				propsList[j] = strings.Trim(propsList[j], " \t\r\n")
			}
		*/
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

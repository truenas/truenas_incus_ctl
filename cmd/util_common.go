package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	//"os"
	//"log"
	"slices"
	"strings"
	"truenas/admin-tool/core"
)

type typeRetrieveParams struct {
	retrieveType      string
	primaryColumn     string
	shouldGetAllProps bool
	shouldRecurse     bool
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

func RetrieveDatasetOrSnapshotInfos(api core.Session, entries []string, propsList []string, params typeRetrieveParams) ([]map[string]interface{}, error) {
	var endpoint string
	switch params.retrieveType {
	case "dataset":
		endpoint = "pool.dataset.query"
	case "snapshot":
		endpoint = "zfs.snapshot.query"
	default:
		return nil, fmt.Errorf("Unrecognised retrieve format \"" + params.retrieveType + "\"")
	}

	if params.primaryColumn == "" {
		return nil, fmt.Errorf("Error querying " + params.retrieveType + "s: primary column was not set")
	}

	var builder strings.Builder
	builder.WriteString("[[ ")

	// first arg = query-filter
	if len(entries) > 0 {
		builder.WriteString("[")
		core.WriteEncloseAndEscape(&builder, params.primaryColumn, "\"")
		if len(entries) == 1 {
			builder.WriteString(", \"=\", ")
			core.WriteEncloseAndEscape(&builder, entries[0], "\"")
		} else {
			builder.WriteString(", \"in\", ")
			builder.WriteString("[")
			for i, elem := range entries {
				if i > 0 {
					builder.WriteString(",")
				}
				core.WriteEncloseAndEscape(&builder, elem, "\"")
			}
			builder.WriteString("]")
		}
		builder.WriteString("]")
	}
	builder.WriteString("], ") // end first arg

	// second arg = query-options
	builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":")
	builder.WriteString(fmt.Sprint(params.shouldRecurse))
	builder.WriteString(", \"properties\":")
	if params.shouldGetAllProps {
		builder.WriteString("null")
	} else {
		builder.WriteString("[")
		if len(propsList) > 0 {
			core.WriteEncloseAndEscape(&builder, propsList[0], "\"")
			for i := 1; i < len(propsList); i++ {
				builder.WriteString(",")
				core.WriteEncloseAndEscape(&builder, propsList[i], "\"")
			}
		}
		builder.WriteString("]")
	}
	builder.WriteString(", \"user_properties\":false }} ]")

	query := builder.String()
	DebugString(query)

	data, err := api.CallString(endpoint, "20s", query)
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
		return nil, nil
	}

	outputMap := make(map[string]map[string]interface{})
	outputMapKeys := make([]string, 0, 0)

	// Do not refactor this loop condition into a range!
	// This loop modifies the size of resultsList as it iterates.
	for i := 0; i < len(resultsList); i++ {
		children, _ := core.ExtractJsonArrayOfMaps(resultsList[i], "children")
		if len(children) > 0 {
			resultsList = append(append(resultsList[0:i+1], children...), resultsList[i+1:]...)
		}

		var primary string
		if primaryValue, ok := resultsList[i][params.primaryColumn]; ok {
			if primaryStr, ok := primaryValue.(string); ok {
				primary = primaryStr
			}
		}
		if len(primary) == 0 {
			continue
		}
		if _, exists := outputMap[primary]; exists {
			continue
		}

		dict := make(map[string]interface{})
		dict[params.primaryColumn] = primary

		insertProperties(dict, resultsList[i], []string{params.primaryColumn, "children", "properties"})
		if innerProps, exists := resultsList[i]["properties"]; exists {
			if innerPropsMap, ok := innerProps.(map[string]interface{}); ok {
				insertProperties(dict, innerPropsMap, []string{params.primaryColumn})
			}
		}

		outputMap[primary] = dict
		outputMapKeys = append(outputMapKeys, primary)
	}

	slices.Sort(outputMapKeys)
	nKeys := len(outputMapKeys)

	outputList := make([]map[string]interface{}, nKeys, nKeys)
	for i, _ := range outputMapKeys {
		outputList[i] = outputMap[outputMapKeys[i]]
	}

	return outputList, nil
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
		if valueMap, ok := value.(map[string]interface{}); ok {
			if actualValue, ok := valueMap["parsed"]; ok {
				dstMap[key] = actualValue
			} else if actualValue, ok := valueMap["value"]; ok {
				dstMap[key] = actualValue
			}
		} else {
			dstMap[key] = value
		}
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

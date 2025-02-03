package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	//"log"
	"slices"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

type typeRetrieveParams struct {
	retrieveType      string
	shouldGetAllProps bool
	shouldRecurse     bool
}

func BuildNameStrAndPropertiesJson(cmd *cobra.Command, nameStr string) string {
	var builder strings.Builder
	builder.WriteString("[")
	core.WriteEncloseAndEscape(&builder, nameStr, "\"")
	builder.WriteString(",{")

	usedOptions, _, allTypes := getCobraFlags(cmd)
	nProps := 0
	for key, value := range usedOptions {
		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(&builder, key, "\"")
		builder.WriteString(":")
		if t, _ := allTypes[key]; t == "string" {
			core.WriteEncloseAndEscape(&builder, value, "\"")
		} else {
			builder.WriteString(value)
		}
		nProps++
	}

	builder.WriteString("}]")
	return builder.String()
}

func RetrieveDatasetOrSnapshotInfos(api core.Session, names []string, propsList []string, params typeRetrieveParams) ([]map[string]interface{}, error) {
	var endpoint string
	switch params.retrieveType {
	case "dataset":
		endpoint = "pool.dataset.query"
	case "snapshot":
		endpoint = "zfs.snapshot.query"
	default:
		return nil, fmt.Errorf("Unrecognised retrieve format \"" + params.retrieveType + "\"")
	}
	
	var builder strings.Builder
	builder.WriteString("[[ ")
	// first arg = query-filter
	if len(names) == 1 {
		builder.WriteString("[\"id\", \"=\", ")
		core.WriteEncloseAndEscape(&builder, names[0], "\"")
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
	fmt.Println(query)

	data, err := api.CallString(endpoint, "20s", query)
	if err != nil {
		return nil, err
	}

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

	datasetMap := make(map[string]map[string]interface{})
	datasetMapKeys := make([]string, 0, 0)

	// Do not refactor this loop condition into a range!
	// This loop modifies the size of resultsList as it iterates.
	for i := 0; i < len(resultsList); i++ {
		children, _ := core.ExtractJsonArrayOfMaps(resultsList[i], "children")
		if len(children) > 0 {
			resultsList = append(append(resultsList[0:i+1], children...), resultsList[i+1:]...)
		}

		var name string
		if nameValue, ok := resultsList[i]["name"]; ok {
			if nameStr, ok := nameValue.(string); ok {
				name = nameStr
			}
		}
		if len(name) == 0 {
			continue
		}
		if _, exists := datasetMap[name]; exists {
			continue
		}

		dict := make(map[string]interface{})
		dict["name"] = name

		var propsMap map[string]interface{}
		if props, ok := resultsList[i]["properties"]; ok {
			propsMap, ok = props.(map[string]interface{})
		}
		for key, value := range propsMap {
			if valueMap, ok := value.(map[string]interface{}); ok {
				if actualValue, ok := valueMap["parsed"]; ok {
					dict[key] = actualValue
				} else if actualValue, ok := valueMap["value"]; ok {
					dict[key] = actualValue
				}
			}
		}
		datasetMap[name] = dict
		datasetMapKeys = append(datasetMapKeys, name)
	}

	slices.Sort(datasetMapKeys)
	nKeys := len(datasetMapKeys)
	datasetList := make([]map[string]interface{}, nKeys, nKeys)
	for i, _ := range datasetMapKeys {
		datasetList[i] = datasetMap[datasetMapKeys[i]]
	}

	return datasetList, nil
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

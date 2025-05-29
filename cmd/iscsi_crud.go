package cmd

import (
	//"encoding/json"
	"fmt"
	"log"
	//"path"
	"strconv"
	"strings"
	//"time"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

type iscsiCrudFeature struct {
	kind string
	defValue interface{}
	description string
}

var iscsiCrudCategories = []string { "target", "extent", "targetextent", "initiator", "portal", "auth" }

var iscsiCrudIdentifierMap = map[string][]string {
	"target": []string {"id", "name", "alias"},
	"extent": []string {"id", "name", "disk"},
	"targetextent": []string {"id", "target", "extent"},
	"initiator": []string {"id", "initiators", "comment"},
	"portal": []string {"id", "listen", "tag", "comment"},
	"auth": []string {"id", "user"},
}

var iscsiCrudFeatureMap = map[string]map[string]iscsiCrudFeature {
	"target": map[string]iscsiCrudFeature {
		"alias": iscsiCrudFeature { kind: "String", defValue: "", description: "Path of attached extent" },
	},
	"extent": map[string]iscsiCrudFeature {
		"disk": iscsiCrudFeature { kind: "String", defValue: "", description: "Path to zvol" },
	},
	"targetextent": map[string]iscsiCrudFeature {
		"target": iscsiCrudFeature { kind: "String", defValue: "", description: "ID or name of target" },
		"lunid":  iscsiCrudFeature { kind: "Int64", defValue: 0, description: "LUN ID" },
		"extent": iscsiCrudFeature { kind: "String", defValue: "", description: "ID or name of extent" },
	},
	"initiator": map[string]iscsiCrudFeature {
		"initiators": iscsiCrudFeature { kind: "String", defValue: "", description: "List of initiators in this group" },
		"comment":    iscsiCrudFeature { kind: "String", defValue: "", description: "Initiator group description/comment" },
	},
	"portal": map[string]iscsiCrudFeature {
		"listen":  iscsiCrudFeature { kind: "StringArray", defValue: "", description: "Remote IP:port" },
		"tag":     iscsiCrudFeature { kind: "Int64", defValue: 0, description: "Portal tag" },
		"comment": iscsiCrudFeature { kind: "String", defValue: "", description: "Portal description/comment" },
	},
	"auth": map[string]iscsiCrudFeature {
		"user": iscsiCrudFeature { kind: "String", defValue: "", description: "User name" },
	},
}

func WrapIscsiCrudFunc(cmdFunc func(*cobra.Command,string,core.Session,[]string)error, category string) func(*cobra.Command,[]string) error {
	return func(cmd *cobra.Command, args []string) error {
		api := InitializeApiClient()
		if api == nil {
			return nil
		}
		err := cmdFunc(cmd, category, api, args)
		return api.Close(err)
	}
}

func AddIscsiCrudCommandFlag(cmd *cobra.Command, name string, feature iscsiCrudFeature) {
	switch feature.kind {
		case "String":
			cmd.Flags().String(name, feature.defValue.(string), feature.description)
		case "StringArray":
			cmd.Flags().String(name, feature.defValue.(string), feature.description)
		case "Int64":
			if intValue, ok := feature.defValue.(int); ok {
				cmd.Flags().Int(name, intValue, feature.description)
			} else if int64Value, ok := feature.defValue.(int64); ok {
				cmd.Flags().Int64(name, int64Value, feature.description)
			}
		default:
			log.Fatal("Flag type \"" + feature.kind + "\" is currently not supported by AddIscsiCrudCommandFlag()")
	}
}

var iscsiCrudListEnums map[string][]string
var iscsiCrudUpdateCreateEnums map[string][]string

func AddIscsiCrudCommands(parentCmd *cobra.Command) {
	listFormatDesc := AddFlagsEnum(&iscsiCrudListEnums, "format", []string{"csv", "json", "table", "compact"})

	for _, category := range iscsiCrudCategories {
		cmdList := &cobra.Command { Use: "list", RunE: WrapIscsiCrudFunc(iscsiCrudList, category) }
		cmdCreate := &cobra.Command { Use: "create", RunE: WrapIscsiCrudFunc(iscsiCrudUpdateCreate, category) }
		cmdUpdate := &cobra.Command { Use: "update", RunE: WrapIscsiCrudFunc(iscsiCrudUpdateCreate, category) }
		cmdDelete := &cobra.Command { Use: "delete", RunE: WrapIscsiCrudFunc(iscsiCrudDelete, category) }

		features := iscsiCrudFeatureMap[category]
		for name, f := range features {
			AddIscsiCrudCommandFlag(cmdCreate, name, f)
			AddIscsiCrudCommandFlag(cmdUpdate, name, f)
		}
		cmdUpdate.Flags().Bool("create", false, "If the iscsi category doesn't exist, create it with the given properties")

		cmdList.Flags().BoolP("recursive", "r", false, "")
		cmdList.Flags().BoolP("user-properties", "u", false, "Include user-properties")
		cmdList.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
		cmdList.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
		cmdList.Flags().String("format", "table", "Output table format. Defaults to \"table\" " + listFormatDesc)
		cmdList.Flags().StringP("output", "o", "", "Output property list")
		cmdList.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
		cmdList.Flags().Bool("all", false, "Output all properties")

		cmd := &cobra.Command { Use: category }
		cmd.AddCommand(cmdList)
		cmd.AddCommand(cmdCreate)
		cmd.AddCommand(cmdUpdate)
		cmd.AddCommand(cmdDelete)
		parentCmd.AddCommand(cmd)
	}
}

func iscsiCrudQuery(api core.Session, category string, values []string, properties []string, extras typeQueryParams) typeQueryResponse {
	if len(values) == 0 {
		response, _ := QueryApi(api, "iscsi." + category, nil, nil, properties, extras)
		return response
	}
	combined := typeQueryResponse {
		resultsMap: make(map[string]map[string]interface{}),
		intKeys: make([]int, 0),
		strKeys: make([]string, 0),
	}

	idValues := make([]string, 0)
	for _, v := range values {
		if _, errNotNumber := strconv.Atoi(v); errNotNumber == nil {
			idValues = append(idValues, v)
		}
	}
	if len(idValues) > 0 {
		response, err := QueryApi(api, "iscsi." + category, idValues, core.StringRepeated("id", len(idValues)), properties, extras)
		if err == nil {
			MergeResponseInto(&combined, &response)
		}
	}

	params := iscsiCrudIdentifierMap[category]
	for _, attr := range params {
		queryValues := values
		if attr == "disk" {
			queryValues = make([]string, len(values))
			for i := 0; i < len(values); i++ {
				if strings.HasPrefix(values[i], "zvol/") {
					queryValues[i] = values[i]
				} else {
					queryValues[i] = "zvol/" + values[i]
				}
			}
		} else if attr == "listen" {
			
		} else if iscsiCrudFeatureMap[category][attr].kind == "StringArray" {
			queryValues = make([]string, len(values))
			for i := 0; i < len(values); i++ {
				if strings.HasPrefix(values[i], "[") {
					queryValues[i] = values[i]
				} else {
					queryValues[i] = "[" + values[i] + "]"
				}
			}
		}
		response, err := QueryApi(api, "iscsi." + category, queryValues, core.StringRepeated(attr, len(queryValues)), properties, extras)
		if err != nil {
			continue
		}
		MergeResponseInto(&combined, &response)
	}
	return combined
}

func iscsiQueryTargetExtentWithJoin(api core.Session, values []string, properties []string, extras typeQueryParams) typeQueryResponse {
	response := iscsiCrudQuery(api, "targetextent", nil, properties, extras)

	oldShouldGetAll := extras.shouldGetAllProps
	extras.shouldGetAllProps = false

	targetResponse := iscsiCrudQuery(api, "target", values, nil, extras)
	extentResponse := iscsiCrudQuery(api, "extent", values, nil, extras)

	missingTargets := make(map[string]string)
	missingExtents := make(map[string]string)
	listToRemove := make([]string, 0)
	for k, _ := range response.resultsMap {
		found := false
		missingTarget := ""
		missingExtent := ""
		if targetId, ok := response.resultsMap[k]["target"]; ok {
			idStr := fmt.Sprint(targetId)
			if target, ok := targetResponse.resultsMap[idStr]; ok {
				response.resultsMap[k]["target_name"], _ = target["name"]
				found = true
			} else {
				missingTarget = idStr
			}
		}
		if extentId, ok := response.resultsMap[k]["extent"]; ok {
			idStr := fmt.Sprint(extentId)
			if extent, ok := extentResponse.resultsMap[idStr]; ok {
				response.resultsMap[k]["extent_name"], _ = extent["name"]
				found = true
			} else {
				missingExtent = idStr
			}
		}
		if !found {
			listToRemove = append(listToRemove, k)
		} else if missingTarget != "" {
			missingTargets[missingTarget] = k
		} else if missingExtent != "" {
			missingExtents[missingExtent] = k
		}
	}

	DeleteResponseEntries(&response, listToRemove)

	listMissingTargets := make([]string, 0)
	for k, _ := range missingTargets {
		listMissingTargets = append(listMissingTargets, k)
	}
	listMissingExtents := make([]string, 0)
	for k, _ := range missingExtents {
		listMissingExtents = append(listMissingExtents, k)
	}

	if len(listMissingTargets) > 0 {
		subRes, _ := QueryApi(api, "iscsi.target", listMissingTargets, core.StringRepeated("id", len(listMissingTargets)), nil, extras)
		for targetId, obj := range subRes.resultsMap {
			if teId, ok := missingTargets[targetId]; ok {
				response.resultsMap[teId]["target_name"], _ = obj["name"]
			}
		}
	}
	if len(listMissingExtents) > 0 {
		subRes, _ := QueryApi(api, "iscsi.extent", listMissingExtents, core.StringRepeated("id", len(listMissingExtents)), nil, extras)
		for extentId, obj := range subRes.resultsMap {
			if teId, ok := missingExtents[extentId]; ok {
				response.resultsMap[teId]["extent_name"], _ = obj["name"]
			}
		}
	}
	extras.shouldGetAllProps = oldShouldGetAll
	return response
}

func iscsiCrudList(cmd *cobra.Command, category string, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, false, iscsiCrudListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsStringTrue(options.allFlags, "parsable")),
		shouldGetAllProps:  core.IsStringTrue(options.allFlags, "all") || (category == "targetextent" && len(properties) == 0),
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}

	var response typeQueryResponse

	if category == "targetextent" {
		response = iscsiQueryTargetExtentWithJoin(api, args, properties, extras)
	} else {
		response = iscsiCrudQuery(api, category, args, properties, extras)
	}

	results := GetListFromQueryResponse(&response)

	required := []string{"id"}
	if category == "target" || category == "extent" {
		required = append(required, "name")
	}

	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(results, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, category + "s", columnsList, results)
	PrintTable(api, str)
	return err
}

func iscsiResolveTargets(args, attrs, values []string) {
	for i, arg := range args {
		colonPos := strings.Index(arg, ":")
		slashPos := strings.Index(arg, "/")
		isDisk := true
		if slashPos >= 0 && colonPos >= 0 {
			isDisk = slashPos < colonPos
		} else if slashPos == -1 {
			isDisk = false
		}
		if isDisk {
			attrs[i] = "alias"
		} else {
			attrs[i] = "name"
		}
		values[i] = arg
	}
}

func iscsiResolveExtents(args, attrs, values []string) {
	for i, arg := range args {
		colonPos := strings.Index(arg, ":")
		slashPos := strings.Index(arg, "/")
		isDisk := true
		if slashPos >= 0 && colonPos >= 0 {
			isDisk = slashPos < colonPos
		} else if slashPos == -1 {
			isDisk = false
		}
		if isDisk {
			attrs[i] = "disk"
			if strings.HasPrefix(arg, "zvol/") {
				values[i] = arg
			} else {
				values[i] = "zvol/" + arg
			}
		} else {
			attrs[i] = "name"
			values[i] = arg
		}
	}
}

func iscsiResolveTargetExtents(args, attrs, values []string) {
	log.Fatal("creating/updating iscsi targetextents is currently not supported")
}

func iscsiResolveInitiators(args, attrs, values []string) {
	log.Fatal("creating/updating iscsi initiators is currently not supported")
}

func iscsiResolvePortals(args, attrs, values []string) {
	log.Fatal("creating/updating iscsi portals is currently not supported")
}

func iscsiResolveAuths(args, attrs, values []string) {
	log.Fatal("creating/updating iscsi auths is currently not supported")
}

func iscsiCrudUpdateCreate(cmd *cobra.Command, category string, api core.Session, args []string) error {
	isUpdate := false
	isCreate := false
	if strings.HasPrefix(cmd.Use, "update") {
		isUpdate = true
	} else if strings.HasPrefix(cmd.Use, "create") {
		isCreate = true
	} else {
		log.Fatal("iscsiCrudUpdateCreate was called from a command that was neither update nor create")
	}

	options, _ := GetCobraFlags(cmd, false, iscsiCrudUpdateCreateEnums)
	if isUpdate {
		isCreate = core.IsStringTrue(options.allFlags, "create")
		RemoveFlag(options, "create")
	}

	outMap := make(map[string]interface{})

	for propName, valueStr := range options.usedFlags {
		isProp := false
		switch propName {
		case "option":
			kvArray := ConvertParamsStringToKvArray(valueStr)
			if err := WriteKvArrayToMap(outMap, kvArray, iscsiCrudUpdateCreateEnums); err != nil {
				return err
			}
		default:
			isProp = true
		}
		if isProp {
			value, err := ParseStringAndValidate(propName, valueStr, iscsiCrudUpdateCreateEnums)
			if err != nil {
				return err
			}
			outMap[propName] = value
		}
	}

	attrs := make([]string, len(args))
	values := make([]string, len(args))
	switch category {
	case "target":
		iscsiResolveTargets(args, attrs, values)
	case "extent":
		iscsiResolveExtents(args, attrs, values)
	case "targetextent":
		iscsiResolveTargetExtents(args, attrs, values)
	case "initiator":
		iscsiResolveInitiators(args, attrs, values)
	case "portal":
		iscsiResolvePortals(args, attrs, values)
	case "auth":
		iscsiResolveAuths(args, attrs, values)
	}

	if isUpdate {
		extras := typeQueryParams{
			valueOrder:         BuildValueOrder(true),
			shouldGetAllProps:  false,
			shouldGetUserProps: false,
			shouldRecurse:      false,
		}
		idsResponse, err := QueryApi(api, "iscsi." + category, values, attrs, nil, extras)
		if err != nil {
			return err
		}

		orderedIds := make([]interface{}, len(values))
		for _, record := range idsResponse.resultsMap {
			for i := 0; i < len(values); i++ {
				if orderedIds[i] != nil {
					continue
				}
				if value, ok := record[attrs[i]]; ok {
					if fmt.Sprint(value) == values[i] {
						orderedIds[i], _ = record["id"]
						break
					}
				}
			}
		}

		listToCreate := make([]interface{}, 0)
		listToUpdate := make([]interface{}, 0)

		for i, id := range orderedIds {
			if id != nil {
				listToUpdate = append(listToUpdate, []interface{} { id, core.DeepCopy(outMap) })
			} else {
				obj := core.DeepCopy(outMap).(map[string]interface{})
				obj[attrs[i]] = values[i]
				listToCreate = append(listToCreate, []interface{} { obj })
			}
		}

		if !isCreate && len(listToCreate) > 0 {
			var combinedAttrValues strings.Builder
			for i := 0; i < len(values); i++ {
				if i > 0 {
					combinedAttrValues.WriteString(",")
				}
				combinedAttrValues.WriteString(attrs[i])
				combinedAttrValues.WriteString(":")
				combinedAttrValues.WriteString(values[i])
			}
			return fmt.Errorf("Some %ss could not be found (%v)\nTry passing --create to create any missing %ss with the given settings", category, combinedAttrValues.String(), category)
		}

		if len(listToUpdate) > 0 {
			out, _, err := MaybeBulkApiCallArray(api, "iscsi." + category + ".update", int64(10 + 10 * len(listToUpdate)), listToUpdate, len(listToCreate) == 0)
			if err != nil {
				return err
			}
			if out != nil {
				DebugString(string(out))
			}
		}
		if len(listToCreate) > 0 {
			_, _, err := MaybeBulkApiCallArray(api, "iscsi." + category + ".create", int64(10 + 10 * len(listToCreate)), listToCreate, false)
			if err != nil {
				return err
			}
		}
	} else {
		listToCreate := make([]interface{}, 0)
		for i := 0; i < len(values); i++ {
			obj := core.DeepCopy(outMap).(map[string]interface{})
			obj[attrs[i]] = values[i]
			listToCreate = append(listToCreate, []interface{} { obj })
		}
		out, _, err := MaybeBulkApiCallArray(api, "iscsi." + category + ".create", int64(10 + 10 * len(listToCreate)), listToCreate, true)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}

	return nil
}

func iscsiCrudDelete(cmd *cobra.Command, category string, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, false, nil)
	options = options

	cmd.SilenceUsage = true

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  false,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}

	response := iscsiCrudQuery(api, category, args, nil, extras)

	idsToDelete := make([]interface{}, 0)
	for k, _ := range response.resultsMap {
		if n, errNotNumber := strconv.Atoi(k); errNotNumber == nil {
			idsToDelete = append(idsToDelete, []interface{}{n})
		} else {
			idsToDelete = append(idsToDelete, []interface{}{k})
		}
	}
	if len(idsToDelete) > 0 {
		_, _, err := MaybeBulkApiCallArray(api, "iscsi." + category + ".delete", int64(10 + 10 * len(idsToDelete)), idsToDelete, false)
		return err
	}
	return nil
}

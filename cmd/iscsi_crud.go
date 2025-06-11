package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

type iscsiCrudFeature struct {
	kind        string
	defValue    interface{}
	description string
}

var iscsiCrudCategories = []string{"target", "extent", "targetextent", "initiator", "portal", "auth"}

var iscsiCrudIdentifierMap = map[string][]string{
	"target":       []string{"id", "name", "alias"},
	"extent":       []string{"id", "name", "disk"},
	"targetextent": []string{"id", "target", "extent"},
	"initiator":    []string{"id", "initiators", "comment"},
	"portal":       []string{"id", "listen", "tag", "comment"},
	"auth":         []string{"id", "user"},
}

var iscsiCrudRequiredAttrMap = map[string][]string{
	"target":       []string{"name", "alias"},
	"extent":       []string{"name", "disk"},
	"targetextent": []string{"target", "extent"},
	"initiator":    []string{},
	"portal":       []string{"listen"},
	"auth":         []string{"tag", "user", "secret"},
}

var iscsiTargetUpdateCreateEnums map[string][]string
var iscsiExtentUpdateCreateEnums map[string][]string
var iscsiAuthUpdateCreateEnums map[string][]string

var iscsiCrudFeatureMap = map[string]map[string]iscsiCrudFeature{
	"target": map[string]iscsiCrudFeature{
		"name":  iscsiCrudFeature{kind: "String", defValue: "", description: "Target name"},
		"alias": iscsiCrudFeature{kind: "String", defValue: "", description: "Alias (path of attached extent by convention)"},
		"mode": iscsiCrudFeature{kind: "String", defValue: "iscsi", description: "" +
			AddFlagsEnum(&iscsiTargetUpdateCreateEnums, "mode", []string{"iscsi", "fc", "both"})},
		"groups":           iscsiCrudFeature{kind: "StringArray", defValue: "", description: "Array of groups"},
		"auth-networks":    iscsiCrudFeature{kind: "StringArray", defValue: "", description: "Array of authorized networks"},
		"iscsi-parameters": iscsiCrudFeature{kind: "StringArray", defValue: "", description: "JSON object of additional parameters"},
	},
	"extent": map[string]iscsiCrudFeature{
		"name": iscsiCrudFeature{kind: "String", defValue: "", description: "Extent name"},
		"disk": iscsiCrudFeature{kind: "String", defValue: "", description: "Zvol disk (incompatible with path)"},
		"type": iscsiCrudFeature{kind: "String", defValue: "disk", description: "" +
			AddFlagsEnum(&iscsiExtentUpdateCreateEnums, "type", []string{"disk", "file"})},
		"serial":   iscsiCrudFeature{kind: "String", defValue: "", description: "Serial number"},
		"path":     iscsiCrudFeature{kind: "String", defValue: "", description: "Mount path (incompatible with disk)"},
		"filesize": iscsiCrudFeature{kind: "SizeString", defValue: "", description: "File size"},
		"blocksize": iscsiCrudFeature{kind: "String", defValue: "512", description: "" +
			AddFlagsEnum(&iscsiExtentUpdateCreateEnums, "blocksize", []string{"512", "1024", "2048", "4096"})},
		"pblocksize":      iscsiCrudFeature{kind: "Bool", defValue: false, description: "?"},
		"avail-threshold": iscsiCrudFeature{kind: "Int", defValue: 0, description: "Available threshold"},
		"comment":         iscsiCrudFeature{kind: "String", defValue: "", description: "Comment"},
		"insecure-tpc":    iscsiCrudFeature{kind: "Bool", defValue: true, description: "?"},
		"xen":             iscsiCrudFeature{kind: "Bool", defValue: false, description: "Xen"},
		"rpm": iscsiCrudFeature{kind: "String", defValue: "ssd", description: "" +
			AddFlagsEnum(&iscsiExtentUpdateCreateEnums, "rpm", []string{"unknown", "ssd", "5400", "7200", "10000", "15000"})},
		"ro":      iscsiCrudFeature{kind: "Bool", defValue: false, description: "Read-only mode"},
		"enabled": iscsiCrudFeature{kind: "Bool", defValue: true, description: "Enabled"},
	},
	"targetextent": map[string]iscsiCrudFeature{
		"target": iscsiCrudFeature{kind: "String", defValue: "", description: "ID or name of target"},
		"lunid":  iscsiCrudFeature{kind: "Int64", defValue: 0, description: "LUN ID"},
		"extent": iscsiCrudFeature{kind: "String", defValue: "", description: "ID or name of extent"},
	},
	"initiator": map[string]iscsiCrudFeature{
		"initiators": iscsiCrudFeature{kind: "String", defValue: "", description: "List of initiators in this group"},
		"comment":    iscsiCrudFeature{kind: "String", defValue: "", description: "Initiator group description/comment"},
	},
	"portal": map[string]iscsiCrudFeature{
		"match-host": iscsiCrudFeature{kind: "Bool", defValue: false, description: "Set remote IP to match the host on port " + fmt.Sprint(DEFAULT_ISCSI_PORT)},
		"listen":     iscsiCrudFeature{kind: "StringArray", defValue: "", description: "Remote IP:port"},
		"comment":    iscsiCrudFeature{kind: "String", defValue: "", description: "Portal description/comment"},
	},
	"auth": map[string]iscsiCrudFeature{
		"tag":        iscsiCrudFeature{kind: "Int64", defValue: 0, description: "Authorization tag"},
		"user":       iscsiCrudFeature{kind: "String", defValue: "", description: "User name"},
		"secret":     iscsiCrudFeature{kind: "String", defValue: "", description: "Password"},
		"peeruser":   iscsiCrudFeature{kind: "String", defValue: "", description: "Peer user name"},
		"peersecret": iscsiCrudFeature{kind: "String", defValue: "", description: "Peer password"},
		"discovery-auth": iscsiCrudFeature{kind: "String", defValue: "none", description: "" +
			AddFlagsEnum(&iscsiAuthUpdateCreateEnums, "discovery-auth", []string{"none", "chap", "chap_mutual"})},
	},
}

func WrapIscsiCrudFunc(cmdFunc func(*cobra.Command, string, core.Session, []string) error, category string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		api := InitializeApiClient()
		if api == nil {
			return nil
		}
		err := cmdFunc(cmd, category, api, args)
		return api.Close(err)
	}
}

func WrapIscsiCrudFuncNoArgs(cmdFunc func(*cobra.Command, string, core.Session) error, category string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		api := InitializeApiClient()
		if api == nil {
			return nil
		}
		err := cmdFunc(cmd, category, api)
		return api.Close(err)
	}
}

func AddIscsiCrudCommandFlag(cmd *cobra.Command, name string, feature iscsiCrudFeature) {
	switch feature.kind {
	case "Bool":
		cmd.Flags().Bool(name, feature.defValue.(bool), feature.description)
	case "StringArray":
		cmd.Flags().String(name, feature.defValue.(string), feature.description)
	case "SizeString":
		fallthrough
	case "String":
		cmd.Flags().String(name, feature.defValue.(string), feature.description)
	case "Int":
		fallthrough
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

func AddIscsiCrudCommands(parentCmd *cobra.Command) {
	listFormatDesc := AddFlagsEnum(&iscsiCrudListEnums, "format", []string{"csv", "json", "table", "compact"})

	for _, category := range iscsiCrudCategories {
		cmdList := &cobra.Command{
			Use:     "list [terms...]",
			Short:   "List " + category + "s. Each parameter is a search term for a distinct entry.",
			Aliases: []string{"ls"},
			RunE:    WrapIscsiCrudFunc(iscsiCrudList, category),
		}
		cmdCreate := &cobra.Command{
			Use:   "create",
			Short: "Create a " + category + ". Flags are passed to specify the new entry.",
			Args:  cobra.ExactArgs(0),
			RunE:  WrapIscsiCrudFuncNoArgs(iscsiCrudUpdateCreate, category),
		}
		cmdUpdate := &cobra.Command{
			Use:   "update",
			Short: "Update a " + category + ", optionally with an id. Flags are passed to specify the new entry.",
			Args:  cobra.ExactArgs(0),
			RunE:  WrapIscsiCrudFuncNoArgs(iscsiCrudUpdateCreate, category),
		}
		cmdDelete := &cobra.Command{
			Use:     "delete [ids...]",
			Short:   "Delete one or more " + category + "s by id.",
			Aliases: []string{"rm"},
			RunE:    WrapIscsiCrudFunc(iscsiCrudDelete, category),
		}

		features := iscsiCrudFeatureMap[category]
		for name, f := range features {
			AddIscsiCrudCommandFlag(cmdCreate, name, f)
			AddIscsiCrudCommandFlag(cmdUpdate, name, f)
		}
		cmdUpdate.Flags().String("id", "", "id of object to update, if not set the object is searched for with the given properties")

		cmdList.Flags().BoolP("recursive", "r", false, "")
		cmdList.Flags().BoolP("user-properties", "u", false, "Include user-properties")
		cmdList.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
		cmdList.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
		cmdList.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+listFormatDesc)
		cmdList.Flags().StringP("output", "o", "", "Output property list")
		cmdList.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
		cmdList.Flags().Bool("all", false, "Output all properties")

		cmd := &cobra.Command{Use: category}
		cmd.AddCommand(cmdList)
		cmd.AddCommand(cmdCreate)
		cmd.AddCommand(cmdUpdate)
		cmd.AddCommand(cmdDelete)
		parentCmd.AddCommand(cmd)
	}
}

func iscsiCrudQuery(api core.Session, category string, values []string, properties []string, extras typeQueryParams) (typeQueryResponse, error) {
	if len(values) == 0 {
		return QueryApi(api, "iscsi."+category, nil, nil, properties, extras)
	}

	params := iscsiCrudIdentifierMap[category]
	queryValues := make([]string, len(values)*len(params))
	queryAttrs := make([]string, len(queryValues))
	for i, attr := range params {
		pos := i * len(values)
		if attr == "disk" {
			for j := 0; j < len(values); j++ {
				if strings.HasPrefix(values[j], "zvol/") {
					queryValues[pos+j] = values[j]
				} else {
					queryValues[pos+j] = "zvol/" + values[j]
				}
			}
		} else if attr == "listen" {
			defaultHostname := api.GetHostName()
			for j := 0; j < len(values); j++ {
				queryValues[pos+j] = core.IpPortToJsonString(values[j], defaultHostname, DEFAULT_ISCSI_PORT)
			}
		} else if iscsiCrudFeatureMap[category][attr].kind == "StringArray" {
			for j := 0; j < len(values); j++ {
				if strings.HasPrefix(values[j], "[") {
					queryValues[pos+j] = values[j]
				} else {
					queryValues[pos+j] = "[" + values[j] + "]"
				}
			}
		} else {
			for j := 0; j < len(values); j++ {
				queryValues[pos+j] = values[j]
			}
		}
		for j := 0; j < len(values); j++ {
			queryAttrs[pos+j] = attr
		}
	}
	return QueryApi(api, "iscsi."+category, queryValues, queryAttrs, properties, extras)
}

func iscsiQueryTargetExtentWithJoin(api core.Session, values []string, properties []string, extras typeQueryParams) (typeQueryResponse, error) {
	emptyResponse := typeQueryResponse{}

	response, err := iscsiCrudQuery(api, "targetextent", nil, properties, extras)
	if err != nil {
		return emptyResponse, err
	}

	oldShouldGetAll := extras.shouldGetAllProps
	extras.shouldGetAllProps = false

	targetResponse, err := iscsiCrudQuery(api, "target", values, nil, extras)
	if err != nil {
		return emptyResponse, err
	}
	extentResponse, err := iscsiCrudQuery(api, "extent", values, nil, extras)
	if err != nil {
		return emptyResponse, err
	}

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
	return response, nil
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
		response, err = iscsiQueryTargetExtentWithJoin(api, args, properties, extras)
	} else {
		response, err = iscsiCrudQuery(api, category, args, properties, extras)
	}

	if err != nil {
		return err
	}

	results := GetListFromQueryResponse(&response)

	required := iscsiCrudIdentifierMap[category]
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(results, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, category+"s", columnsList, results)
	PrintTable(api, str)
	return err
}

func iscsiCrudUpdateCreate(cmd *cobra.Command, category string, api core.Session) error {
	isUpdate := false
	if strings.HasPrefix(cmd.Use, "update") {
		isUpdate = true
	} else if strings.HasPrefix(cmd.Use, "create") {
		isUpdate = false
	} else {
		log.Fatal("iscsiCrudUpdateCreate was called from a command that was neither update nor create")
	}

	var updateCreateEnums map[string][]string
	switch category {
	case "target":
		updateCreateEnums = iscsiTargetUpdateCreateEnums
	case "extent":
		updateCreateEnums = iscsiExtentUpdateCreateEnums
	case "auth":
		updateCreateEnums = iscsiAuthUpdateCreateEnums
	}

	options, _ := GetCobraFlags(cmd, false, updateCreateEnums)

	givenIdStr, _ := options.usedFlags["id"]
	RemoveFlag(options, "id")
	if !isUpdate && givenIdStr != "" {
		return fmt.Errorf("--id is incompatible with create")
	}

	shouldMatchHost := core.IsStringTrue(options.allFlags, "match_host")
	RemoveFlag(options, "match_host")

	outMap := make(map[string]interface{})

	for propName, valueStr := range options.usedFlags {
		isProp := false
		switch propName {
		case "option":
			kvArray := ConvertParamsStringToKvArray(valueStr)
			if err := WriteKvArrayToMap(outMap, kvArray, updateCreateEnums); err != nil {
				return err
			}
		case "disk":
			if !strings.HasPrefix(valueStr, "zvol/") {
				valueStr = "zvol/" + valueStr
			}
			isProp = true
		default:
			isProp = true
		}
		if isProp {
			value, err := ParseStringAndValidate(propName, valueStr, updateCreateEnums)
			if err != nil {
				return err
			}
			outMap[propName] = value
		}
	}

	if category == "portal" {
		if shouldMatchHost {
			outMap["listen"] = ":"
		}
		if value, exists := outMap["listen"]; exists {
			valueStr, _ := value.(string)
			outMap["listen"] = core.IpPortToJsonString(valueStr, api.GetHostName(), DEFAULT_ISCSI_PORT)
		}
	}

	cmd.SilenceUsage = true

	method := "iscsi." + category
	params := []interface{}{outMap}

	if !isUpdate {
		required := iscsiCrudRequiredAttrMap[category]
		missingAttrs := make([]string, 0)
		for _, key := range required {
			if _, exists := outMap[key]; !exists {
				missingAttrs = append(missingAttrs, key)
			}
		}
		if len(missingAttrs) > 0 {
			return fmt.Errorf("An iSCSI "+category+" also requires these attributes to be set on creation: %v", missingAttrs)
		}
		method += ".create"
	} else if givenIdStr != "" {
		var existingId interface{}
		if idNumber, errNotNumber := strconv.Atoi(givenIdStr); errNotNumber == nil {
			existingId = idNumber
		} else {
			existingId = givenIdStr
		}
		params = append([]interface{}{existingId}, params...)
		method += ".update"
	} else {
		identifiers := iscsiCrudIdentifierMap[category]
		queryFilter := make([]interface{}, 0)
		for _, key := range identifiers {
			if value, exists := outMap[key]; exists {
				queryFilter = append(queryFilter, []interface{}{
					key,
					"=",
					value,
				})
			}
		}

		queryParams := []interface{}{
			queryFilter,
			make(map[string]interface{}),
		}
		out, err := core.ApiCall(api, "iscsi."+category+".query", defaultCallTimeout, queryParams)
		if err != nil {
			return err
		}

		var jsonResponse interface{}
		if err = json.Unmarshal(out, &jsonResponse); err != nil {
			return fmt.Errorf("response error: %v", err)
		}

		responseMap, ok := jsonResponse.(map[string]interface{})
		if !ok {
			return fmt.Errorf("API response was not a JSON object")
		}

		resultsList, errMsg := core.ExtractJsonArrayOfMaps(responseMap, "result")
		if errMsg != "" {
			return fmt.Errorf("API response results: " + errMsg)
		}

		if len(resultsList) != 1 {
			if len(resultsList) == 0 {
				return fmt.Errorf("No matches for this %s were found", category)
			}
			msg := fmt.Sprintf("%d matches for this %s were found:", len(resultsList), category)
			columnsList := GetUsedPropertyColumns(resultsList, []string{"id"})
			str, err := core.BuildTableData("compact", category+"s", columnsList, resultsList)
			if err != nil {
				msg += "\n" + str
			}
			return fmt.Errorf(msg)
		}

		existingId, _ := resultsList[0]["id"]

		if len(queryFilter) == len(outMap) {
			fmt.Println("Only identifiable parameters for the " + category + " were specified,\n" +
				"and no id was provided, therefore no change would take place.\n" +
				"The id for this " + category + " is " + fmt.Sprint(existingId) +
				"\nExiting.")
			return nil
		}

		params = append([]interface{}{existingId}, params...)
		method += ".update"
	}

	DebugString(method)
	DebugJson(params)

	out, err := core.ApiCall(api, method, defaultCallTimeout, params)
	if err != nil {
		return err
	}
	DebugString(string(out))
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

	response, err := iscsiCrudQuery(api, category, args, nil, extras)
	if err != nil {
		return err
	}

	idsToDelete := make([]interface{}, 0)
	for idKey, _ := range response.resultsMap {
		if n, errNotNumber := strconv.Atoi(idKey); errNotNumber == nil {
			idsToDelete = append(idsToDelete, []interface{}{n})
		} else {
			idsToDelete = append(idsToDelete, []interface{}{idKey})
		}
	}
	if len(idsToDelete) > 0 {
		out, _, err := MaybeBulkApiCallArray(api, "iscsi."+category+".delete", int64(10+10*len(idsToDelete)), idsToDelete, true)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}
	fmt.Printf("Deleted %d %ss\n", len(idsToDelete), category)
	return nil
}

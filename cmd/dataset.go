package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
	Use: "dataset",
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("dataset")
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var datasetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a dataset/zvol.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		createOrUpdateDataset("create", validateAndLogin(), args)
	},
}

var datasetUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Updates an existing dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
	Run: func(cmd *cobra.Command, args []string) {
		createOrUpdateDataset("update", validateAndLogin(), args)
	},
}

var datasetDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
	Run: func(cmd *cobra.Command, args []string) {
		deleteDataset(validateAndLogin(), args)
	},
}

var datasetListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Prints a table of all datasets/zvols, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		listDataset(validateAndLogin(), args)
	},
}

var datasetInspectCmd = &cobra.Command{
	Use:     "inspect",
	Short:   "Prints properties of a dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"get"},
	Run: func(cmd *cobra.Command, args []string) {
		inspectDataset(validateAndLogin(), args)
	},
}

var datasetPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a clone dataset to no longer depend on the origin snapshot.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		promoteDataset(validateAndLogin(), args)
	},
}

var datasetRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename a ZFS dataset",
	Long: `Renames the given dataset. The new target can be located anywhere in the ZFS hierarchy, with the exception of snapshots.
Snapshots can only be re‐named within the parent file system or volume.
When renaming a snapshot, the parent file system of the snapshot does not need to be specified as part of the second argument.
Renamed file systems can inherit new mount points, in which case they are unmounted and remounted at the new mount point.`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"mv"},
	Run: func(cmd *cobra.Command, args []string) {
		renameDataset(validateAndLogin(), args)
	},
}

var g_datasetParametersCreateUpdate = []core.Parameter{
	core.MakeParameter("String", "", "comments", "", "User defined comments"),
	core.MakeParameter("String", "", "sync", "standard", "Controls the behavior of synchronous requests (\"standard\",\"always\",\"disabled\")"),
	core.MakeParameter("String", "", "snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible  (\"hidden\", \"visible\")"),
	core.MakeParameter("String", "", "compression", "off", "Controls the compression algorithm used for this dataset\n(\"on\",\"off\",\"gzip\","+
		"\"gzip-{n}\",\"lz4\",\"lzjb\",\"zle\",\"zstd\",\"zstd-{n}\",\"zstd-fast\",\"zstd-fast-{n}\")"),
	core.MakeParameter("String", "", "atime", "off", "Controls whether the access time for files is updated when they are read (\"on\",\"off\")"),
	core.MakeParameter("String", "", "exec", "off", "Controls whether processes can be executed from within this file system (\"on\",\"off\")"),
	core.MakeParameter("String", "", "managedby", "truenas-admin", "Manager of this dataset, must not be empty"),
	core.MakeParameter("Bool", "", "quota", false, ""),
	//core.MakeParameter("Bool", "", "quota_warning", false, ""),
	//core.MakeParameter("Bool", "", "quota_critical", false, ""),
	core.MakeParameter("Bool", "", "refquota", false, ""),
	//core.MakeParameter("Bool", "", "refquota_warning", false, ""),
	//core.MakeParameter("Bool", "", "refquota_critical", false, ""),
	core.MakeParameter("Bool", "", "reservation", false, ""),
	core.MakeParameter("Bool", "", "refreservation", false, ""),
	core.MakeParameter("Bool", "", "special_small_block_size", false, ""),
	core.MakeParameter("Bool", "", "copies", false, ""),
	core.MakeParameter("Bool", "", "deduplication", false, ""),
	core.MakeParameter("Bool", "", "checksum", false, ""),
	core.MakeParameter("Bool", "", "readonly", false, ""),
	core.MakeParameter("Bool", "", "recordsize", false, ""),
	core.MakeParameter("Bool", "", "casesensitivity", false, ""),
	core.MakeParameter("Bool", "", "aclmode", false, ""),
	core.MakeParameter("Bool", "", "acltype", false, ""),
	core.MakeParameter("Bool", "", "share_type", false, ""),
	core.MakeParameter("Bool", "p", "create_parents", false, "Creates all the non-existing parent datasets"),
	core.MakeParameter("String", "", "user_props", "", "Sets the specified properties"),
	core.MakeParameter("String", "o", "option", "", "Specify property=value,..."),
	core.MakeParameter("Int64", "V", "volume", 0, "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size."),
	core.MakeParameter("String", "b", "volblocksize", "512", "Volume block size (\"512\",\"1K\",\"2K\",\"4K\",\"8K\",\"16K\",\"32K\",\"64K\",\"128K\")"),
	core.MakeParameter("Bool", "s", "sparse", false, "Creates a sparse volume with no reservation"),
	core.MakeParameter("Bool", "", "force_size", false, ""),
	core.MakeParameter("String", "", "snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible (\"hidden\",\"visible\")"),
}

var g_datasetParametersDelete = []core.Parameter{
	core.MakeParameter("Bool", "r", "recursive", false, "Also delete/destroy all children datasets. When the root dataset is specified,\n"+
		"it will destroy all the children of the root dataset present leaving root dataset intact"),
	core.MakeParameter("Bool", "f", "force", false, "Force delete busy datasets"),
}

var g_datasetParametersListInspect = []core.Parameter{
	core.MakeParameter("Bool", "r", "recursive", false, "Retrieves properties for children"),
	core.MakeParameter("Bool", "u", "user-properties", false, "Include user-properties"),
	core.MakeParameter("Bool", "j", "json", false, "Equivalent to --format=json"),
	core.MakeParameter("Bool", "H", "no-headers", false, "Equivalent to --format=compact. More easily parsed by scripts"),
	core.MakeParameter("String", "", "format", "table", "Format (csv|json|table|compact) (default \"table\")"),
	core.MakeParameter("String", "o", "output", "", "Output property list"),
	core.MakeParameter("Bool", "a", "all", false, "Output all properties"),
	//core.MakeParameter("Bool", "p", "parseable", false, ""),
	core.MakeParameter("String", "s", "source", "default", "A comma-separated list of sources to display.\n"+
		"Those properties coming from a source other than those in this list are ignored.\n"+
		"Each source must be one of the following: local, default, inherited, temporary, received, or none.\n"+
		"The default value is all sources."),
}

var g_datasetParametersRename = []core.Parameter{
	core.MakeParameter("Bool", "s", "update-shares", false, "Will update any shares as part of rename"),
}

func init() {
	inputs := make([]reflect.Value, 5)
	for i := 0; i < len(g_datasetParametersCreateUpdate); i++ {
		core.AddParameterToFlags(datasetCreateCmd.Flags(), inputs, g_datasetParametersCreateUpdate, i)
		core.AddParameterToFlags(datasetUpdateCmd.Flags(), inputs, g_datasetParametersCreateUpdate, i)
	}
	for i := 0; i < len(g_datasetParametersDelete); i++ {
		core.AddParameterToFlags(datasetDeleteCmd.Flags(), inputs, g_datasetParametersDelete, i)
	}
	for i := 0; i < len(g_datasetParametersListInspect); i++ {
		core.AddParameterToFlags(datasetListCmd.Flags(), inputs, g_datasetParametersListInspect, i)
		core.AddParameterToFlags(datasetInspectCmd.Flags(), inputs, g_datasetParametersListInspect, i)
	}
	for i := 0; i < len(g_datasetParametersRename); i++ {
		core.AddParameterToFlags(datasetRenameCmd.Flags(), inputs, g_datasetParametersRename, i)
	}

	datasetCmd.AddCommand(datasetCreateCmd)
	datasetCmd.AddCommand(datasetUpdateCmd)
	datasetCmd.AddCommand(datasetDeleteCmd)
	datasetCmd.AddCommand(datasetListCmd)
	datasetCmd.AddCommand(datasetInspectCmd)
	datasetCmd.AddCommand(datasetPromoteCmd)
	datasetCmd.AddCommand(datasetRenameCmd)
	rootCmd.AddCommand(datasetCmd)
}

type typeRetrieveParams struct {
	shouldGetAllProps bool
	shouldRecurse     bool
}

func validateAndLogin() core.Session {
	var api core.Session
	if g_useMock {
		api = &core.MockSession{
			Source: &core.FileRawa{FileName: "datasets.tsv"},
		}
	} else {
		api = &core.RealSession{
			HostUrl:     g_url,
			ApiKey:      g_apiKey,
			KeyFileName: g_keyFile,
		}
	}

	err := api.Login()
	if err != nil {
		api.Close()
		log.Fatal(err)
	}

	return api
}

func createOrUpdateDataset(cmdType string, api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	if cmdType != "create" && cmdType != "update" {
		log.Fatal(errors.New("cmdType was not create or update"))
	}

	name, err := core.EncloseWith(args[0], "\"")
	if err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString("[{\"name\":")
	builder.WriteString(name)

	nProps := 0
	shouldCreateParents := false
	wroteCreateParents := false
	var userPropsStr string

	for i := 0; i < len(g_datasetParametersCreateUpdate); i++ {
		isProp := false
		switch g_datasetParametersCreateUpdate[i].Name {
		case "create_parents":
			shouldCreateParents = g_datasetParametersCreateUpdate[i].Value.VBool
		case "user_props":
			userPropsStr = g_datasetParametersCreateUpdate[i].Value.VStr
		case "option":
			paramsKV, err := convertParamsStrToFlatKVArray(g_datasetParametersCreateUpdate[i].Value.VStr)
			if err != nil {
				log.Fatal(err)
			}
			for j := 0; j < len(paramsKV); j += 2 {
				if nProps > 0 {
					builder.WriteString(",")
				}
				builder.WriteString(paramsKV[j])
				builder.WriteString(":")
				builder.WriteString(paramsKV[j+1])
				nProps++
				if paramsKV[j] == "\"create_ancestors\"" {
					wroteCreateParents = true
				}
			}
		default:
			isProp = true
		}
		if isProp && !g_datasetParametersCreateUpdate[i].IsDefault() {
			if nProps > 0 {
				builder.WriteString(",")
			}
			prop, err := core.EncloseWith(g_datasetParametersCreateUpdate[i].Name, "\"")
			if err != nil {
				log.Fatal(err)
			}
			builder.WriteString(",")
			builder.WriteString(prop)
			builder.WriteString(":")
			builder.WriteString(g_datasetParametersCreateUpdate[i].GetJsonValue())
			nProps++
		}
	}

	if !wroteCreateParents && shouldCreateParents {
		builder.WriteString(",\"create_ancestors\":true")
	}

	if userPropsStr != "" {
		paramsKV, err := convertParamsStrToFlatKVArray(userPropsStr)
		if err != nil {
			log.Fatal(err)
		}
		builder.WriteString(",user_properties:[")
		for j := 0; j < len(paramsKV); j += 2 {
			if j > 0 {
				builder.WriteString(",")
			}
			builder.WriteString("{\"key\":")
			builder.WriteString(paramsKV[j])
			builder.WriteString(",\"value\":")
			builder.WriteString(paramsKV[j+1])
			builder.WriteString("}")
		}
		builder.WriteString("]")
	}

	builder.WriteString("}]")

	_, err = api.CallString("zfs.dataset."+cmdType, "10s", builder.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}
}

func deleteDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	name, err := core.EncloseWith(args[0], "\"")
	if err != nil {
		log.Fatal(err)
	}

	_, err = api.CallString("pool.dataset.delete", "10s", "["+name+"]")
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}
}

func listDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	var datasetNames []string
	if len(args) > 0 {
		datasetNames = []string{args[0]}
	}

	format, err := getTableFormat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	properties := enumerateDatasetProperties()

	extras := typeRetrieveParams{}
	extras.shouldGetAllProps = format == "json" || core.FindParameterValue(g_datasetParametersListInspect, "all").VBool
	// `zfs list` will "recurse" if no names are specified.
	extras.shouldRecurse = len(datasetNames) == 0 || core.FindParameterValue(g_datasetParametersListInspect, "recursive").VBool

	datasets, err := retrieveDatasetInfos(api, datasetNames, properties, extras)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	var builder strings.Builder
	columnsList := getUsedPropertyColumns(datasets)

	switch format {
	case "compact":
		core.WriteListCsv(&builder, datasets, columnsList, false)
	case "csv":
		core.WriteListCsv(&builder, datasets, columnsList, true)
	case "json":
		builder.WriteString("{\"datasets\":")
		core.WriteJson(&builder, datasets)
		builder.WriteString("}\n")
	case "table":
		core.WriteListTable(&builder, datasets, columnsList, true)
	default:
		fmt.Fprintln(os.Stderr, "Unrecognised table format", format)
		return
	}

	os.Stdout.WriteString(builder.String())
}

func inspectDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	format, err := getTableFormat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	properties := enumerateDatasetProperties()

	extras := typeRetrieveParams{}
	extras.shouldGetAllProps = format == "json" || len(properties) == 0
	extras.shouldRecurse = core.FindParameterValue(g_datasetParametersListInspect, "recursive").VBool

	datasets, err := retrieveDatasetInfos(api, args, properties, extras)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	var builder strings.Builder
	columnsList := getUsedPropertyColumns(datasets)

	switch format {
	case "compact":
		core.WriteInspectCsv(&builder, datasets, columnsList, false)
	case "csv":
		core.WriteInspectCsv(&builder, datasets, columnsList, true)
	case "json":
		builder.WriteString("{\"datasets\":")
		core.WriteJson(&builder, datasets)
		builder.WriteString("}\n")
	case "table":
		core.WriteInspectTable(&builder, datasets, columnsList, true)
	default:
		fmt.Fprintln(os.Stderr, "Unrecognised table format", format)
		return
	}

	os.Stdout.WriteString(builder.String())
}

func promoteDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func renameDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	var err error
	var builder strings.Builder

	builder.WriteString("[")
	err = core.WriteEncloseWith(&builder, args[0], "\"")
	if err != nil {
		log.Fatal(err)
	}

	builder.WriteString(",{\"new_name\":")
	err = core.WriteEncloseWith(&builder, args[1], "\"")
	if err != nil {
		log.Fatal(err)
	}

	// Is this correct?
	if core.FindParameterValue(g_datasetParametersRename, "update-shares").VBool {
		builder.WriteString(",\"update_shares\":true")
	}

	builder.WriteString("}]")

	api.CallString("zfs.dataset.rename", "10s", builder.String())
}

func convertParamsStrToFlatKVArray(fullParamsStr string) ([]string, error) {
	var array []string
	if fullParamsStr == "" {
		return nil, nil
	}

	array = make([]string, 0, 0)
	params := strings.Split(fullParamsStr, ",")
	for j := 0; j < len(params); j++ {
		parts := strings.Split(params[j], "=")
		var value string
		if len(parts) == 0 {
			continue
		} else if len(parts) == 1 {
			value = "true"
		} else {
			value = parts[1]
		}
		prop, err := core.EncloseWith(parts[0], "\"")
		if err != nil {
			return array, err
		}
		if value != "true" && value != "false" && value != "null" {
			_, errNotNumber := strconv.Atoi(value)
			if errNotNumber != nil {
				value, err = core.EncloseWith(value, "\"")
				if err != nil {
					return array, err
				}
			}
		}
		array = append(array, prop, value)
	}
	return array, nil
}

func enumerateDatasetProperties() []string {
	outputParam := core.FindParameterValue(g_datasetParametersListInspect, "output")
	if outputParam == nil {
		return nil
	}

	var propsList []string
	propsStr := outputParam.VStr
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

func retrieveDatasetInfos(api core.Session, datasetNames []string, propsList []string, extras typeRetrieveParams) ([]map[string]interface{}, error) {
	var builder strings.Builder
	builder.WriteString("[[ ")
	// first arg = query-filter
	if len(datasetNames) == 1 {
		name, err := core.EncloseWith(datasetNames[0], "\"")
		if err != nil {
			log.Fatal(err)
		}
		builder.WriteString("[\"id\", \"=\", ")
		builder.WriteString(name)
		builder.WriteString("]")
	}
	builder.WriteString("], ") // end first arg
	// second arg = query-options
	builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":")
	builder.WriteString(fmt.Sprint(extras.shouldRecurse))
	builder.WriteString(", \"properties\":")
	if extras.shouldGetAllProps {
		builder.WriteString("null")
	} else {
		builder.WriteString("[")
		for i := 0; i < len(propsList); i++ {
			prop, err := core.EncloseWith(propsList[i], "\"")
			if err != nil {
				log.Fatal(err)
			}
			if i >= 1 {
				builder.WriteString(",")
			}
			builder.WriteString(prop)
		}

		builder.WriteString("]")
	}
	builder.WriteString(", \"user_properties\":false }} ]")

	query := builder.String()
	data, err := api.CallString("pool.dataset.query", "20s", query)
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

	resultsList, errMsg := extractJsonArrayOfMaps(responseMap, "result")
	if errMsg != "" {
		return nil, errors.New("API response results: " + errMsg)
	}
	if len(resultsList) == 0 {
		return nil, nil
	}

	datasetList := make([]map[string]interface{}, 0)

	// Do not refactor this loop condition into a range!
	// This loop modifies the size of resultsList as it iterates.
	for i := 0; i < len(resultsList); i++ {
		children, _ := extractJsonArrayOfMaps(resultsList[i], "children")
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
		datasetList = append(datasetList, dict)
	}

	return datasetList, nil
}

func extractJsonArrayOfMaps(obj map[string]interface{}, key string) ([]map[string]interface{}, string) {
	if value, ok := obj[key]; ok {
		if array, ok := value.([]interface{}); ok {
			if len(array) == 0 {
				return nil, ""
			}
			list := make([]map[string]interface{}, 0)
			for i := 0; i < len(array); i++ {
				if elem, ok := array[i].(map[string]interface{}); ok {
					list = append(list, elem)
				} else {
					return nil, "contained a non-object entry"
				}
			}
			return list, ""
		}
		return nil, "was not an array"
	}
	return nil, "did not contain a list"
}

func getUsedPropertyColumns(datasets []map[string]interface{}) []string {
	columnsMap := make(map[string]bool)
	columnsList := make([]string, 0)
	columnsMap["name"] = true
	for _, d := range datasets {
		for key, _ := range d {
			if _, exists := columnsMap[key]; !exists {
				columnsMap[key] = true
				columnsList = append(columnsList, key)
			}
		}
	}

	slices.Sort(columnsList)
	return append([]string{"name"}, columnsList...)
}

func getTableFormat() (string, error) {
	isJson := core.FindParameterValue(g_datasetParametersListInspect, "json").VBool
	isCompact := core.FindParameterValue(g_datasetParametersListInspect, "no-headers").VBool
	if isJson && isCompact {
		return "", errors.New("--json and --no-headers cannot be used together")
	} else if isJson {
		return "json", nil
	} else if isCompact {
		return "compact", nil
	}

	return core.FindParameterValue(g_datasetParametersListInspect, "format").VStr, nil
}

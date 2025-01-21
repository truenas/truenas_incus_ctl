package cmd

import (
    "os"
    "fmt"
    "log"
    "errors"
    "slices"
    "reflect"
    "strings"
    "encoding/json"
    "truenas/admin-tool/core"
    "github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
    Use: "dataset",
    Run: func (cmd *cobra.Command, args []string) {
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
    Run: func (cmd *cobra.Command, args []string) {
        createOrUpdateDataset("create", validateAndLogin(cmd, args, 1), args)
    },
}

var datasetUpdateCmd = &cobra.Command{
    Use:   "update",
    Short: "Updates an existing dataset/zvol.",
    Aliases: []string{"set"},
    Run: func (cmd *cobra.Command, args []string) {
        createOrUpdateDataset("update", validateAndLogin(cmd, args, 1), args)
    },
}

var datasetDeleteCmd = &cobra.Command{
    Use:   "delete",
    Short: "Deletes a dataset/zvol.",
    Aliases: []string{"rm"},
    Run: func (cmd *cobra.Command, args []string) {
        deleteDataset(validateAndLogin(cmd, args, 1), args)
    },
}

var datasetListCmd = &cobra.Command{
    Use:   "list",
    Short: "Prints a table of all datasets/zvols, given a source and an optional set of properties.",
    Aliases: []string{"ls"},
    Run: func (cmd *cobra.Command, args []string) {
        listDataset(validateAndLogin(cmd, args, 0), args)
    },
}

var datasetInspectCmd = &cobra.Command{
    Use:   "inspect",
    Short: "Prints properties of a dataset/zvol.",
    Args: cobra.MinimumNArgs(1),
    Aliases: []string{"get"},
    Run: func (cmd *cobra.Command, args []string) {
        inspectDataset(validateAndLogin(cmd, args, 0), args)
    },
}

var datasetPromoteCmd = &cobra.Command{
    Use:   "promote",
    Short: "Promote a clone dataset to no longer depend on the origin snapshot.",
    Args: cobra.ExactArgs(1),
    Run: func (cmd *cobra.Command, args []string) {
        promoteDataset(validateAndLogin(cmd, args, 0), args)
    },
}

var datasetRenameCmd = &cobra.Command{
    Use:   "rename",
    Short: "Rename a ZFS dataset",
    Long: `Renames the given dataset. The new target can be located anywhere in the ZFS hierarchy, with the exception of snapshots.
Snapshots can only be re‐named within the parent file system or volume.
When renaming a snapshot, the parent file system of the snapshot does not need to be specified as part of the second argument.
Renamed file systems can inherit new mount points, in which case they are unmounted and remounted at the new mount point.`,
    Aliases: []string{"mv"},
    Run: func (cmd *cobra.Command, args []string) {
        renameDataset(validateAndLogin(cmd, args, 0), args)
    },
}

type ParamValue struct {
    vStr string
    vInt64 int64
    vBool bool
}

type Parameter struct {
    typeStr string
    shortcut string
    name string
    value ParamValue
    defaultValue ParamValue
    description string
}

func (p *Parameter) isDefault() bool {
    if p.typeStr == "String" {
        return p.value.vStr == p.defaultValue.vStr
    }
    if p.typeStr == "Int64" {
        return p.value.vInt64 == p.defaultValue.vInt64
    }
    if p.typeStr == "Bool" {
        return p.value.vBool == p.defaultValue.vBool
    }
    return false
}

func (p *Parameter) getJsonValue() string {
    if p.typeStr == "String" {
        return "\"" + p.value.vStr + "\""
    }
    if p.typeStr == "Int64" {
        return fmt.Sprintf("%d", p.value.vInt64)
    }
    if p.typeStr == "Bool" {
        if p.value.vBool {
            return "true"
        } else {
            return "false"
        }
    }
    return "null"
}

func makeParameter(typeStr string, shortcut string, name string, value interface{}, description string) Parameter {
    p := Parameter{}
    givenType := ""
    switch value.(type) {
        case string:
            givenType = "String"
            p.value.vStr = value.(string)
            p.defaultValue.vStr = p.value.vStr
        case int64:
            givenType = "Int64"
            p.value.vInt64 = value.(int64)
            p.defaultValue.vInt64 = p.value.vInt64
        case int:
            givenType = "Int64"
            p.value.vInt64 = int64(value.(int))
            p.defaultValue.vInt64 = p.value.vInt64
        case bool:
            givenType = "Bool"
            p.value.vBool = value.(bool)
            p.defaultValue.vBool = p.value.vBool
        default:
            log.Fatal(errors.New("Unsupported parameter type " + reflect.TypeOf(value).Name()))
    }
    if typeStr != givenType {
        log.Fatal(errors.New("Type mismatch: given type is " + typeStr + ", given value is a " + givenType))
    }

    p.typeStr = typeStr
    p.shortcut = shortcut
    p.name = name
    p.description = description
    return p
}

var g_parametersCreateUpdate = []Parameter{
    makeParameter("String", "", "comments", "", "User defined comments"),
    makeParameter("String", "", "sync", "standard", "Controls the behavior of synchronous requests (\"standard\",\"always\",\"disabled\")"),
    makeParameter("String", "", "snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible  (\"hidden\", \"visible\")"),
    makeParameter("String", "", "compression", "off", "Controls the compression algorithm used for this dataset\n(\"on\",\"off\",\"gzip\"," +
                              "\"gzip-{n}\",\"lz4\",\"lzjb\",\"zle\",\"zstd\",\"zstd-{n}\",\"zstd-fast\",\"zstd-fast-{n}\")"),
    makeParameter("String", "", "atime", "off", "Controls whether the access time for files is updated when they are read (\"on\",\"off\")"),
    makeParameter("String", "", "exec", "off", "Controls whether processes can be executed from within this file system (\"on\",\"off\")"),
    makeParameter("String", "", "managedby", "truenas-admin", "Manager of this dataset, must not be empty"),
    makeParameter("Bool", "", "quota", false, ""),
    //makeParameter("Bool", "", "quota_warning", false, ""),
    //makeParameter("Bool", "", "quota_critical", false, ""),
    makeParameter("Bool", "", "refquota", false, ""),
    //makeParameter("Bool", "", "refquota_warning", false, ""),
    //makeParameter("Bool", "", "refquota_critical", false, ""),
    makeParameter("Bool", "", "reservation", false, ""),
    makeParameter("Bool", "", "refreservation", false, ""),
    makeParameter("Bool", "", "special_small_block_size", false, ""),
    makeParameter("Bool", "", "copies", false, ""),
    makeParameter("Bool", "", "deduplication", false, ""),
    makeParameter("Bool", "", "checksum", false, ""),
    makeParameter("Bool", "", "readonly", false, ""),
    makeParameter("Bool", "", "recordsize", false, ""),
    makeParameter("Bool", "", "casesensitivity", false, ""),
    makeParameter("Bool", "", "aclmode", false, ""),
    makeParameter("Bool", "", "acltype", false, ""),
    makeParameter("Bool", "", "share_type", false, ""),
    makeParameter("Bool", "p", "create_parents", true, "Creates all the non-existing parent datasets"),
    makeParameter("String", "", "user_props", "", "Sets the specified properties"),
    makeParameter("String", "o", "option", "", "Specify property=value,..."),
    makeParameter("Int64", "V", "volume", 0, "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size."),
    makeParameter("String", "b", "volblocksize", "512", "Volume block size (\"512\",\"1K\",\"2K\",\"4K\",\"8K\",\"16K\",\"32K\",\"64K\",\"128K\")"),
    makeParameter("Bool", "s", "sparse", false, "Creates a sparse volume with no reservation"),
    makeParameter("Bool", "", "force_size", false, ""),
    makeParameter("String", "", "snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible (\"hidden\",\"visible\")"),
}

var g_parametersDelete = []Parameter{
    makeParameter("Bool", "r", "recursive", false, "Also delete/destroy all children datasets. When the root dataset is specified,\n" +
        "it will destroy all the children of the root dataset present leaving root dataset intact"),
    makeParameter("Bool", "f", "force", false, "Force delete busy datasets"),
}

var g_parametersListInspect = []Parameter{
    makeParameter("Bool", "r", "recursive", false, "Retrieves properties for children"),
    makeParameter("Bool", "u", "user-properties", false, "Include user-properties"),
    makeParameter("Bool", "j", "json", false, "Equivalent to --format=json"),
    makeParameter("Bool", "H",  "no-headers", false, "Equivalent to --format=compact. More easily parsed by scripts"),
    makeParameter("String", "", "format", "table", "Format (csv|json|table|compact) (default \"table\")"),
    makeParameter("String", "o", "output", "", "Output property list"),
    //makeParameter("Bool", "p", "parseable", false, ""),
    makeParameter("String", "s", "source", "default", "A comma-separated list of sources to display.\n" +
        "Those properties coming from a source other than those in this list are ignored.\n" +
        "Each source must be one of the following: local, default, inherited, temporary, received, or none.\n" +
        "The default value is all sources."),
}

var g_parametersRename = []Parameter{
    makeParameter("Bool", "s", "update-shares", false, "Will update any shares as part of rename"),
}

func addParameter(cmdFlags interface{}, inputs []reflect.Value, paramList []Parameter, idx int) {
    shortcutInc := 0
    var methodSuffix string
    if len(paramList[idx].shortcut) > 0 {
        inputs[2] = reflect.ValueOf(paramList[idx].shortcut)
        shortcutInc = 1
        methodSuffix = "VarP"
    } else {
        methodSuffix = "Var"
    }

    typeName := paramList[idx].typeStr
    switch typeName {
        case "String":
            inputs[0] = reflect.ValueOf(&paramList[idx].value.vStr)
            inputs[2 + shortcutInc] = reflect.ValueOf(paramList[idx].value.vStr)
        case "Int64":
            inputs[0] = reflect.ValueOf(&paramList[idx].value.vInt64)
            inputs[2 + shortcutInc] = reflect.ValueOf(paramList[idx].value.vInt64)
        case "Bool":
            inputs[0] = reflect.ValueOf(&paramList[idx].value.vBool)
            inputs[2 + shortcutInc] = reflect.ValueOf(paramList[idx].value.vBool)
        default:
            log.Fatal(errors.New("Unrecognised type " + typeName))
    }
    inputs[1] = reflect.ValueOf(paramList[idx].name)
    inputs[3 + shortcutInc] = reflect.ValueOf(paramList[idx].description)

    nParams := len(inputs) - 1 + shortcutInc
    reflect.ValueOf(cmdFlags).MethodByName(typeName + methodSuffix).Call(inputs[0:nParams])
}

func init() {
    inputs := make([]reflect.Value, 5)
    for i := 0; i < len(g_parametersCreateUpdate); i++ {
        addParameter(datasetCreateCmd.Flags(), inputs, g_parametersCreateUpdate, i)
        addParameter(datasetUpdateCmd.Flags(), inputs, g_parametersCreateUpdate, i)
    }
    for i := 0; i < len(g_parametersDelete); i++ {
        addParameter(datasetDeleteCmd.Flags(), inputs, g_parametersDelete, i)
    }
    for i := 0; i < len(g_parametersListInspect); i++ {
        addParameter(datasetListCmd.Flags(), inputs, g_parametersListInspect, i)
        addParameter(datasetInspectCmd.Flags(), inputs, g_parametersListInspect, i)
    }
    for i := 0; i < len(g_parametersRename); i++ {
        addParameter(datasetRenameCmd.Flags(), inputs, g_parametersRename, i)
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

func validateAndLogin(cmd *cobra.Command, args []string, minArgs int) core.Session {
    if len(args) < minArgs {
        cmd.HelpFunc()(cmd, args)
        return nil
    }

    api := core.GetApi()
    err := api.Login()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Failed to log in")
        api.Close()
        return nil
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
    builder.WriteString("{\"name\":")
    builder.WriteString(name)
    builder.WriteString(", \"properties\":{")
    nProps := 0
    for i := 0; i < len(g_parametersCreateUpdate); i++ {
        if (!g_parametersCreateUpdate[i].isDefault()) {
            if nProps > 0 {
                builder.WriteString(", ")
            }
            prop, err := core.EncloseWith(g_parametersCreateUpdate[i].name, "\"")
            if err != nil {
                log.Fatal(err)
            }
            builder.WriteString(prop)
            builder.WriteString(":")
            builder.WriteString(g_parametersCreateUpdate[i].getJsonValue())
            nProps++
        }
    }
    builder.WriteString("} }")

    _, err = api.CallString("zfs.dataset." + cmdType, "10s", builder.String())
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

    _, err = api.CallString("pool.dataset.delete", "10s", name)
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
    datasets, err := retrieveDatasetInfos(api, datasetNames, properties, format == "json")
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
    datasets, err := retrieveDatasetInfos(api, args, properties, format == "json" || len(properties) == 0)
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
}

func enumerateDatasetProperties() []string {
    option := findParameter(g_parametersListInspect, "output")
    if option < 0 {
        return nil
    }

    var propsList []string
    propsStr := g_parametersListInspect[option].value.vStr
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

func retrieveDatasetInfos(api core.Session, datasetNames []string, propsList []string, shouldGetAllProps bool) ([]map[string]interface{}, error) {
    var builder strings.Builder
    builder.WriteString("[ ")
    if len(datasetNames) == 1 {
        name, err := core.EncloseWith(datasetNames[0], "\"")
        if err != nil {
            log.Fatal(err)
        }
        builder.WriteString("[[\"id\", \"=\", ")
        builder.WriteString(name)
        builder.WriteString("]], ")
    }

    builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":false, \"properties\":")
    if shouldGetAllProps {
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

    data, err := api.CallString("zfs.dataset.query", "20s", builder.String())
    if err != nil {
        return nil, err
    }

    var response interface{}
    if err = json.Unmarshal(data, &response); err != nil {
		return nil, errors.New(fmt.Sprintf("response error: %v", err))
	}

    responseMap, ok := response.(map[string]interface{})
    if !ok {
        return nil, errors.New("API response was not a JSON object")
    }

    var resultsList []map[string]interface{}
    if results, ok := responseMap["result"]; ok {
        if resultsArray, ok := results.([]interface{}); ok {
            resultsList = make([]map[string]interface{}, 0)
            for i := 0; i < len(resultsArray); i++ {
                if elem, ok := resultsArray[i].(map[string]interface{}); ok {
                    resultsList = append(resultsList, elem)
                } else {
                    return nil, errors.New("API response results contained a non-object entry")
                }
            }
        } else {
            return nil, errors.New("API response results was not an array")
        }
    }
    if len(resultsList) == 0 {
        return nil, errors.New("API response did not contain a list of results")
    }

    datasetList := make([]map[string]interface{}, 0)
    for i := 0; i < len(resultsList); i++ {
        var name string
        if nameValue, ok := resultsList[i]["name"]; ok {
            if nameStr, ok := nameValue.(string); ok {
                name = nameStr
            }
        }
        if len(name) == 0 {
            continue
        }

        shouldAdd := false
        if len(datasetNames) == 0 {
            shouldAdd = true
        } else {
            for j := 0; j < len(datasetNames); j++ {
                if name == datasetNames[j] {
                    shouldAdd = true
                    break
                }
            }
        }
        if !shouldAdd {
            continue
        }

        dict := make(map[string]interface{})
        dict["name"] = name

        var propsMap map[string]interface{}
        if props, ok := resultsList[i]["properties"]; ok {
            propsMap, ok = props.(map[string]interface{})
        }
        for key, value := range(propsMap) {
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

func getUsedPropertyColumns(datasets []map[string]interface{}) []string {
    columnsMap := make(map[string]bool)
    columnsList := make([]string, 0)
    columnsMap["name"] = true
    for _, d := range(datasets) {
        for key, _ := range(d) {
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
    isJson := g_parametersListInspect[findParameter(g_parametersListInspect, "json")].value.vBool
    isCompact := g_parametersListInspect[findParameter(g_parametersListInspect, "no-headers")].value.vBool
    if isJson && isCompact {
        return "", errors.New("--json and --no-headers cannot be used together")
    } else if isJson {
        return "json", nil
    } else if isCompact {
        return "compact", nil
    }

    return g_parametersListInspect[findParameter(g_parametersListInspect, "format")].value.vStr, nil
}

func findParameter(list []Parameter, name string) int {
    for i := 0; i < len(list); i++ {
        if list[i].name == name {
            return i
        }
    }
    return -1
}

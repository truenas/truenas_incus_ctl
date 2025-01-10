package cmd

import (
    "os"
    "fmt"
    "log"
    "errors"
    "reflect"
    "strings"
    "truenas/admin-tool/core"
    "github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
    Use:   "dataset",
    Short: "A brief description of your command",
    Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
    Short: "A brief description of your command",
    Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
    Run: func (cmd *cobra.Command, args []string) {
        createDataset(validateAndLogin(cmd, args, 1), args)
    },
}

var datasetDeleteCmd = &cobra.Command{
    Use:   "delete",
    Short: "A brief description of your command",
    Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
    Run: func (cmd *cobra.Command, args []string) {
        deleteDataset(validateAndLogin(cmd, args, 1), args)
    },
}

var datasetListCmd = &cobra.Command{
    Use:   "list",
    Short: "A brief description of your command",
    Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
    Run: func (cmd *cobra.Command, args []string) {
        listDataset(validateAndLogin(cmd, args, 0), args)
    },
}

type ParamValue struct {
    vStr string
    vInt64 int64
    vBool bool
}

type Parameter struct {
    typeStr string
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

func makeParameter(typeStr string, name string, value interface{}, description string) Parameter {
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
    p.name = name
    p.description = description
    return p
}

var g_parameters = []Parameter{
    makeParameter("String", "comments", "", "User defined comments"),
    makeParameter("String", "sync", "standard", "Controls the behavior of synchronous requests (\"standard\",\"always\",\"disabled\")"),
    makeParameter("String", "snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible  (\"hidden\", \"visible\")"),
    makeParameter("String", "compression", "off", "Controls the compression algorithm used for this dataset\n(\"on\",\"off\",\"gzip\"," +
                              "\"gzip-{n}\",\"lz4\",\"lzjb\",\"zle\",\"zstd\",\"zstd-{n}\",\"zstd-fast\",\"zstd-fast-{n}\")"),
    makeParameter("String", "atime", "off", "Controls whether the access time for files is updated when they are read (\"on\",\"off\")"),
    makeParameter("String", "exec", "off", "Controls whether processes can be executed from within this file system (\"on\",\"off\")"),
    makeParameter("String", "managedby", "truenas-admin", "Manager of this dataset, must not be empty"),
    makeParameter("Bool", "quota", false, ""),
    //makeParameter("Bool", "quota_warning", false, ""),
    //makeParameter("Bool", "quota_critical", false, ""),
    makeParameter("Bool", "refquota", false, ""),
    //makeParameter("Bool", "refquota_warning", false, ""),
    //makeParameter("Bool", "refquota_critical", false, ""),
    makeParameter("Bool", "reservation", false, ""),
    makeParameter("Bool", "refreservation", false, ""),
    makeParameter("Bool", "special_small_block_size", false, ""),
    makeParameter("Bool", "copies", false, ""),
    makeParameter("Bool", "deduplication", false, ""),
    makeParameter("Bool", "checksum", false, ""),
    makeParameter("Bool", "readonly", false, ""),
    makeParameter("Bool", "recordsize", false, ""),
    makeParameter("Bool", "casesensitivity", false, ""),
    makeParameter("Bool", "aclmode", false, ""),
    makeParameter("Bool", "acltype", false, ""),
    makeParameter("Bool", "share_type", false, ""),
    makeParameter("Bool", "create_parents", true, "Creates all the non-existing parent datasets"),
    makeParameter("String", "user_props", "", "Sets the specified properties"),
    makeParameter("String", "option", "", "Specify property=value,..."),
    makeParameter("Int64", "volume", 0, "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size."),
    makeParameter("String", "volblocksize", "512", "Volume block size (\"512\",\"1K\",\"2K\",\"4K\",\"8K\",\"16K\",\"32K\",\"64K\",\"128K\")"),
    makeParameter("Bool", "sparse", false, "Creates a sparse volume with no reservation"),
    makeParameter("Bool", "force_size", false, ""),
    makeParameter("String", "snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible (\"hidden\",\"visible\")"),
}

func addParameter(cmdFlags interface{}, inputs []reflect.Value, idx int) {
    typeName := g_parameters[idx].typeStr
    switch typeName {
        case "String":
            inputs[0] = reflect.ValueOf(&g_parameters[idx].value.vStr)
            inputs[2] = reflect.ValueOf(g_parameters[idx].value.vStr)
        case "Int64":
            inputs[0] = reflect.ValueOf(&g_parameters[idx].value.vInt64)
            inputs[2] = reflect.ValueOf(g_parameters[idx].value.vInt64)
        case "Bool":
            inputs[0] = reflect.ValueOf(&g_parameters[idx].value.vBool)
            inputs[2] = reflect.ValueOf(g_parameters[idx].value.vBool)
        default:
            log.Fatal(errors.New("Unrecognised type " + typeName))
    }
    inputs[1] = reflect.ValueOf(g_parameters[idx].name)
    inputs[3] = reflect.ValueOf(g_parameters[idx].description)
    reflect.ValueOf(cmdFlags).MethodByName(typeName + "Var").Call(inputs)
}

func init() {
    // Does this need ref := &name... ?
    inputs := make([]reflect.Value, 4)
    for i := 0; i < len(g_parameters); i++ {
        addParameter(datasetCreateCmd.Flags(), inputs, i)
    }

    datasetCmd.AddCommand(datasetCreateCmd)
    datasetCmd.AddCommand(datasetDeleteCmd)
    datasetCmd.AddCommand(datasetListCmd)
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
        fmt.Println("Failed to log in")
        api.Close()
        return nil
    }

    return api
}

func createDataset(api core.Session, args []string) {
    if api == nil {
        return
    }
    defer api.Close()

    name, err := core.EncloseWith(args[0], "\"")
    if err != nil {
        log.Fatal(err)
    }

    var builder strings.Builder
    builder.WriteString("{\"name\":")
    builder.WriteString(name)
    builder.WriteString(", \"properties\":{")
    nProps := 0
    for i := 0; i < len(g_parameters); i++ {
        if (!g_parameters[i].isDefault()) {
            if nProps > 0 {
                builder.WriteString(", ")
            }
            builder.WriteString("\"")
            builder.WriteString(g_parameters[i].name)
            builder.WriteString("\":")
            builder.WriteString(g_parameters[i].getJsonValue())
            nProps++
        }
    }
    builder.WriteString("} }")
    fmt.Println(builder.String())

    data, err := api.CallString("zfs.dataset.create", "10s", builder.String())
    if err != nil {
        fmt.Println("API error:", err)
        return
    }
    os.Stdout.Write(data)
    fmt.Println()
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

    data, err := api.CallString("pool.dataset.delete", "10s", name)
    if err != nil {
        fmt.Println("API error:", err)
        return
    }
    os.Stdout.Write(data)
    fmt.Println()
}

func listDataset(api core.Session, args []string) {
    if api == nil {
        return
    }
    defer api.Close()

    var builder strings.Builder
    builder.WriteString("[ ")
    if len(args) > 0 {
        name, err := core.EncloseWith(args[0], "\"")
        if err != nil {
            log.Fatal(err)
        }
        builder.WriteString("[[\"id\", \"=\", ")
        builder.WriteString(name)
        builder.WriteString("]], ")
    }

    builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":false, \"properties\":[")
    for i := 1; i < len(args); i++ {
        prop, err := core.EncloseWith(args[i], "\"")
        if err != nil {
            log.Fatal(err)
        }
        if i >= 2 {
            builder.WriteString(",")
        }
        builder.WriteString(prop)
    }
    builder.WriteString("], \"user_properties\":false }} ]")

    data, err := api.CallString("zfs.dataset.query", "20s", builder.String())
    if err != nil {
        fmt.Println("API error:", err)
        return
    }
    os.Stdout.Write(data)
    fmt.Println()
}

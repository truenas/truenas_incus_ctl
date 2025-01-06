package cmd

import (
    "os"
    "fmt"
    "log"
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

var g_comments string
var g_sync string
var g_snapdir string
var g_compression string
var g_atime string
var g_exec string
var g_managedby string
var g_quota bool
var g_quota_warning bool
var g_quota_critical bool
var g_refquota bool
var g_refquota_warning bool
var g_refquota_critical bool
var g_reservation bool
var g_refreservation bool
var g_special_small_block_size bool
var g_copies bool
var g_deduplication bool
var g_checksum bool
var g_readonly bool
var g_recordsize bool
var g_casesensitivity bool
var g_aclmode bool
var g_acltype bool
var g_share_type bool
var g_create_parents bool
var g_user_props string
var g_option string
var g_volume int64
var g_volblocksize string
var g_sparse bool
var g_force_size bool
var g_snapdev string

func init() {
    datasetCreateCmd.Flags().StringVar(&g_comments, "comments", "", "User defined comments")
    datasetCreateCmd.Flags().StringVar(&g_sync, "sync", "standard", "Controls the behavior of synchronous requests (\"standard\",\"always\",\"disabled\")")
    datasetCreateCmd.Flags().StringVar(&g_snapdir, "snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible  (\"hidden\", \"visible\")")
    datasetCreateCmd.Flags().StringVar(&g_compression, "compression", "off", "Controls the compression algorithm used for this dataset\n(\"on\",\"off\",\"gzip\"," +
                              "\"gzip-{n}\",\"lz4\",\"lzjb\",\"zle\",\"zstd\",\"zstd-{n}\",\"zstd-fast\",\"zstd-fast-{n}\")")
    datasetCreateCmd.Flags().StringVar(&g_atime, "atime", "off", "Controls whether the access time for files is updated when they are read (\"on\",\"off\")")
    datasetCreateCmd.Flags().StringVar(&g_exec, "exec", "off", "Controls whether processes can be executed from within this file system (\"on\",\"off\")")
    datasetCreateCmd.Flags().StringVar(&g_managedby, "managedby", "truenas-admin", "Manager of this dataset, must not be empty")

    datasetCreateCmd.Flags().BoolVar(&g_quota, "quota", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_quota_warning, "quota_warning", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_quota_critical, "quota_critical", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_refquota, "refquota", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_refquota_warning, "refquota_warning", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_refquota_critical, "refquota_critical", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_reservation, "reservation", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_refreservation, "refreservation", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_special_small_block_size, "special_small_block_size", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_copies, "copies", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_deduplication, "deduplication", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_checksum, "checksum", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_readonly, "readonly", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_recordsize, "recordsize", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_casesensitivity, "casesensitivity", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_aclmode, "aclmode", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_acltype, "acltype", false, "")
    datasetCreateCmd.Flags().BoolVar(&g_share_type, "share_type", false, "")

    datasetCreateCmd.Flags().BoolVar(&g_create_parents, "create_parents", true, "Creates all the non-existing parent datasets")
    datasetCreateCmd.Flags().StringVar(&g_user_props, "user_props", "", "Sets the specified properties")
    datasetCreateCmd.Flags().StringVar(&g_option, "option", "", "Specify property=value,...")
    datasetCreateCmd.Flags().Int64Var(&g_volume, "volume", 0, "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size.")
    datasetCreateCmd.Flags().StringVar(&g_volblocksize, "volblocksize", "512", "Volume block size (\"512\",\"1K\",\"2K\",\"4K\",\"8K\",\"16K\",\"32K\",\"64K\",\"128K\")")
    datasetCreateCmd.Flags().BoolVar(&g_sparse, "sparse", false, "Creates a sparse volume with no reservation")
    datasetCreateCmd.Flags().BoolVar(&g_force_size, "force_size", false, "")
    datasetCreateCmd.Flags().StringVar(&g_snapdev, "snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible (\"hidden\",\"visible\")")

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
    if len(args) >= 2 {
        builder.WriteString(", \"properties\": {")
        for i := 1; i < len(args); i++ {
            
        }
        builder.WriteString("}")
    }
    builder.WriteString("}")

    data, err := api.CallString("zfs.dataset.create", "10s", builder.String())
    if err != nil {
        fmt.Println("API error:", err)
        return
    }
    os.Stdout.Write(data)
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

    builder.WriteString("{\"extra\": {\"flat\": false, \"retrieve_children\":false, \"properties\":[")
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
}

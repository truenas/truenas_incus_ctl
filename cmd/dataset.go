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
        createDataset(validateAndLogin(cmd, args), args)
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
        deleteDataset(validateAndLogin(cmd, args), args)
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
        listDataset(validateAndLogin(cmd, args), args)
    },
}

func init() {
    datasetCmd.AddCommand(datasetCreateCmd)
    datasetCmd.AddCommand(datasetDeleteCmd)
    datasetCmd.AddCommand(datasetListCmd)
    rootCmd.AddCommand(datasetCmd)
}

func validateAndLogin(cmd *cobra.Command, args []string) core.Session {
    if len(args) == 0 {
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
    api.Call("pool.dataset.delete", "10s", name)
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

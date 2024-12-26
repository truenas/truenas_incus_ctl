package cmd

import (
    "os"
    "fmt"
    //"flag"
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

    params := "[ [[\"id\", \"=\", \"dozer/jacks-incus\"]], {\"extra\": {\"flat\": false,\"retrieve_children\":false,\"properties\":[\"compression\"],\"user_properties\":false }} ]"
    data, err := api.CallString("zfs.dataset.create", "10s", params)
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
    api.Call("pool.dataset.delete", "10s", args)
}

func listDataset(api core.Session, args []string) {
    if api == nil {
        return
    }
    defer api.Close()

    /*
    idParams := [...]string{ "id", "=", "dozer/jacks-incus" }
    var params [][]string
    params = append(params, idParams[:])
    */

    params := "[ [[\"id\", \"=\", \"dozer/jacks-incus\"]], {\"extra\": {\"flat\": false,\"retrieve_children\":false,\"properties\":[\"compression\"],\"user_properties\":false }} ]"
    data, err := api.CallString("zfs.dataset.query", "20s", params)
    if err != nil {
        fmt.Println("API error:", err)
        return
    }
    os.Stdout.Write(data)
}

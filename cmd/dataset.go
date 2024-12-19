package cmd

import (
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
        //fmt.Println("dataset create")
        if len(args) == 0 {
            cmd.HelpFunc()(cmd, args)
            return
        }
        api := core.GetApi()
        err := api.Login()
        if err != nil {
            fmt.Println("Failed to log in")
            return
        }
        fmt.Println(api)
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
        //fmt.Println("dataset create")
        if len(args) == 0 {
            cmd.HelpFunc()(cmd, args)
            return
        }
        api := core.GetApi()
        err := api.Login()
        if err != nil {
            fmt.Println("Failed to log in")
            return
        }
        fmt.Println(api)
    },
}

func init() {
    datasetCmd.AddCommand(datasetCreateCmd)
    rootCmd.AddCommand(datasetCmd)
}

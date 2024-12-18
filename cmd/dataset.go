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
        s, err := core.Login()
        if err != nil {
            fmt.Println("Failed to log in")
            return
        }
        fmt.Println(s)
    },
}

func init() {
    datasetCmd.PersistentFlags().StringP("comments", "", "", "")
    datasetCmd.PersistentFlags().StringP("sync", "", "", "")
    datasetCmd.PersistentFlags().StringP("snapdir", "", "", "")
    datasetCmd.PersistentFlags().StringP("atime", "", "", "")
    datasetCmd.PersistentFlags().StringP("exec", "", "", "")
    datasetCmd.PersistentFlags().StringP("managedby", "", "", "")
    datasetCmd.PersistentFlags().BoolP("quota", "", false, "")
    datasetCmd.PersistentFlags().BoolP("quota_warning", "", false, "")
    datasetCmd.PersistentFlags().BoolP("quota_critical", "", false, "")
    datasetCmd.PersistentFlags().BoolP("refquota", "", false, "")
    datasetCmd.PersistentFlags().BoolP("refquota_warning", "", false, "")
    datasetCmd.PersistentFlags().BoolP("refquota_critical", "", false, "")
    datasetCmd.PersistentFlags().BoolP("reservation", "", false, "")
    datasetCmd.PersistentFlags().BoolP("refreservation", "", false, "")
    datasetCmd.PersistentFlags().BoolP("special_small_block_size", "", false, "")
    datasetCmd.PersistentFlags().BoolP("copies", "", false, "")
    datasetCmd.PersistentFlags().BoolP("deduplication", "", false, "")
    datasetCmd.PersistentFlags().BoolP("checksum", "", false, "")
    datasetCmd.PersistentFlags().BoolP("readonly", "", false, "")
    datasetCmd.PersistentFlags().BoolP("recordsize", "", false, "")
    datasetCmd.PersistentFlags().BoolP("casesensitivity", "", false, "")
    datasetCmd.PersistentFlags().BoolP("aclmode", "", false, "")
    datasetCmd.PersistentFlags().BoolP("acltype", "", false, "")
    datasetCmd.PersistentFlags().BoolP("share_type", "", false, "")
    datasetCmd.PersistentFlags().BoolP("create_parents", "p", false, "")
    datasetCmd.PersistentFlags().StringP("user_props", "", "", "")
    datasetCmd.PersistentFlags().StringP("volume", "V", "", "")
    datasetCmd.PersistentFlags().StringP("volblocksize", "b", "", "")
    datasetCmd.PersistentFlags().BoolP("sparse", "s", false, "")
    datasetCmd.PersistentFlags().BoolP("force_size", "", false, "")
    datasetCmd.PersistentFlags().StringP("snapdev", "", "", "")

    datasetCmd.AddCommand(datasetCreateCmd)
    rootCmd.AddCommand(datasetCmd)
}

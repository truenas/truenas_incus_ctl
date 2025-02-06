package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

/*
0.1.0 initial version
0.1.1 added url, apikey and keyfile
0.1.2 added `share nfs` functionality
0.1.3 improved querying, removed --name and --id from `share nfs list`
0.1.4 added `--update-shares` to `dataset rename`
0.1.5 removed inspect command
0.1.6 `share nfs delete“ now supports <id|dataset|path>
0.1.7 `share nfs update now supports `--create`
0.1.8 most methods now return non-zero on error
*/
const VERSION = "0.1.8"

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

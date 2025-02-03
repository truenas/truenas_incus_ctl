package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

/*
0.1.0 initial version
0.1.1 added url, apikey and keyfile
0.1.2 added `share nfs` functionality
*/
const VERSION = "0.1.2"

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	//"errors"
	//"strings"
	//"os"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:   "replication",
	Short: "Start replicating a dataset from one pool to another, locally or across any network",
	Aliases: []string{"backup", "repl"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var replRunCmd = &cobra.Command{
	Use:   "start",
}

func init() {
	replRunCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runReplication(cmd, ValidateAndLogin(), args)
	}

	replCmd.AddCommand(replRunCmd)
	rootCmd.AddCommand(replCmd)
}

func runReplication(cmd *cobra.Command, api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	return nil
}

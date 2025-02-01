package cmd

import (
	//"encoding/json"
	//"errors"
	//"fmt"
	//"log"
	//"os"
	//"strconv"
	//"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var snapshotCloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "clone snapshot of ZFS dataset",
	Args:  cobra.MinimumNArgs(2),
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Take a snapshot of dataset, possibly recursive",
	Args:  cobra.MinimumNArgs(1),
}

var snapshotDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a snapshot of dataset, possibly recursive",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var snapshotListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all snapshots",
	Aliases: []string{"ls"},
}

var snapshotRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a given snapshot",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	snapshotCloneCmd.Run = func(cmd *cobra.Command, args []string) {
		cloneSnapshot(ValidateAndLogin(), args)
	}

	snapshotCreateCmd.Run = func(cmd *cobra.Command, args []string) {
		createSnapshot(ValidateAndLogin(), args)
	}

	snapshotDeleteCmd.Run = func(cmd *cobra.Command, args []string) {
		deleteSnapshot(ValidateAndLogin(), args)
	}
	
	snapshotListCmd.Run = func(cmd *cobra.Command, args []string) {
		listSnapshot(ValidateAndLogin(), args)
	}

	snapshotRollbackCmd.Run = func(cmd *cobra.Command, args []string) {
		rollbackSnapshot(ValidateAndLogin(), args)
	}

	snapshotCreateCmd.Flags().BoolP("recursive", "r", false, "")

	snapshotDeleteCmd.Flags().BoolP("recursive", "r", false, "recursively delete children")
	snapshotDeleteCmd.Flags().Bool("defer", false, "defer the deletion of snapshot")

	snapshotRollbackCmd.Flags().BoolP("force", "f", false, "force unmount of any clones")
	snapshotRollbackCmd.Flags().BoolP("recursive", "r", false, "destroy any snapshots and bookmarks more recent than the one specified")
	snapshotRollbackCmd.Flags().BoolP("recursive-clones", "R", false, "like recursive, but also destroy any clones")
	snapshotRollbackCmd.Flags().Bool("recursive-rollback", false, "perform a completem recursive rollback of each child snapshots.\n" +
		"If any child does not have specified snapshot, this operation will fail.")

	snapshotCmd.AddCommand(snapshotCloneCmd)
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRollbackCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func cloneSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func createSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func deleteSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func listSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func rollbackSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}


package cmd

import (
	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Create, list, update or delete NFS or iSCSI shares.",
}

func init() {
	rootCmd.AddCommand(shareCmd)
}

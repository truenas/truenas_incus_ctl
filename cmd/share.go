package cmd

import (
	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Create, list, update or delete network shares. Currently only NFS is supported.",
}

func init() {
	rootCmd.AddCommand(shareCmd)
}

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
0.2.0 renamed to `truenas_incus_ctl`, first published version
0.3.0 added bulk api calls, allowing for multiple datasets, snaps, shares to be edited in one go
0.3.1 fixed job waiting
0.3.2 added snapshot create --delete flag
0.4.0 added replication
0.4.1 dataset list -p fix
0.4.2 added additional repplication options
0.4.3 Increased timeout for asynchronous API calls
0.4.4 Snapshot lists are now sorted by dataset then txg
0.5.0 Add initial iSCSI support
0.5.1 Full support for iSCSI, added human-readable size parsing
0.5.2 Add a connection daemon, allowing for logins to be cached, more flexibility in handling jobs, etc
0.5.3 `share iscsi locate --activate/--deactivate`
*/
const VERSION = "0.5.3"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of this program",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

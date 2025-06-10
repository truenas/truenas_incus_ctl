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
0.6.0 added config command, changed definition of global flags
0.6.1 `share iscsi locate --create/--delete`
0.6.2 `snapshot rename` now calls zfs.snapshot.rename end-point
0.7.0 Add service commands, iscsi test, --daemon-socket to override path to the daemon's socket, add --portal and --initiator flags
0.7.1 Sends a sendtargets command before a plain discover. This seems to be required before verifying a portal
0.8.0 Add support for server side "iscsi defer" https://github.com/truenas/middleware/pull/16614
*/
const VERSION = "0.8.0"

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

package cmd

import (
	//"encoding/json"
	//"errors"
	"fmt"
	"log"
	"os"
	//"strconv"
	"strings"
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
	snapshotCreateCmd.Flags().String("exclude", "", "List of datasets to exclude")
	snapshotCreateCmd.Flags().StringP("option", "o", "", "Specify property=value,...")
	snapshotCreateCmd.Flags().Bool("suspend-vms", false, "")
	snapshotCreateCmd.Flags().Bool("vmware-sync", false, "")

	snapshotDeleteCmd.Flags().BoolP("recursive", "r", false, "recursively delete children")
	snapshotDeleteCmd.Flags().Bool("defer", false, "defer the deletion of snapshot")

	snapshotListCmd.Flags().BoolP("recursive", "r", false, "")
	snapshotListCmd.Flags().BoolP("user-properties", "u", false, "Include user-properties")
	snapshotListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	snapshotListCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	snapshotListCmd.Flags().String("format", "table", "Format (csv|json|table|compact) (default \"table\")")
	snapshotListCmd.Flags().StringP("output", "o", "", "Output property list")
	snapshotListCmd.Flags().Bool("all", false, "")

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

	var builder strings.Builder

	builder.WriteString("[{")
	builder.WriteString("\"snapshot\":")
	core.WriteEncloseAndEscape(&builder, args[0], "\"")
	builder.WriteString(",\"dataset_dst\":")
	core.WriteEncloseAndEscape(&builder, args[1], "\"")
	builder.WriteString(",\"dataset_properties\":{")

	// write properties...

	builder.WriteString("}}]")

	stmt := builder.String()
	fmt.Println(stmt)

	out, err := api.CallString("zfs.snapshot.clone", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.WriteString(string(out))
}

func createSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	usedOptions, allOptions, _ := getCobraFlags(snapshotCreateCmd)

	snapshot := args[0]
	datasetLen := strings.Index(snapshot, "@")
	if datasetLen <= 0 {
		log.Fatal(fmt.Errorf("No dataset name was found in snapshot specifier.\nExpected <datasetname>@<snapshotname>."))
	}
	dataset := snapshot[0:datasetLen]

	snapshotIsolated := snapshot[datasetLen+1:]

	var builder strings.Builder
	builder.WriteString("[{")

	builder.WriteString("\"dataset\":")
	core.WriteEncloseAndEscape(&builder, dataset, "\"")
	builder.WriteString(",\"name\":")
	core.WriteEncloseAndEscape(&builder, snapshotIsolated, "\"")

	// "naming_schema":""

	builder.WriteString(",\"recursive\":")
	builder.WriteString(allOptions["recursive"])
	builder.WriteString(",\"exclude\":[")
	builder.WriteString("]")

	if value, exists := usedOptions["suspend_vms"]; exists {
		builder.WriteString(",\"suspend_vms\":")
		builder.WriteString(value)
	}
	if value, exists := usedOptions["vmware_sync"]; exists {
		builder.WriteString(",\"vmware_sync\":")
		builder.WriteString(value)
	}

	builder.WriteString(",\"properties\":{")

	// option ...

	builder.WriteString("}}]")

	stmt := builder.String()
	fmt.Println(stmt)

	out, err := api.CallString("zfs.snapshot.create", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.WriteString(string(out))
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

	_, allOptions, _ := getCobraFlags(snapshotListCmd)

	var propsList []string

	var builder strings.Builder
	builder.WriteString("[[ ")
	// first arg = query-filter
	if len(args) == 1 {
		builder.WriteString("[\"id\", \"=\", ")
		core.WriteEncloseAndEscape(&builder, args[0], "\"")
		builder.WriteString("]")
	}
	builder.WriteString("], ") // end first arg
	// second arg = query-options
	builder.WriteString("{\"extra\":{\"flat\":false, \"retrieve_children\":")
	builder.WriteString(allOptions["recursive"])
	builder.WriteString(", \"properties\":")
	if allOptions["all"] == "true" {
		builder.WriteString("null")
	} else {
		builder.WriteString("[")
		if len(propsList) > 0 {
			core.WriteEncloseAndEscape(&builder, propsList[0], "\"")
			for i := 1; i < len(propsList); i++ {
				builder.WriteString(",")
				core.WriteEncloseAndEscape(&builder, propsList[i], "\"")
			}
		}
		builder.WriteString("]")
	}
	builder.WriteString(", \"user_properties\":false }} ]")

	query := builder.String()
	fmt.Println(query)

	out, err := api.CallString("zfs.snapshot.query", "10s", query)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.WriteString(string(out))
}

func rollbackSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}


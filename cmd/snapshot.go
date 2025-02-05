package cmd

import (
	//"encoding/json"
	"errors"
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

var g_snapshotListEnums map[string][]string

func init() {
	snapshotCloneCmd.Run = func(cmd *cobra.Command, args []string) {
		cloneSnapshot(ValidateAndLogin(), args)
	}

	snapshotCreateCmd.Run = func(cmd *cobra.Command, args []string) {
		createSnapshot(ValidateAndLogin(), args)
	}

	snapshotDeleteCmd.Run = func(cmd *cobra.Command, args []string) {
		deleteOrRollbackSnapshot(cmd, ValidateAndLogin(), args)
	}

	snapshotListCmd.Run = func(cmd *cobra.Command, args []string) {
		listSnapshot(ValidateAndLogin(), args)
	}

	snapshotRollbackCmd.Run = func(cmd *cobra.Command, args []string) {
		deleteOrRollbackSnapshot(cmd, ValidateAndLogin(), args)
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
	snapshotListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" " +
			AddFlagsEnum(&g_snapshotListEnums, "format", []string{"csv","json","table","compact"}))
	snapshotListCmd.Flags().StringP("output", "o", "", "Output property list")
	snapshotListCmd.Flags().Bool("all", false, "")

	snapshotRollbackCmd.Flags().BoolP("force", "f", false, "force unmount of any clones")
	snapshotRollbackCmd.Flags().BoolP("recursive", "r", false, "destroy any snapshots and bookmarks more recent than the one specified")
	snapshotRollbackCmd.Flags().BoolP("recursive-clones", "R", false, "like recursive, but also destroy any clones")
	snapshotRollbackCmd.Flags().Bool("recursive-rollback", false, "perform a completem recursive rollback of each child snapshots.\n"+
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
	DebugString(stmt)

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

	options, _ := GetCobraFlags(snapshotCreateCmd, nil)

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
	builder.WriteString(options.allFlags["recursive"])
	builder.WriteString(",\"exclude\":[")
	builder.WriteString("]")

	if value, exists := options.usedFlags["suspend_vms"]; exists {
		builder.WriteString(",\"suspend_vms\":")
		builder.WriteString(value)
	}
	if value, exists := options.usedFlags["vmware_sync"]; exists {
		builder.WriteString(",\"vmware_sync\":")
		builder.WriteString(value)
	}

	builder.WriteString(",\"properties\":{")

	// option ...

	builder.WriteString("}}]")

	stmt := builder.String()
	DebugString(stmt)

	out, err := api.CallString("zfs.snapshot.create", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.WriteString(string(out))
}

func deleteOrRollbackSnapshot(cmd *cobra.Command, api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	cmdType := strings.Split(cmd.Use, " ")[0]
	if cmdType != "delete" && cmdType != "rollback" {
		log.Fatal(errors.New("cmdType was not delete or rollback"))
	}

	snapshot := args[0]
	datasetLen := strings.Index(snapshot, "@")
	if datasetLen <= 0 {
		log.Fatal(fmt.Errorf("No dataset name was found in snapshot specifier.\nExpected <datasetname>@<snapshotname>."))
	}

	options, _ := GetCobraFlags(cmd, nil)
	params := BuildNameStrAndPropertiesJson(options, snapshot)
	DebugString(params)

	out, err := api.CallString("zfs.snapshot."+cmdType, "10s", params)
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}
}

func listSnapshot(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, err := GetCobraFlags(snapshotListCmd, g_snapshotListEnums)
	if err != nil {
		log.Fatal(err)
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getSnapshotListTypes(args)
	if err != nil {
		log.Fatal(err)
	}

	// `zfs list` will "recurse" if no names are specified.
	extras := typeRetrieveParams{
		retrieveType:      "snapshot",
		shouldGetAllProps: format == "json" || core.IsValueTrue(options.allFlags, "all"),
		shouldRecurse:     len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	snapshots, err := QueryApi(api, args, idTypes, properties, extras)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	required := []string{"name"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(snapshots, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	core.PrintTableDataList(format, "snapshots", columnsList, snapshots)
}

func getSnapshotListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t := core.IdentifyObject(args[i])
		if t == "id" || t == "share" {
			return typeList, errors.New("querying snapshots based on mount point is not yet supported")
		} else if t == "snapshot" {
			t = "name"
		} else if t == "snapshot_only" {
			t = "snapshot_name"
			args[i] = args[i][1:]
		}
		typeList[i] = t
	}

	return typeList, nil
}

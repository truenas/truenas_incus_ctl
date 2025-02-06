package cmd

import (
	//"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
	Use: "dataset",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var datasetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a dataset/zvol.",
	Args:  cobra.MinimumNArgs(1),
}

var datasetUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Updates an existing dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
}

var datasetDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a dataset/zvol.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var datasetListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Prints a table of all datasets/zvols, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var datasetPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a clone dataset to no longer depend on the origin snapshot.",
	Args:  cobra.ExactArgs(1),
}

var datasetRenameCmd = &cobra.Command{
	Use:   "rename [flags]... <old dataset>[@<old snapshot>] <new dataset|new snapshot>",
	Short: "Rename a ZFS dataset",
	Long: `Renames the given dataset. The new target can be located anywhere in the ZFS hierarchy, with the exception of snapshots.
Snapshots can only be re‐named within the parent file system or volume.
When renaming a snapshot, the parent file system of the snapshot does not need to be specified as part of the second argument.
Renamed file systems can inherit new mount points, in which case they are unmounted and remounted at the new mount point.`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"mv"},
}

var g_compressionEnum = [...]string{
	"on", "off", "gzip",
	"gzip-1", "gzip-9",
	"lz4", "lzjb", "zle", "zstd",
	"zstd-1", "zstd-2", "zstd-3", "zstd-4", "zstd-5", "zstd-6", "zstd-7", "zstd-8", "zstd-9", "zstd-10",
	"zstd-11", "zstd-12", "zstd-13", "zstd-14", "zstd-15", "zstd-16", "zstd-17", "zstd-18", "zstd-19",
	"zstd-fast",
	"zstd-fast-1", "zstd-fast-2", "zstd-fast-3", "zstd-fast-4", "zstd-fast-5", "zstd-fast-6", "zstd-fast-7", "zstd-fast-8", "zstd-fast-9",
	"zstd-fast-10", "zstd-fast-20", "zstd-fast-30", "zstd-fast-40", "zstd-fast-50", "zstd-fast-60", "zstd-fast-70", "zstd-fast-80", "zstd-fast-90",
	"zstd-fast-100", "zstd-fast-500", "zstd-fast-1000",
}

var g_datasetCreateUpdateEnums map[string][]string
var g_datasetListEnums map[string][]string

func init() {
	datasetCreateCmd.Run = func(cmd *cobra.Command, args []string) {
		createOrUpdateDataset(cmd, ValidateAndLogin(), args)
	}

	datasetUpdateCmd.Run = func(cmd *cobra.Command, args []string) {
		createOrUpdateDataset(cmd, ValidateAndLogin(), args)
	}

	datasetDeleteCmd.Run = func(cmd *cobra.Command, args []string) {
		deleteDataset(ValidateAndLogin(), args)
	}

	datasetListCmd.Run = func(cmd *cobra.Command, args []string) {
		listDataset(ValidateAndLogin(), args)
	}

	datasetPromoteCmd.Run = func(cmd *cobra.Command, args []string) {
		promoteDataset(ValidateAndLogin(), args)
	}

	datasetRenameCmd.Run = func(cmd *cobra.Command, args []string) {
		renameDataset(ValidateAndLogin(), args)
	}

	createUpdateCmds := []*cobra.Command{datasetCreateCmd, datasetUpdateCmd}
	for _, cmd := range createUpdateCmds {
		cmd.Flags().String("comments", "", "User defined comments")
		cmd.Flags().String("sync", "standard", "Controls the behavior of synchronous requests "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "sync", []string{"standard", "always", "disabled"}))
		cmd.Flags().String("snapdir", "hidden", "Controls whether the .zfs directory is disabled, hidden or visible "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "snapdir", []string{"hidden", "visible"}))
		cmd.Flags().String("compression", "off", "Controls the compression algorithm used for this dataset\n"+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "compression", g_compressionEnum[:]))
		cmd.Flags().String("atime", "off", "Controls whether the access time for files is updated when they are read "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "atime", []string{"on", "off"}))
		cmd.Flags().String("exec", "inherit", "Controls whether processes can be executed from within this file system "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "exec", []string{"inherit", "on", "off"}))
		cmd.Flags().String("managedby", "truenas-admin", "Manager of this dataset, must not be empty")
		cmd.Flags().Bool("quota", false, "")
		//cmd.Flags().Bool("quota_warning", false, "")
		//cmd.Flags().Bool("quota_critical", false, "")
		cmd.Flags().Bool("refquota", false, "")
		//cmd.Flags().Bool("refquota_warning", false, "")
		//cmd.Flags().Bool("refquota_critical", false, "")
		cmd.Flags().Bool("reservation", false, "")
		cmd.Flags().Bool("refreservation", false, "")
		cmd.Flags().Bool("special_small_block_size", false, "")
		cmd.Flags().Bool("copies", false, "")
		cmd.Flags().Bool("deduplication", false, "")
		cmd.Flags().Bool("checksum", false, "")
		cmd.Flags().Bool("readonly", false, "")
		cmd.Flags().Bool("recordsize", false, "")
		cmd.Flags().Bool("casesensitivity", false, "")
		cmd.Flags().Bool("aclmode", false, "")
		cmd.Flags().Bool("acltype", false, "")
		cmd.Flags().Bool("share_type", false, "")
		cmd.Flags().BoolP("create_parents", "p", false, "Creates all the non-existing parent datasets")
		cmd.Flags().String("user_props", "", "Sets the specified properties")
		cmd.Flags().StringP("option", "o", "", "Specify property=value,...")
		cmd.Flags().Int64P("volume", "V", 0, "Creates a volume of the given size instead of a filesystem, should be a multiple of the block size.")
		cmd.Flags().StringP("volblocksize", "b", "512", "Volume block size (\"512\",\"1K\",\"2K\",\"4K\",\"8K\",\"16K\",\"32K\",\"64K\",\"128K\")")
		cmd.Flags().BoolP("sparse", "s", false, "Creates a sparse volume with no reservation")
		cmd.Flags().Bool("force_size", false, "")
		cmd.Flags().String("snapdev", "hidden", "Controls whether the volume snapshot devices are hidden or visible "+
			AddFlagsEnum(&g_datasetCreateUpdateEnums, "snapdev", []string{"hidden", "visible"}))
	}

	g_datasetCreateUpdateEnums["type"] = []string{"volume", "filesystem"}

	datasetDeleteCmd.Flags().BoolP("recursive", "r", false, "Also delete/destroy all children datasets. When the root dataset is specified,\n"+
		"it will destroy all the children of the root dataset present leaving root dataset intact")
	datasetDeleteCmd.Flags().BoolP("force", "f", false, "Force delete busy datasets")

	datasetListCmd.Flags().BoolP("recursive", "r", false, "Retrieves properties for children")
	datasetListCmd.Flags().BoolP("user-properties", "u", false, "Include user-properties")
	datasetListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	datasetListCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	datasetListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_datasetListEnums, "format", []string{"csv", "json", "table", "compact"}))
	datasetListCmd.Flags().StringP("output", "o", "", "Output property list")
	datasetListCmd.Flags().BoolP("all", "a", false, "Output all properties")
	//datasetListCmd.Flags().BoolP("parseable", "p", false, "")
	datasetListCmd.Flags().StringP("source", "s", "default", "A comma-separated list of sources to display.\n"+
		"Those properties coming from a source other than those in this list are ignored.\n"+
		"Each source must be one of the following: local, default, inherited, temporary, received, or none.\n"+
		"The default value is all sources.")

	datasetRenameCmd.Flags().BoolP("update-shares", "s", false, "Will update any shares as part of rename")

	datasetCmd.AddCommand(datasetCreateCmd)
	datasetCmd.AddCommand(datasetUpdateCmd)
	datasetCmd.AddCommand(datasetDeleteCmd)
	datasetCmd.AddCommand(datasetListCmd)
	datasetCmd.AddCommand(datasetPromoteCmd)
	datasetCmd.AddCommand(datasetRenameCmd)
	rootCmd.AddCommand(datasetCmd)
}

func createOrUpdateDataset(cmd *cobra.Command, api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	cmdType := strings.Split(cmd.Use, " ")[0]
	if cmdType != "create" && cmdType != "update" {
		log.Fatal(errors.New("cmdType was not create or update"))
	}

	nameEsc := core.EncloseAndEscape(args[0], "\"")
	nProps := 0

	var builder strings.Builder
	if cmdType == "create" {
		builder.WriteString("[{\"name\":")
		builder.WriteString(nameEsc)
		nProps++
	} else {
		builder.WriteString("[")
		builder.WriteString(nameEsc)
		builder.WriteString(",{")
	}

	shouldCreateParents := false
	wroteCreateParents := false
	var userPropsStr string

	options, err := GetCobraFlags(cmd, g_datasetCreateUpdateEnums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for propName, valueStr := range options.usedFlags {
		isProp := false
		switch propName {
		case "create_parents":
			shouldCreateParents = valueStr == "true"
		case "user_props":
			userPropsStr = valueStr
		case "option":
			paramsKV, err := convertParamsStrToFlatKVArray(valueStr)
			if err != nil {
				log.Fatal(err)
			}
			for j := 0; j < len(paramsKV); j += 2 {
				if nProps > 0 {
					builder.WriteString(",")
				}
				key := paramsKV[j]
				builder.WriteString(key)
				builder.WriteString(":")
				value := paramsKV[j+1]
				builder.WriteString(value)
				nProps++
				if paramsKV[j] == "\"create_ancestors\"" {
					wroteCreateParents = true
				}
			}
		default:
			isProp = true
		}
		if isProp {
			if nProps > 0 {
				builder.WriteString(",")
			}
			core.WriteEncloseAndEscape(&builder, propName, "\"")
			builder.WriteString(":")

			if t, exists := options.allTypes[propName]; exists && t == "string" {
				valueStr = core.EncloseAndEscape(valueStr, "\"")
			}
			builder.WriteString(valueStr)
			nProps++
		}
	}

	if !wroteCreateParents && shouldCreateParents {
		builder.WriteString(",\"create_ancestors\":true")
	}

	if userPropsStr != "" {
		paramsKV, err := convertParamsStrToFlatKVArray(userPropsStr)
		if err != nil {
			log.Fatal(err)
		}
		if nProps > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("user_properties:[")
		for j := 0; j < len(paramsKV); j += 2 {
			if j > 0 {
				builder.WriteString(",")
			}
			builder.WriteString("{\"key\":")
			builder.WriteString(paramsKV[j])
			builder.WriteString(",\"value\":")
			builder.WriteString(paramsKV[j+1])
			builder.WriteString("}")
		}
		builder.WriteString("]")
	}

	builder.WriteString("}]")

	params := builder.String()
	DebugString(params)

	out, err := api.CallString("pool.dataset."+cmdType, "10s", params)
	_ = out
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}
}

func deleteDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, _ := GetCobraFlags(datasetDeleteCmd, nil)
	params := BuildNameStrAndPropertiesJson(options, args[0])
	DebugString(params)

	out, err := api.CallString("pool.dataset.delete", "10s", params)
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}
}

func listDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, err := GetCobraFlags(datasetListCmd, g_datasetListEnums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getDatasetListTypes(args)
	if err != nil {
		log.Fatal(err)
	}

	// `zfs list` will "recurse" if no names are specified.
	extras := typeRetrieveParams{
		retrieveType:      "dataset",
		shouldGetAllProps: format == "json" || core.IsValueTrue(options.allFlags, "all"),
		shouldRecurse:     len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	datasets, err := QueryApi(api, args, idTypes, properties, extras)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	LowerCaseValuesFromEnums(datasets, g_datasetCreateUpdateEnums)

	required := []string{"name"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(datasets, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	core.PrintTableDataList(format, "datasets", columnsList, datasets)
}

func promoteDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	nameEsc := core.EncloseAndEscape(args[0], "\"")
	out, err := api.CallString("pool.dataset.promote", "10s", "["+nameEsc+"]")
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	os.Stdout.WriteString(string(out))
}

func renameDataset(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, _ := GetCobraFlags(datasetRenameCmd, nil)

	source := args[0]
	dest := args[1]

	var builder strings.Builder
	builder.WriteString("[")
	core.WriteEncloseAndEscape(&builder, source, "\"")
	builder.WriteString(",{\"new_name\":")
	core.WriteEncloseAndEscape(&builder, dest, "\"")

	builder.WriteString("}]")
	stmt := builder.String()
	DebugString(stmt)

	out, err := api.CallString("zfs.dataset.rename", "10s", stmt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	if errMsg := core.ExtractApiError(out); errMsg != "" {
		fmt.Fprintln(os.Stderr, "API response error: ", errMsg)
		return
	}

	// no point updating the share if we're renaming a snapshot.
	if core.IsValueTrue(options.allFlags, "update_shares") && !strings.Contains(source, "@") {
		idStr, found, err := LookupNfsIdByPath(api, "/mnt/"+source)
		if err != nil {
			log.Fatal(err)
		}
		if !found {
			fmt.Println("INFO: this dataset did not appear to have a share")
			return
		}

		var nfsBuilder strings.Builder
		nfsBuilder.WriteString("[")
		nfsBuilder.WriteString(idStr)
		nfsBuilder.WriteString(",{\"path\":")
		core.WriteEncloseAndEscape(&nfsBuilder, "/mnt/"+dest, "\"")
		nfsBuilder.WriteString("}]")

		nfsStmt := nfsBuilder.String()
		DebugString(nfsStmt)

		out, err = api.CallString("sharing.nfs.update", "10s", nfsStmt)
		if err != nil {
			log.Fatal(err)
		}

		if errMsg := core.ExtractApiError(out); errMsg != "" {
			fmt.Fprintln(os.Stderr, "API response error: ", errMsg)
			return
		}
	}
}

func convertParamsStrToFlatKVArray(fullParamsStr string) ([]string, error) {
	var array []string
	if fullParamsStr == "" {
		return nil, nil
	}

	array = make([]string, 0, 0)
	params := strings.Split(fullParamsStr, ",")
	for j := 0; j < len(params); j++ {
		parts := strings.Split(params[j], "=")
		var value string
		if len(parts) == 0 {
			continue
		} else if len(parts) == 1 {
			value = "true"
		} else {
			value = parts[1]
		}
		prop := core.EncloseAndEscape(parts[0], "\"")
		if value != "true" && value != "false" && value != "null" {
			_, errNotNumber := strconv.Atoi(value)
			if errNotNumber != nil {
				value = core.EncloseAndEscape(value, "\"")
			}
		}
		array = append(array, prop, value)
	}
	return array, nil
}

func getDatasetListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t, value := core.IdentifyObject(args[i])
		if t == "id" || t == "share" {
			return typeList, errors.New("querying datasets based on mount point is not yet supported")
		} else if t == "snapshot" || t == "snapshot_only" {
			return typeList, errors.New("querying datasets based on shapshot is not yet supported")
		} else if t == "dataset" {
			t = "name"
		}
		typeList[i] = t
		args[i] = value
	}

	return typeList, nil
}

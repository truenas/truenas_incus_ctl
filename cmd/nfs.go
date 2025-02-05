package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

var nfsCmd = &cobra.Command{
	Use:   "nfs",
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

var nfsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a nfs share.",
	Args:  cobra.MinimumNArgs(1),
}

var nfsUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Updates an existing nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
}

var nfsDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var nfsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Prints a table of all nfs shares, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var g_nfsListEnums map[string][]string

func init() {
	nfsCreateCmd.Run = func(cmd *cobra.Command, args []string) {
		createNfs(ValidateAndLogin(), args)
	}

	nfsUpdateCmd.Run = func(cmd *cobra.Command, args []string) {
		updateNfs(ValidateAndLogin(), args)
	}

	nfsDeleteCmd.Run = func(cmd *cobra.Command, args []string) {
		deleteNfs(ValidateAndLogin(), args)
	}

	nfsListCmd.Run = func(cmd *cobra.Command, args []string) {
		listNfs(ValidateAndLogin(), args)
	}

	nfsUpdateCmd.Flags().String("path", "", "Mount path")

	createUpdateCmds := []*cobra.Command{nfsCreateCmd, nfsUpdateCmd}
	for _, cmd := range createUpdateCmds {
		cmd.Flags().Bool("read-only", false, "Export as write protected, default false")
		cmd.Flags().Bool("ro", false, "Equivalent to --read-only=true")
		cmd.Flags().String("comment", "", "")
		cmd.Flags().String("networks", "", "A list of authorized networks that are allowed to access the share "+
			"using CIDR notation. If empty, all networks are allowed")
		cmd.Flags().String("hosts", "", "List of hosts")
		cmd.Flags().String("maproot-user", "", "")
		cmd.Flags().String("maproot-group", "", "")
		cmd.Flags().String("mapall-user", "", "")
		cmd.Flags().String("mapall-group", "", "")
		cmd.Flags().String("security", "", "Array of Enum(sys,krb5,krb5i,krb5p)")
		cmd.Flags().Bool("enabled", false, "")
	}

	nfsListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	nfsListCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	nfsListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" " +
		AddFlagsEnum(&g_nfsListEnums, "format", []string{"csv","json","table","compact"}))
	nfsListCmd.Flags().StringP("output", "o", "", "Output property list")
	nfsListCmd.Flags().BoolP("all", "a", false, "Output all properties")

	nfsCmd.AddCommand(nfsCreateCmd)
	nfsCmd.AddCommand(nfsUpdateCmd)
	nfsCmd.AddCommand(nfsDeleteCmd)
	nfsCmd.AddCommand(nfsListCmd)

	shareCmd.AddCommand(nfsCmd)
}

func createNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	var sharePath string

	switch core.IdentifyObject(args[0]) {
	case "dataset":
		sharePath = "/mnt/" + args[0]
	case "share":
		sharePath = args[0]
	default:
		fmt.Fprintln(os.Stderr, "Unrecognized nfs create spec \"" + args[0] + "\"")
		return
	}

	options, _ := GetCobraFlags(nfsCreateCmd, nil)

	securityList, err := ValidateEnumArray(options.allFlags["security"], []string{"sys","krb5","krb5i","krb5p"})
	if err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString("[{\"path\":")
	core.WriteEncloseAndEscape(&builder, sharePath, "\"")
	builder.WriteString(",")

	writeNfsCreateUpdateProperties(&builder, options, securityList)

	builder.WriteString("}]")

	stmt := builder.String()
	DebugString(stmt)

	out, err := api.CallString("sharing.nfs.create", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func updateNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	var sharePath string
	var idStr string

	switch core.IdentifyObject(args[0]) {
	case "id":
		idStr = args[0]
	case "dataset":
		sharePath = "/mnt/" + args[0]
	case "share":
		sharePath = args[0]
	default:
		fmt.Fprintln(os.Stderr, "Unrecognized nfs create spec \"" + args[0] + "\"")
		return
	}

	options, _ := GetCobraFlags(nfsUpdateCmd, nil)

	securityList, err := ValidateEnumArray(options.allFlags["security"], []string{"sys","krb5","krb5i","krb5p"})
	if err != nil {
		log.Fatal(err)
	}

	if idStr == "" {
		idStr, err = LookupNfsIdByPath(api, sharePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	var builder strings.Builder
	builder.WriteString("[")
	builder.WriteString(idStr)
	builder.WriteString(",{")

	writeNfsCreateUpdateProperties(&builder, options, securityList)

	builder.WriteString("}]")

	stmt := builder.String()
	DebugString(stmt)

	out, err := api.CallString("sharing.nfs.update", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func writeNfsCreateUpdateProperties(builder *strings.Builder, options FlagMap, securityList []string) {
	nProps := 0
	for propName, valueStr := range options.usedFlags {
		if propName == "security" {
			continue
		}
		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(builder, propName, "\"")
		builder.WriteString(":")
		if t, exists := options.allTypes[propName]; exists && t == "string" {
			core.WriteEncloseAndEscape(builder, valueStr, "\"")
		} else {
			builder.WriteString(valueStr)
		}
		nProps++
	}

	if nProps > 0 {
		builder.WriteString(",")
	}
	builder.WriteString("\"security\":[")
	core.WriteJsonStringArray(builder, securityList)
	builder.WriteString("]")
}

func deleteNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	var sharePath string
	var idStr string

	switch core.IdentifyObject(args[0]) {
	case "id":
		idStr = args[0]
	case "dataset":
		sharePath = "/mnt/" + args[0]
	case "share":
		sharePath = args[0]
	default:
		fmt.Fprintln(os.Stderr, "Unrecognized nfs create spec \"" + args[0] + "\"")
		return
	}

	var err error
	if idStr == "" {
		idStr, err = LookupNfsIdByPath(api, sharePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	out, err := api.CallString("sharing.nfs.delete", "10s", "["+idStr+"]")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func listNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, err := GetCobraFlags(nfsListCmd, g_nfsListEnums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		log.Fatal(err)
	}

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getNfsListTypes(args)
	if err != nil {
		log.Fatal(err)
	}

	extras := typeRetrieveParams{
		retrieveType:      "nfs",
		shouldGetAllProps: format == "json" || core.IsValueTrue(options.allFlags, "all"),
		shouldRecurse:     len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	shares, err := QueryApi(api, args, idTypes, properties, extras)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		return
	}

	required := []string{"id", "path"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(shares, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	core.PrintTableDataList(format, "shares", columnsList, shares)
}

func getNfsListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t := core.IdentifyObject(args[i])
		if t == "snapshot" || t == "snapshot_only" {
			return typeList, errors.New("querying nfs shares based on snapshot is not supported")
		} else if t == "dataset" {
			return typeList, errors.New("querying nfs shares based on dataset is not yet supported")
		} else if t == "share" {
			t = "path"
		}
		typeList[i] = t
	}

	return typeList, nil
}

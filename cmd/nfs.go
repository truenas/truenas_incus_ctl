package cmd

import (
	"errors"
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
	Use:   "create [flags]... <dataset|path>",
	Short: "Creates a nfs share.",
	Args:  cobra.MinimumNArgs(1),
}

var nfsUpdateCmd = &cobra.Command{
	Use:     "update [flags]... <id|dataset|path>",
	Short:   "Updates an existing nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
}

var nfsDeleteCmd = &cobra.Command{
	Use:     "delete [flags]... <id|dataset|path>",
	Short:   "Deletes an nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var nfsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Prints a table of all nfs shares, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var g_nfsCreateUpdateEnums map[string][]string
var g_nfsListEnums map[string][]string

func init() {
	nfsCreateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return createNfs(ValidateAndLogin(), args)
	}

	nfsUpdateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return updateNfs(ValidateAndLogin(), args)
	}

	nfsDeleteCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return deleteNfs(ValidateAndLogin(), args)
	}

	nfsListCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return listNfs(ValidateAndLogin(), args)
	}

	nfsUpdateCmd.Flags().String("path", "", "Mount path")

	g_nfsCreateUpdateEnums = make(map[string][]string)

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

	nfsUpdateCmd.Flags().BoolP("create", "c", false, "If the share doesn't exist, create it. Off by default.")

	g_nfsCreateUpdateEnums["security"] = []string{"sys", "krb5", "krb5i", "krb5p"}

	nfsListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	nfsListCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	nfsListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_nfsListEnums, "format", []string{"csv", "json", "table", "compact"}))
	nfsListCmd.Flags().StringP("output", "o", "", "Output property list")
	nfsListCmd.Flags().BoolP("parseable", "p", false, "")
	nfsListCmd.Flags().BoolP("all", "a", false, "Output all properties")

	nfsCmd.AddCommand(nfsCreateCmd)
	nfsCmd.AddCommand(nfsUpdateCmd)
	nfsCmd.AddCommand(nfsDeleteCmd)
	nfsCmd.AddCommand(nfsListCmd)

	shareCmd.AddCommand(nfsCmd)
}

func createNfs(api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	var sharePath string
	typeStr, spec := core.IdentifyObject(args[0])

	switch typeStr {
	case "dataset":
		sharePath = "/mnt/" + spec
	case "share":
		sharePath = spec
	default:
		return errors.New("Unrecognized nfs create spec \""+spec+"\"")
	}

	options, _ := GetCobraFlags(nfsCreateCmd, nil)

	options.usedFlags["path"] = sharePath
	options.allTypes["path"] = "string"

	var builder strings.Builder
	builder.WriteString("[")

	if err := writeNfsCreateUpdateProperties(&builder, options); err != nil {
		return err
	}

	builder.WriteString("]")

	stmt := builder.String()
	DebugString(stmt)

	nfsCreateCmd.SilenceUsage = true

	out, err := core.ApiCallString(api, "sharing.nfs.create", "10s", stmt)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func updateNfs(api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	var sharePath string
	var idStr string
	typeStr, spec := core.IdentifyObject(args[0])

	switch typeStr {
	case "id":
		idStr = spec
	case "dataset":
		sharePath = "/mnt/" + spec
	case "share":
		sharePath = spec
	default:
		return errors.New("Unrecognized nfs update spec \""+spec+"\"")
	}

	options, _ := GetCobraFlags(nfsUpdateCmd, nil)

	shouldCreate := false
	var existingProperties map[string]string

	if idStr == "" {
		var found bool
		var err error
		existingProperties = make(map[string]string)
		idStr, found, err = LookupNfsIdByPath(api, sharePath, existingProperties)
		if err != nil {
			nfsUpdateCmd.SilenceUsage = true
			return err
		}
		if !found {
			if core.IsValueTrue(options.allFlags, "create") {
				shouldCreate = true
			} else {
				nfsUpdateCmd.SilenceUsage = true
				return errors.New("Could not find NFS share \""+sharePath+"\".\n"+
					"Try passing -c to create a share if it doesn't exist.")
			}
		}
	}

	// now that we know whether to create or not, let's not pass this flag on to the API
	delete(options.usedFlags, "create")
	delete(options.allFlags, "create")

	var builder strings.Builder
	builder.WriteString("[")

	if shouldCreate {
		options.usedFlags["path"] = sharePath
		options.allTypes["path"] = "string"
	} else {
		// ideally, we'd examine the props we already retreived when inspecting the id (if we did), and only
		// if there are changes to be made, would we do another update.
		if existingProperties != nil {
			anyDiffs := false
			for key, value := range options.usedFlags {
				if elem, exists := existingProperties[key]; exists {
					var flag string
					if options.allTypes[key] == "string" {
						flag = core.EncloseAndEscape(value, "\"")
					} else {
						flag = value
					}
					if elem != flag {
						anyDiffs = true
						break
					}
				} else {
					// this flag is not an existing property of this nfs share
					anyDiffs = true
					break
				}
			}
			if !anyDiffs {
				DebugString("share does not require updating, exiting")
				return nil
			}
		}
		builder.WriteString(idStr)
		builder.WriteString(",")
	}

	if err := writeNfsCreateUpdateProperties(&builder, options); err != nil {
		return err
	}

	nfsUpdateCmd.SilenceUsage = true

	builder.WriteString("]")

	stmt := builder.String()
	DebugString(stmt)

	var verb string
	if shouldCreate {
		verb = "create"
	} else {
		verb = "update"
	}

	out, err := core.ApiCallString(api, "sharing.nfs."+verb, "10s", stmt)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func writeNfsCreateUpdateProperties(builder *strings.Builder, options FlagMap) error {
	builder.WriteString("{")

	nProps := 0
	for propName, valueStr := range options.usedFlags {
		var securityListStr string
		if propName == "security" {
			securityList, err := ValidateEnumArray(valueStr, []string{"sys", "krb5", "krb5i", "krb5p"})
			if err != nil {
				return err
			}
			var sb strings.Builder
			sb.WriteString("[")
			core.WriteJsonStringArray(&sb, securityList)
			sb.WriteString("]")
			securityListStr = sb.String()
		}
		if propName == "read-only" {
			propName = "ro"
		}

		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(builder, propName, "\"")
		builder.WriteString(":")

		if securityListStr != "" {
			builder.WriteString(securityListStr)
		} else {
			if t, exists := options.allTypes[propName]; exists && t == "string" {
				core.WriteEncloseAndEscape(builder, valueStr, "\"")
			} else {
				builder.WriteString(valueStr)
			}
		}
		nProps++
	}

	builder.WriteString("}")

	return nil
}

func deleteNfs(api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	var sharePath string
	var idStr string
	typeStr, spec := core.IdentifyObject(args[0])

	switch typeStr {
	case "id":
		idStr = spec
	case "dataset":
		sharePath = "/mnt/" + spec
	case "share":
		sharePath = spec
	default:
		return errors.New("Unrecognized nfs create spec \""+spec+"\"")
	}

	nfsDeleteCmd.SilenceUsage = true

	var err error
	if idStr == "" {
		var found bool
		idStr, found, err = LookupNfsIdByPath(api, sharePath, nil)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("Could not find nfs share for path \""+sharePath+"\"")
		}
	}

	out, err := core.ApiCallString(api, "sharing.nfs.delete", "10s", "["+idStr+"]")
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func listNfs(api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	options, err := GetCobraFlags(nfsListCmd, g_nfsListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	nfsListCmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getNfsListTypes(args)
	if err != nil {
		return err
	}

	extras := typeRetrieveParams{
		valueOrder:         BuildValueOrder(core.IsValueTrue(options.allFlags, "parseable")),
		shouldGetAllProps:  format == "json" || core.IsValueTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	shares, err := QueryApi(api, "nfs", args, idTypes, properties, extras)
	if err != nil {
		return err
	}

	LowerCaseValuesFromEnums(shares, g_nfsCreateUpdateEnums)

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
	return nil
}

func getNfsListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t, value := core.IdentifyObject(args[i])
		if t == "snapshot" || t == "snapshot_only" {
			return typeList, errors.New("querying nfs shares based on snapshot is not supported")
		} else if t == "dataset" {
			return typeList, errors.New("querying nfs shares based on dataset is not yet supported")
		} else if t == "share" {
			t = "path"
		}
		typeList[i] = t
		args[i] = value
	}

	return typeList, nil
}

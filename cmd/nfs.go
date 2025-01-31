package cmd

import (
	"fmt"
	"log"
	"strconv"
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

var nfsInspectCmd = &cobra.Command{
	Use:     "inspect",
	Short:   "Prints properties of a nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"get"},
}

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

	nfsInspectCmd.Run = func(cmd *cobra.Command, args []string) {
		inspectNfs(ValidateAndLogin(), args)
	}

	createUpdateCmds := []*cobra.Command{nfsCreateCmd, nfsUpdateCmd}
	for _, cmd := range createUpdateCmds {
		cmd.Flags().Bool("read-only", false, "Export as write protected, default false")
		cmd.Flags().Bool("ro", false, "Equivalent to --read-only=true")
		cmd.Flags().String("comment", "", "")
		cmd.Flags().String("networks", "", "A list of authorized networks that are allowed to access the share " +
			"using CIDR notation. If empty, all networks are allowed")
		cmd.Flags().String("hosts", "", "List of hosts")
		cmd.Flags().String("maproot-user", "", "")
		cmd.Flags().String("maproot-group", "", "")
		cmd.Flags().String("mapall-user", "", "")
		cmd.Flags().String("mapall-group", "", "")
		cmd.Flags().String("security", "", "Array of Enum(sys,krb5,krb5i,krb5p)")
		cmd.Flags().Bool("enabled", false, "")
	}

	listInspectCmds := []*cobra.Command{nfsListCmd, nfsInspectCmd}
	for _, cmd := range listInspectCmds {
		cmd.Flags().String("name", "", "")
		cmd.Flags().Int("id", -1, "")
		cmd.Flags().String("query-filter", "", "")
	}

	nfsCmd.AddCommand(nfsCreateCmd)
	nfsCmd.AddCommand(nfsUpdateCmd)
	nfsCmd.AddCommand(nfsDeleteCmd)
	nfsCmd.AddCommand(nfsListCmd)
	nfsCmd.AddCommand(nfsInspectCmd)

	shareCmd.AddCommand(nfsCmd)
}

func createNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	datasetName := args[0]
	sharePath := "/mnt/" + datasetName

	var builder strings.Builder
	builder.WriteString("[{\"path\":")
	core.WriteEncloseAndEscape(&builder, sharePath, "\"")

	optionsUsed, _, allTypes := getCobraFlags(nfsCreateCmd)
	nProps := 0
	for propName, valueStr := range optionsUsed {
		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(&builder, propName, "\"")
		builder.WriteString(":")
		if t, exists := allTypes[propName]; exists && t == "string" {
			core.WriteEncloseAndEscape(&builder, valueStr, "\"")
		} else {
			builder.WriteString(valueStr)
		}
		nProps++
	}

	builder.WriteString("}]")

	stmt := builder.String()
	fmt.Println(stmt)

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

	idStr := args[0]
	_, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Errorf("ID \"%s\" was not a number", idStr)
	}

	var builder strings.Builder
	builder.WriteString("[")
	builder.WriteString(idStr)
	builder.WriteString(",{")

	optionsUsed, _, allTypes := getCobraFlags(nfsUpdateCmd)
	nProps := 0
	for propName, valueStr := range optionsUsed {
		if nProps > 0 {
			builder.WriteString(",")
		}
		core.WriteEncloseAndEscape(&builder, propName, "\"")
		builder.WriteString(":")
		if t, exists := allTypes[propName]; exists && t == "string" {
			core.WriteEncloseAndEscape(&builder, valueStr, "\"")
		} else {
			builder.WriteString(valueStr)
		}
		nProps++
	}

	builder.WriteString("}]")

	stmt := builder.String()
	fmt.Println(stmt)

	out, err := api.CallString("sharing.nfs.update", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func deleteNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

func listNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	_, allOptions, _ := getCobraFlags(nfsListCmd)

	var builder strings.Builder

	builder.WriteString("[")

	if name := allOptions["name"]; name != "" {
		path := "/mnt/" + name
		builder.WriteString("[[\"path\",\"=\",")
		core.WriteEncloseAndEscape(&builder, path, "\"")
		builder.WriteString("]]")
	} else if id, _ := strconv.Atoi(allOptions["id"]); id >= 0 {
		builder.WriteString("[[\"id\",\"=\",")
		builder.WriteString(fmt.Sprint(id))
		builder.WriteString("]]")
	}

	builder.WriteString("]")

	stmt := builder.String()
	fmt.Println(stmt)

	out, err := api.CallString("sharing.nfs.query", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func inspectNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	//_, allOptions, allTypes := getCobraFlags(nfsInspectCmd)
}

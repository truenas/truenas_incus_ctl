package cmd

import (
	"fmt"
	"log"
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
		createOrUpdateNfs(cmd, ValidateAndLogin(), args)
	}

	nfsUpdateCmd.Run = func(cmd *cobra.Command, args []string) {
		createOrUpdateNfs(cmd, ValidateAndLogin(), args)
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

	nfsCmd.AddCommand(nfsCreateCmd)
	nfsCmd.AddCommand(nfsUpdateCmd)
	nfsCmd.AddCommand(nfsDeleteCmd)
	nfsCmd.AddCommand(nfsListCmd)
	nfsCmd.AddCommand(nfsInspectCmd)

	shareCmd.AddCommand(nfsCmd)
}

func createOrUpdateNfs(cmd *cobra.Command, api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	datasetName := args[0]
	sharePath := "/mnt/" + datasetName

	var builder strings.Builder
	builder.WriteString("[{\"path\":")
	core.WriteEncloseAndEscape(&builder, sharePath, "\"")
	builder.WriteString("}]")

	query := builder.String()
	fmt.Println(query)

	out, err := api.CallString("sharing.nfs.create", "10s", query)
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
}

func inspectNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()
}

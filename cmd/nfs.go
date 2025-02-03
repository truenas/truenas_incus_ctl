package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
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

var g_nfsListInspectEnums map[string][]string

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
		cmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
		cmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
		cmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" " +
			AddFlagsEnum(&g_nfsListInspectEnums, "format", []string{"csv","json","table","compact"}))
		cmd.Flags().StringP("output", "o", "", "Output property list")
		cmd.Flags().BoolP("all", "a", false, "Output all properties")
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

	options, _ := GetCobraFlags(nfsCreateCmd, nil)

	securityList, err := ValidateEnumArray(options.allFlags["security"], []string{"sys","krb5","krb5i","krb5p"})
	if err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString("[{\"path\":")
	core.WriteEncloseAndEscape(&builder, sharePath, "\"")
	builder.WriteString(",")

	writeCreateUpdateProperties(&builder, options, securityList)

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
		log.Fatal(fmt.Errorf("ID \"%s\" was not a number", idStr))
	}

	options, _ := GetCobraFlags(nfsUpdateCmd, nil)

	securityList, err := ValidateEnumArray(options.allFlags["security"], []string{"sys","krb5","krb5i","krb5p"})
	if err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString("[")
	builder.WriteString(idStr)
	builder.WriteString(",{")

	writeCreateUpdateProperties(&builder, options, securityList)

	builder.WriteString("}]")

	stmt := builder.String()
	fmt.Println(stmt)

	out, err := api.CallString("sharing.nfs.update", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

func writeCreateUpdateProperties(builder *strings.Builder, options FlagMap, securityList []string) {
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

	idStr := args[0]
	_, err := strconv.Atoi(idStr)
	if err != nil {
		log.Fatal(fmt.Errorf("ID \"%s\" was not a number", idStr))
	}

	out, err := api.CallString("sharing.nfs.delete", "10s", "[" + idStr + "]")
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

	options, err := GetCobraFlags(nfsListCmd, g_nfsListInspectEnums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		log.Fatal(err)
	}

	stmt := makeNfsQueryStatement(options.allFlags)
	fmt.Println(stmt)

	out, err := api.CallString("sharing.nfs.query", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}

	shares, err := unpackNfsQuery(out)
	if err != nil {
		log.Fatal(err)
	}

	required := []string{"id", "path"}
	var columnsList []string
	if all, _ := options.allFlags["all"]; all == "true" {
		columnsList = GetUsedPropertyColumns(shares, required)
	} else {
		outputCols := options.allFlags["output"]
		var specList []string
		if outputCols != "" {
			specList = strings.Split(outputCols, ",")
		}
		columnsList = MakePropertyColumns(required, specList)
	}

	core.PrintTableData(format, "shares", columnsList, shares)
}

func inspectNfs(api core.Session, args []string) {
	if api == nil {
		return
	}
	defer api.Close()

	options, err := GetCobraFlags(nfsListCmd, g_nfsListInspectEnums)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		log.Fatal(err)
	}

	if _, errNotNumber := strconv.Atoi(args[0]); errNotNumber == nil {
		options.allFlags["id"] = args[0]
	} else {
		options.allFlags["name"] = args[0]
	}

	stmt := makeNfsQueryStatement(options.allFlags)
	fmt.Println(stmt)

	out, err := api.CallString("sharing.nfs.query", "10s", stmt)
	if err != nil {
		log.Fatal(err)
	}

	shares, err := unpackNfsQuery(out)
	if err != nil {
		log.Fatal(err)
	}

	required := []string{"id", "path"}
	outputCols := options.allFlags["output"]

	var columnsList []string
	if outputCols != "" {
		columnsList = MakePropertyColumns(required, strings.Split(outputCols, ","))
	} else {
		columnsList = GetUsedPropertyColumns(shares, required)
	}

	core.PrintTableData(format, "shares", columnsList, shares)
}

func makeNfsQueryStatement(allOptions map[string]string) string {
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
	} else {
		builder.WriteString("[]")
	}

	//builder.WriteString(",{\"extra\":{\"properties\":[\"ro\"]}}")

	builder.WriteString("]")
	return builder.String()
}

func unpackNfsQuery(data json.RawMessage) ([]map[string]interface{}, error) {
	var response interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("response error: %v", err)
	}

	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, errors.New("API response was not a JSON object")
	}

	resultsList, errMsg := core.ExtractJsonArrayOfMaps(responseMap, "result")
	if errMsg != "" {
		return nil, errors.New("API response results: " + errMsg)
	}

	return resultsList, nil
}

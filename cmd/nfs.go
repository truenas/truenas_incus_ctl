package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var nfsCmd = &cobra.Command{
	Use:   "nfs",
	Short: "Create, list, update or delete NFS shares",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var nfsCreateCmd = &cobra.Command{
	Use:   "create <dataset|path>...",
	Short: "Creates a nfs share.",
	Args:  cobra.MinimumNArgs(1),
}

var nfsUpdateCmd = &cobra.Command{
	Use:     "update <id|dataset|path>...",
	Short:   "Updates an existing nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"set"},
}

var nfsDeleteCmd = &cobra.Command{
	Use:     "delete <id|dataset|path>...",
	Short:   "Deletes an nfs share.",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

var nfsListCmd = &cobra.Command{
	Use:     "list [id|dataset|path]...",
	Short:   "Prints a table of all nfs shares, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var g_nfsCreateUpdateEnums map[string][]string
var g_nfsListEnums map[string][]string

func init() {
	nfsCreateCmd.RunE = WrapCommandFunc(createNfs)
	nfsUpdateCmd.RunE = WrapCommandFunc(updateNfs)
	nfsDeleteCmd.RunE = WrapCommandFunc(deleteNfs)
	nfsListCmd.RunE = WrapCommandFunc(listNfs)

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

	nfsUpdateCmd.Flags().BoolP("create", "c", false, "If a share doesn't exist, create it. Off by default.")

	g_nfsCreateUpdateEnums["security"] = []string{"sys", "krb5", "krb5i", "krb5p"}

	nfsListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	nfsListCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	nfsListCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_nfsListEnums, "format", []string{"csv", "json", "table", "compact"}))
	nfsListCmd.Flags().StringP("output", "o", "", "Output property list")
	nfsListCmd.Flags().BoolP("parseable", "p", false, "Show raw values instead of the already parsed values")
	nfsListCmd.Flags().BoolP("all", "a", false, "Output all properties")

	nfsCmd.AddCommand(nfsCreateCmd)
	nfsCmd.AddCommand(nfsUpdateCmd)
	nfsCmd.AddCommand(nfsDeleteCmd)
	nfsCmd.AddCommand(nfsListCmd)

	shareCmd.AddCommand(nfsCmd)
}

type typeNfsSpecs struct {
	paths []string
	idList []string
	specs []string
	types []string
	existsMap map[string]int
}

func createNfs(cmd *cobra.Command, api core.Session, args []string) error {
	paths := make([]string, 0)
	for i := 0; i < len(args); i++ {
		typeStr, spec := core.IdentifyObject(args[0])

		switch typeStr {
		case "dataset":
			paths = append(paths, "/mnt/"+spec)
		case "share":
			paths = append(paths, spec)
		default:
			return errors.New("Unrecognized nfs create spec \"" + spec + "\"")
		}
	}

	options, _ := GetCobraFlags(cmd, nil)

	options.usedFlags["path"] = paths[0]
	options.allTypes["path"] = "string"

	propsMap, err := writeNfsCreateUpdateProperties(options)
	if err != nil {
		return err
	}

	params := []interface{}{propsMap}

	cmd.SilenceUsage = true

	objRemap := map[string][]interface{}{"path": core.ToAnyArray(paths)}
	out, err := MaybeBulkApiCall(api, "sharing.nfs.create", 10, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func updateNfs(cmd *cobra.Command, api core.Session, args []string) error {
	specs, err := getIdAndPathLists(args)
	if err != nil {
		return err
	}

	options, _ := GetCobraFlags(cmd, nil)
	flagCreate := core.IsValueTrue(options.allFlags, "create")

	// now that we know whether to create or not, let's not pass this flag on to the API
	delete(options.usedFlags, "create")
	delete(options.allFlags, "create")

	propsMap, err := writeNfsCreateUpdateProperties(options)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  true,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}
	response, err := QueryApi(api, "nfs", specs.specs, specs.types, nil, extras)
	if err != nil {
		return err
	}

	foundIds := make(map[string]string)
	foundPaths := make(map[string]string)
	for _, r := range response.resultsMap {
		id := fmt.Sprint(r["id"])
		path := fmt.Sprint(r["path"])
		foundIds[id] = path
		foundPaths[path] = id
	}

	// list of ids not found in response: if not empty, then error
	for _, id := range specs.idList {
		if _, exists := foundIds[id]; !exists {
			return fmt.Errorf("Could not update nfs share with ID %s: did not exist", id)
		}
	}

	// list of paths not found in response: if not empty, and no --create, then error
	listToCreate := make([]string, 0)
	listToUpdate := make([]int, 0)
	for _, p := range specs.paths {
		if idStr, exists := foundPaths[p]; exists {
			anyDiffs := false
			props := response.resultsMap[idStr]
			for key, value := range options.usedFlags {
				if elem, exists := props[key]; exists {
					if elem != value {
						anyDiffs = true
						break
					}
				} else {
					// this flag is not an existing property of this nfs share
					anyDiffs = true
					break
				}
			}
			if anyDiffs {
				id, _ := strconv.Atoi(idStr)
				listToUpdate = append(listToUpdate, id)
			}
		} else {
			if !flagCreate {
				return errors.New("Could not find NFS share \"" + p + "\".\n" +
					"Try passing -c or --create to create a share if it doesn't exist.")
			}
			listToCreate = append(listToCreate, p)
		}
	}

	params := []interface{}{propsMap}

	if len(listToUpdate) > 0 {
		objRemap := map[string][]interface{}{"": core.ToAnyArray(listToUpdate)}
		out, err := MaybeBulkApiCall(api, "sharing.nfs.update", 10, params, objRemap, false)
		if err != nil {
			return err
		}
		DebugString(string(out))
	} else {
		DebugString("No NFS shares required updating")
	}

	if len(listToCreate) > 0 {
		objRemap := map[string][]interface{}{"path": core.ToAnyArray(listToCreate)}
		out, err := MaybeBulkApiCall(api, "sharing.nfs.create", 10, params, objRemap, false)
		if err != nil {
			return err
		}
		DebugString(string(out))
	}

	return nil
}

func writeNfsCreateUpdateProperties(options FlagMap) (map[string]interface{}, error) {
	outMap := make(map[string]interface{})
	for propName, valueStr := range options.usedFlags {
		if propName == "security" {
			securityList, err := ValidateEnumArray(valueStr, []string{"sys", "krb5", "krb5i", "krb5p"})
			if err != nil {
				return nil, err
			}
			if securityList == nil {
				securityList = make([]string, 0)
			}
			outMap["security"] = securityList
		} else {
			if propName == "read-only" {
				propName = "ro"
			}
			value, err := ParseStringAndValidate(propName, valueStr, nil)
			if err != nil {
				return nil, err
			}
			outMap[propName] = value
		}
	}
	return outMap, nil
}

func deleteNfs(cmd *cobra.Command, api core.Session, args []string) error {
	specs, err := getIdAndPathLists(args)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	if len(specs.idList) == len(specs.specs) {
		idListInts := make([]int, len(specs.idList))
		for i, idStr := range specs.idList {
			idListInts[i], _ = strconv.Atoi(idStr)
		}
		params := []interface{}{idListInts[0]}
		DebugJson(params)

		objRemap := map[string][]interface{}{"": core.ToAnyArray(idListInts)}
		_, err := MaybeBulkApiCall(api, "sharing.nfs.delete", 10, params, objRemap, false)
		return err
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  true,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}
	response, err := QueryApi(api, "nfs", specs.specs, specs.types, nil, extras)
	if err != nil {
		return err
	}

	responseIdList := make([]interface{}, len(specs.specs))
	for _, r := range response.resultsMap {
		idStr := fmt.Sprint(r["id"])
		path := fmt.Sprint(r["path"])
		idx := -1
		if index, ok := specs.existsMap[path]; ok {
			idx = index
		} else if index, ok := specs.existsMap[idStr]; ok {
			idx = index
		}
		if idx < 0 {
			return fmt.Errorf("Could not find %s or %s in API response", idStr, path)
		}
		if n, errNotNumber := strconv.Atoi(idStr); errNotNumber == nil {
			responseIdList[idx] = n
		} else {
			responseIdList[idx] = idStr
		}
	}

	if len(responseIdList) == 0 {
		DebugString("No NFS shares were deleted")
		return nil
	}

	params := []interface{}{responseIdList[0]}
	objRemap := map[string][]interface{}{"": responseIdList}
	out, err := MaybeBulkApiCall(api, "sharing.nfs.delete", 10, params, objRemap, false)
	if err != nil {
		return err
	}

	DebugString(string(out))
	return nil
}

func getIdAndPathLists(args []string) (typeNfsSpecs, error) {
	s := typeNfsSpecs{}
	s.paths = make([]string, 0)
	s.idList = make([]string, 0)
	s.specs = make([]string, 0)
	s.types = make([]string, 0)
	s.existsMap = make(map[string]int)

	for i := 0; i < len(args); i++ {
		typeStr, spec := core.IdentifyObject(args[i])
		switch typeStr {
		case "id":
			if _, err := strconv.Atoi(spec); err != nil {
				return s, err
			}
			s.idList = append(s.idList, spec)
			s.specs = append(s.specs, spec)
			s.types = append(s.types, "id")
		case "dataset":
			p := "/mnt/" + spec
			s.paths = append(s.paths, p)
			s.specs = append(s.specs, p)
			s.types = append(s.types, "path")
		case "share":
			s.paths = append(s.paths, spec)
			s.specs = append(s.specs, spec)
			s.types = append(s.types, "path")
		default:
			return s, errors.New("Unrecognized NFS spec \"" + spec + "\"")
		}
	}

	if len(s.specs) == 0 {
		return s, errors.New("No valid NFS specs were found")
	}

	for i, spec := range s.specs {
		s.existsMap[spec] = i
	}

	return s, nil
}

func listNfs(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, g_nfsListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)
	idTypes, err := getNfsListTypes(args)
	if err != nil {
		return err
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsValueTrue(options.allFlags, "parseable")),
		shouldGetAllProps:  core.IsValueTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	response, err := QueryApi(api, "nfs", args, idTypes, properties, extras)
	if err != nil {
		return err
	}

	shares := GetListFromQueryResponse(&response)
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

	str, err := core.BuildTableData(format, "shares", columnsList, shares)
	PrintTable(api, str)
	return err
}

func getNfsListTypes(args []string) ([]string, error) {
	var typeList []string
	if len(args) == 0 {
		return typeList, nil
	}

	typeList = make([]string, len(args), len(args))
	for i := 0; i < len(args); i++ {
		t, value := core.IdentifyObject(args[i])
		if t == "snapshot_only" {
			return nil, errors.New("querying nfs shares based on snapshot is not supported")
		} else if t == "snapshot" {
			value = "/mnt/" + value[0:strings.Index(value, "@")]
			t = "path"
		} else if t == "dataset" {
			value = "/mnt/" + value
			t = "path"
		} else if t == "share" {
			t = "path"
		} else if t != "id" && t != "pool" {
			return nil, errors.New("Unrecognised namespec \"" + args[i] + "\"")
		}
		typeList[i] = t
		args[i] = value
	}

	return typeList, nil
}

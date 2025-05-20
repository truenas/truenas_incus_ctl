package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var iscsiCmd = &cobra.Command{
	Use:   "iscsi",
	Short: "Manage iSCSI connections",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var iscsiCreateCmd = &cobra.Command{
	Use:   "create <dataset>...",
	Short: "Create iscsi extents and targets that map to the given datasets",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiActivateCmd = &cobra.Command{
	Use:   "activate <dataset>...",
	Short: "Activate the iscsi targets that map to the given datasets",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiListCmd = &cobra.Command{
	Use:   "list",
	Short: "List description",
}

var iscsiLocateCmd = &cobra.Command{
	Use:   "locate <dataset>...",
	Short: "Locate the iscsi targets that map to the given datasets",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeactivateCmd = &cobra.Command{
	Use:   "deactivate <dataset>...",
	Short: "Deactivate the iscsi targets that map to the given datasets",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeleteCmd = &cobra.Command{
	Use:     "delete <dataset>...",
	Short:   "Delete the iscsi targets that map to the given datasets, after deactivating them first",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	iscsiCreateCmd.RunE = WrapCommandFunc(createIscsi)
	iscsiActivateCmd.RunE = WrapCommandFunc(activateIscsi)
	iscsiListCmd.RunE = WrapCommandFunc(listIscsi)
	iscsiLocateCmd.RunE = WrapCommandFunc(locateIscsi)
	iscsiDeactivateCmd.RunE = WrapCommandFunc(deactivateIscsi)
	iscsiDeleteCmd.RunE = WrapCommandFunc(deleteIscsi)

	iscsiCreateCmd.Flags().Bool("readonly", false, "Ensure the new iSCSI extent is read-only. Ignored for snapshots.")

	iscsiLocateCmd.Flags().Bool("activate", false, "Activate any shares that could not be located")
	iscsiLocateCmd.Flags().Bool("deactivate", false, "Deactivate any shares that could not be located")

	_iscsiCmds := []*cobra.Command {iscsiCreateCmd, iscsiActivateCmd, iscsiLocateCmd, iscsiDeactivateCmd, iscsiDeleteCmd}
	for _, c := range _iscsiCmds {
		c.Flags().StringP("target-prefix", "t", "", "label to prefix the created target")
		c.Flags().IntP("iscsi-port", "p", 3260, "iSCSI portal port")
	}

	_iscsiAdminCmds := []*cobra.Command {iscsiActivateCmd, iscsiLocateCmd, iscsiDeactivateCmd}
	for _, c := range _iscsiAdminCmds {
		c.Flags().Bool("parsable", false, "Parsable (ie. minimal) output")
	}

	iscsiCmd.AddCommand(iscsiCreateCmd)
	iscsiCmd.AddCommand(iscsiActivateCmd)
	iscsiCmd.AddCommand(iscsiListCmd)
	iscsiCmd.AddCommand(iscsiLocateCmd)
	iscsiCmd.AddCommand(iscsiDeactivateCmd)
	iscsiCmd.AddCommand(iscsiDeleteCmd)

	shareCmd.AddCommand(iscsiCmd)
}

type typeIscsiTargetParams struct {
	verb        string
	id          interface{}
	portalId    int
	initiatorId int
}

func createIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, false, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)
	cmd.SilenceUsage = true

	changes := make([]typeApiCallRecord, 0)
	defer undoIscsiCreateList(api, &changes)

	maybeHashedToVolumeMap := make(map[string]string)
	volumeToMaybeHashedMap := make(map[string]string)
	for _, vol := range args {
		hashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		if _, exists := maybeHashedToVolumeMap[hashed]; exists {
			return fmt.Errorf("There are duplicates in the provided list of datasets")
		}
		maybeHashedToVolumeMap[hashed] = vol
		volumeToMaybeHashedMap[vol] = hashed
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  true,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}
	responseTargetQuery, err := QueryApi(api, "iscsi.target", args, core.StringRepeated("alias", len(args)), nil, extras)
	if err != nil {
		return err
	}

	toCreateMap := make(map[string]bool)
	for _, t := range args {
		toCreateMap[t] = true
	}

	//missingInitiators := make(map[string]bool)
	targets := make(map[string]typeIscsiTargetParams)
	shouldFindPortal := false

	for targetId, target := range responseTargetQuery.resultsMap {
		targetName, _ := target["alias"].(string)
		if targetName == "" {
			return fmt.Errorf("Name could not be found in iSCSI target with ID %v", targetId)
		}
		delete(toCreateMap, targetName)

		anyGroups := false
		if groupsObj, exists := target["groups"]; exists {
			if groups, ok := groupsObj.([]interface{}); ok && len(groups) > 0 {
				anyGroups = true
				portalExists := false
				initiatorExists := false
				if elem, isElemMap := groups[0].(map[string]interface{}); isElemMap {
					_, portalExists = elem["portal"].(float64)
					_, initiatorExists = elem["initiator"].(float64)
				}

				portal := 1    // -1
				initiator := 1 // -1
				if portalExists {
					portal = 1 // 0
				} else {
					shouldFindPortal = true
				}
				if initiatorExists {
					initiator = 1 // 0
				} else {
					//missingInitiators[targetName] = true
				}

				targets[targetName] = typeIscsiTargetParams{
					verb:        "update",
					id:          targetId,
					portalId:    portal,
					initiatorId: initiator,
				}
			}
		}
		if !anyGroups {
			shouldFindPortal = true
			//missingInitiators[targetName] = true

			targets[targetName] = typeIscsiTargetParams{
				verb:        "update",
				id:          targetId,
				portalId:    1, // -1
				initiatorId: 1, // -1
			}
		}
	}

	for targetName, _ := range toCreateMap {
		shouldFindPortal = true
		//missingInitiators[targetName] = true

		targets[targetName] = typeIscsiTargetParams{
			verb:        "create",
			id:          -1,
			portalId:    1, // -1
			initiatorId: 1, // -1
		}
	}

	if len(targets) == 0 {
		fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		return nil
	}

	emptyQueryParams := []interface{}{make([]interface{}, 0), make(map[string]interface{})}

	defaultPortal := -1
	if shouldFindPortal {
		out, err := core.ApiCall(api, "iscsi.portal.query", 10, emptyQueryParams)
		if err != nil {
			return err
		}
		var response map[string]interface{}
		if err = json.Unmarshal(out, &response); err != nil {
			return err
		}
		results, _ := response["result"].([]interface{})
		for i := 0; i < len(results); i++ {
			if obj, ok := results[i].(map[string]interface{}); ok {
				if idObj, exists := obj["id"]; exists {
					if id, ok := idObj.(float64); ok {
						defaultPortal = int(id)
						break
					}
				}
			}
		}
		if defaultPortal == -1 {
			cmd.SilenceUsage = false
			return fmt.Errorf("No iSCSI portal was found for this host. Use:\n" +
				"<truenas_incus_ctl> share iscsi portal create --ip <IP address> --port <port number>\n" +
				"To create one.\n")
		}
	}

	/*
		if len(missingInitiators) > 0 {
			//...
		}
	*/

	targetCreates := make([]interface{}, 0)
	targetUpdates := make([]interface{}, 0)
	for volName, t := range targets {
		group := make(map[string]interface{})
		isGroupEmpty := true
		if t.portalId != 0 {
			pid := t.portalId
			if pid < 0 {
				pid = defaultPortal
			}
			group["portal"] = pid
			isGroupEmpty = false
		}
		if t.initiatorId > 0 {
			group["initiator"] = t.initiatorId
			isGroupEmpty = false
		}

		obj := make(map[string]interface{})
		obj["name"] = volumeToMaybeHashedMap[volName]
		obj["alias"] = volName

		if !isGroupEmpty {
			obj["groups"] = []map[string]interface{}{group}
		}

		if t.verb == "create" {
			targetCreates = append(targetCreates, []interface{}{obj})
		} else {
			if id, errNotNumber := strconv.Atoi(fmt.Sprint(t.id)); errNotNumber == nil {
				t.id = id
			}
			targetUpdates = append(targetUpdates, []interface{}{t.id, obj})
		}
	}

	jobIdUpdate := int64(-1)
	jobIdCreate := int64(-1)
	var rawResultsTargetUpdate json.RawMessage
	var resultsTargetCreate []interface{}
	var errorsTargetCreate []interface{}

	if len(targetUpdates) > 0 {
		rawResultsTargetUpdate, jobIdUpdate, err = MaybeBulkApiCallArray(api, "iscsi.target.update", 10, targetUpdates, false)
		if err != nil {
			return err
		}
	}
	if len(targetCreates) > 0 {
		rawResultsTargetCreate, _, err := MaybeBulkApiCallArray(api, "iscsi.target.create", 10, targetCreates, true)
		if err != nil {
			return err
		}
		resultsTargetCreate, errorsTargetCreate = core.GetResultsAndErrorsFromApiResponseRaw(rawResultsTargetCreate)
		changes = append(changes, typeApiCallRecord {
			endpoint: "iscsi.target.create",
			params: targetCreates,
			resultList: resultsTargetCreate,
			errorList: errorsTargetCreate,
		})
	}

	if jobIdUpdate >= 0 {
		rawResultsTargetUpdate, err = api.WaitForJob(jobIdUpdate)
		if err != nil {
			return err
		}
	}
	/*
	if jobIdCreate >= 0 {
		rawResultsTargetCreate, err = api.WaitForJob(jobIdCreate)
		if err != nil {
			return err
		}
	}
	*/

	jobIdCreate = jobIdCreate
	rawResultsTargetUpdate = rawResultsTargetUpdate

	allTargets := GetListFromQueryResponse(&responseTargetQuery)
	for _, t := range resultsTargetCreate {
		if tMap, ok := t.(map[string]interface{}); ok {
			allTargets = append(allTargets, tMap)
		}
	}

	extentList := make([]string, len(args))
	for i, vol := range args {
		extentList[i] = "zvol/" + vol
	}

	responseExtentQuery, err := QueryApi(
		api,
		"iscsi.extent",
		extentList,
		core.StringRepeated("disk", len(extentList)),
		nil,
		extras,
	)
	if err != nil {
		return err
	}

	extentsByDisk := GetMapFromQueryResponseKeyedOn(&responseExtentQuery, "disk")

	extentsCreate := make([]string, 0)
	extentsIqnCreate := make([]string, 0)
	for _, vol := range args {
		if _, exists := extentsByDisk["zvol/" + vol]; !exists {
			extentsCreate = append(extentsCreate, "zvol/" + vol)
			extentsIqnCreate = append(extentsIqnCreate, volumeToMaybeHashedMap[vol])
		}
	}

	if len(extentsCreate) > 0 {
		isReadOnly := core.IsStringTrue(options.allFlags, "readonly")

		paramsCreate := make([]interface{}, len(extentsCreate))
		for i, _ := range extentsCreate {
			snapOffset := strings.Index(extentsCreate[i], "@")
			paramsCreate[i] = []interface{} {
				map[string]interface{} {
					"name": extentsIqnCreate[i],
					"disk": extentsCreate[i],
					"ro": isReadOnly || snapOffset >= 0,
				},
			}
		}
		out, _, err := MaybeBulkApiCallArray(
			api,
			"iscsi.extent.create",
			10,
			paramsCreate,
			true,
		)
		if err != nil {
			return err
		}

		resultsExtentCreate, errorsExtentCreate := core.GetResultsAndErrorsFromApiResponseRaw(out)
		changes = append(changes, typeApiCallRecord {
			endpoint: "iscsi.extent.create",
			params: paramsCreate,
			resultList: resultsExtentCreate,
			errorList: errorsExtentCreate,
		})

		for _, extent := range resultsExtentCreate {
			if extentMap, ok := extent.(map[string]interface{}); ok {
				extentsByDisk[fmt.Sprint(extentMap["disk"])] = extentMap
			}
		}
	}

	teCreateMap := make(map[string]map[string]interface{})
	for _, target := range allTargets {
		vol, _ := target["alias"].(string)
		if vol == "" {
			continue
		}
		if extent, exists := extentsByDisk["zvol/"+vol]; exists {
			key := fmt.Sprintf("%v-%v", target["id"], extent["id"])
			teCreateMap[key] = map[string]interface{}{
				"target": target["id"],
				"lunid":  0,
				"extent": extent["id"],
			}
		}
	}

	responseTeQuery, err := QueryApi(api, "iscsi.targetextent", nil, nil, nil, extras)
	if err != nil {
		return err
	}
	for _, te := range responseTeQuery.resultsMap {
		key := fmt.Sprintf("%v-%v", te["target"], te["extent"])
		delete(teCreateMap, key)
	}

	teCreateList := make([]interface{}, 0)
	for _, te := range teCreateMap {
		teCreateList = append(teCreateList, []interface{}{te})
	}

	if len(teCreateList) > 0 {
		_, _, err = MaybeBulkApiCallArray(api, "iscsi.targetextent.create", 10, teCreateList, true)
		if err != nil {
			return err
		}
	}

	changes = make([]typeApiCallRecord, 0)
	return nil
}

func undoIscsiCreateList(api core.Session, changes *[]typeApiCallRecord) {
	DebugString("undoIscsiCreateList")
	for _, call := range *changes {
		DebugString(call.endpoint)
		if strings.HasSuffix(call.endpoint, ".create") {
			idList := make([]interface{}, 0)
			for _, r := range call.resultList {
				idList = append(idList, []interface{}{core.GetIdFromObject(r)})
			}
			if len(idList) > 0 {
				MaybeBulkApiCallArray(
					api,
					call.endpoint[:len(call.endpoint)-7] + ".delete",
					10,
					idList,
					false,
				)
			}
		}
	}
}

func listIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	IterateActivatedIscsiShares("", func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
		fullPath := path.Join(root, fullName)
		fmt.Println(fullPath)
	})
	return nil
}

func getIscsiSharesFromSessionAndDiscovery(
	options FlagMap,
	api core.Session,
	args []string,
	hostname string,
	isActivate bool,
	isDeactivate bool,
) ([]typeIscsiLoginSpec, map[string]bool, error) {
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	maybeHashedToVolumeMap := make(map[string]string)
	for _, vol := range args {
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
	}

	var err error
	if err = CheckIscsiAdminToolExists(); err != nil {
		return nil, nil, err
	}

	err = MaybeLaunchIscsiDaemon()
	if err != nil {
		return nil, nil, err
	}

	var targets []typeIscsiLoginSpec

	if !isActivate {
		targets, _ = GetIscsiTargetsFromSession(maybeHashedToVolumeMap)
	}

	if !isDeactivate && len(targets) == 0 {
		portalAddr := hostname + ":" + options.allFlags["iscsi_port"]
		targets, err = GetIscsiTargetsFromDiscovery(maybeHashedToVolumeMap, portalAddr)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(targets) == 0 {
		var notFoundErr error
		if !core.IsStringTrue(options.allFlags, "parsable") {
			notFoundErr = fmt.Errorf("Could not find any matching iscsi shares")
		}
		return nil, nil, notFoundErr
	}

	shares := make(map[string]bool)
	for _, t := range targets {
		shares[t.iqn + ":" + t.target] = true
	}

	return targets, shares, nil
}

func locateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true
	options, _ := GetCobraFlags(cmd, false, nil)
	shouldActivate := core.IsStringTrue(options.allFlags, "activate")
	shouldDeactivate := core.IsStringTrue(options.allFlags, "deactivate")

	/* TODO: ensure the two can be used together, then remove this check
	if shouldActivate && shouldDeactivate {
		return fmt.Errorf("--activate and --deactivate options are incompatible")
	}
	*/

	hostname := core.StripPort(api.GetHostName())
	targets, shares, err := getIscsiSharesFromSessionAndDiscovery(options, api, args, hostname, shouldActivate, shouldDeactivate)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	ipAddrs, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	ipPortalAddr := ipAddrs[0].String() + ":" + options.allFlags["iscsi_port"]
	isMinimal := core.IsStringTrue(options.allFlags, "parsable")

	toDeactivate := make([]string, 0)

	anyLocated := false
	IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipAddr string, iqnTargetName string, targetOnlyName string) {
		if _, exists := shares[iqnTargetName]; !exists {
			return
		}
		anyLocated = true
		if shouldDeactivate {
			toDeactivate = append(toDeactivate, iqnTargetName)
		} else {
			fullPath := path.Join(root, fullName)
			if shouldActivate {
				fmt.Println("located\t" + fullPath)
			} else {
				fmt.Println(fullPath)
			}
		}
		delete(shares, iqnTargetName)
	})

	if shouldDeactivate {
		for _, t := range toDeactivate {
			logoutParams := []string{
				"--mode",
				"node",
				"--targetname",
				t,
				"--portal",
				ipPortalAddr,
				"--logout",
			}
			DebugString(strings.Join(logoutParams, " "))
			_, err := RunIscsiAdminTool(logoutParams)
			if err != nil {
				fmt.Println("failed\t" + t)
			} else {
				fmt.Println("deactivated\t" + t)
			}
		}
		for t, _ := range shares {
			fmt.Println("not-found\t" + t)
		}
	} else if shouldActivate && len(shares) > 0 {
		remainingTargets := make([]typeIscsiLoginSpec, 0)
		for _, t := range targets {
			if _, exists := shares[t.iqn + ":" + t.target]; exists {
				remainingTargets = append(remainingTargets, t)
			}
		}
		return doIscsiActivate(remainingTargets, ipPortalAddr, isMinimal, true)
	} else if !isMinimal && !anyLocated {
		fmt.Println("No matching iscsi shares were found")
	}
	return nil
}

type typeIscsiPathAndIqnTarget struct {
	fullPath string
	iqnTargetName string
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true
	hostname := core.StripPort(api.GetHostName())

	options, _ := GetCobraFlags(cmd, false, nil)
	targets, _, err := getIscsiSharesFromSessionAndDiscovery(options, api, args, hostname, true, false)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	ipAddrs, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	isMinimal := core.IsStringTrue(options.allFlags, "parsable")
	ipAddr := ipAddrs[0].String() + ":" + options.allFlags["iscsi_port"]

	return doIscsiActivate(targets, ipAddr, isMinimal, false)
}

func doIscsiActivate(targets []typeIscsiLoginSpec, ipAddr string, isMinimal bool, isLocate bool) error {
	outerMap := make(map[string]bool)

	for _, t := range targets {
		iqnTarget := t.iqn + ":" + t.target
		if t.remoteIp != ipAddr {
			continue
		}
		loginParams := []string{
			"--mode",
			"node",
			"--targetname",
			iqnTarget,
			"--portal",
			ipAddr,
			"--login",
		}
		DebugString(strings.Join(loginParams, " "))
		_, err := RunIscsiAdminTool(loginParams)
		if err != nil {
			if !isMinimal {
				fmt.Println("failed\t", iqnTarget)
			}
		} else {
			outerMap[iqnTarget] = true
		}
	}

	if len(outerMap) == 0 {
		return fmt.Errorf("No matching iscsi shares were found")
	}

	innerMap := make(map[string]bool)
	for key, value := range outerMap {
		innerMap[key] = value
	}

	shareCh := make(chan typeIscsiPathAndIqnTarget)
	go func() {
		err := core.WaitForFilesToAppear("/dev/disk/by-path", func(fname string, wasCreate bool)bool {
			IterateActivatedIscsiShares(ipAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
				shareCh <- typeIscsiPathAndIqnTarget {
					fullPath: path.Join(root, fullName),
					iqnTargetName: iqnTargetName,
				}
				delete(innerMap, iqnTargetName)
			})
			return len(innerMap) == 0
		})
		if err != nil && !isMinimal {
			fmt.Println("error\t", err)
		}
		close(shareCh)
	}()

	const maxTries = 30
	for i := 0; i < maxTries; i++ {
		select {
			case names := <- shareCh:
				if _, exists := outerMap[names.iqnTargetName]; exists {
					if isLocate {
						fmt.Println("activated\t" + names.fullPath)
					} else {
						fmt.Println(names.fullPath)
					}
					delete(outerMap, names.iqnTargetName)
				}
			case <- time.After(time.Duration(1000) * time.Millisecond):
				IterateActivatedIscsiShares(ipAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
					if _, exists := outerMap[iqnTargetName]; exists {
						fullPath := path.Join(root, fullName)
						if isLocate {
							fmt.Println("activated\t" + fullPath)
						} else {
							fmt.Println(fullPath)
						}
						delete(outerMap, iqnTargetName)
					}
				})
		}
		if len(outerMap) == 0 {
			break
		}
	}

	if !isMinimal {
		for iqnTargetName, _ := range outerMap {
			fmt.Println("timed-out\t" + iqnTargetName)
		}
	}

	return nil
}

func deactivateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, false, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	maybeHashedToVolumeMap := make(map[string]string)
	for _, vol := range args {
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
	}

	cmd.SilenceUsage = true

	if err := CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	hostname := core.StripPort(api.GetHostName())
	ipAddrs, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	ipPortalAddr := ipAddrs[0].String() + ":" + options.allFlags["iscsi_port"]
	isMinimal := core.IsStringTrue(options.allFlags, "parsable")

	DeactivateMatchingIscsiTargets(ipPortalAddr, maybeHashedToVolumeMap, isMinimal, true)

	if !isMinimal {
		for _, vol := range maybeHashedToVolumeMap {
			fmt.Println("Not found: " + vol)
		}
	}

	return nil
}

// This command is needed to delete the iscsi extent/target without deleting the underlying dataset.
// However, deleting a dataset will delete the extent and target as well.
// It will deactivate the share before deleting it.
func deleteIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, false, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	diskNames := make([]string, 0)
	diskNameIndex := make(map[string]int)
	argsMapIndex := make(map[string]int)
	maybeHashedToVolumeMap := make(map[string]string)

	for i, vol := range args {
		diskNames = append(diskNames, "zvol/" + vol)
		diskNameIndex["zvol/" + vol] = i
		argsMapIndex[vol] = i
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
	}

	cmd.SilenceUsage = true

	changes := make([]typeApiCallRecord, 0)
	defer undoIscsiDeleteList(api, &changes)

	if err := CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	hostname := core.StripPort(api.GetHostName())
	ipAddrs, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	ipPortalAddr := ipAddrs[0].String() + ":" + options.allFlags["iscsi_port"]

	DeactivateMatchingIscsiTargets(ipPortalAddr, maybeHashedToVolumeMap, true, false)

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  true,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}

	responseTarget, err := QueryApi(api, "iscsi.target", args, core.StringRepeated("alias", len(args)), nil, extras)
	if err != nil {
		return err
	}

	targetIds := GetIdsOrderedByArgsFromResponse(responseTarget, "alias", args, argsMapIndex)

	targetIdsDelete := make([]interface{}, len(targetIds))
	for i, t := range targetIds {
		targetIdsDelete[i] = []interface{} {t, true, true} // id, force, delete_extents
	}

	if len(targetIdsDelete) > 0 {
		timeout := 15 * len(targetIdsDelete)
		_, _, err = MaybeBulkApiCallArray(api, "iscsi.target.delete", int64(timeout), targetIdsDelete, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func undoIscsiDeleteList(api core.Session, changes *[]typeApiCallRecord) {
	DebugString("undoIscsiDeleteList")
	for _, call := range *changes {
		DebugString(call.endpoint)
		if strings.HasSuffix(call.endpoint, ".delete") {
			idList := make([]interface{}, 0)
			for _, r := range call.resultList {
				idList = append(idList, core.GetIdFromObject(r))
			}
			if len(idList) > 0 {
				MaybeBulkApiCallArray(
					api,
					call.endpoint[:len(call.endpoint)-7] + ".create",
					10,
					idList,
					false,
				)
			}
		}
	}
}

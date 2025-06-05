package cmd

import (
	"encoding/json"
	"fmt"
	"os/user"
	"path"
	"strings"
	"time"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

const DEFAULT_ISCSI_PORT = 3260

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

var iscsiTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test an iSCSI portal connection, and optionally setup one as well",
}

var iscsiSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Ensure the iSCSI service is started and there exists a portal and initiator for the given host",
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
	iscsiTestCmd.RunE = WrapCommandFunc(testIscsi)
	iscsiSetupCmd.RunE = WrapCommandFunc(setupIscsi)
	iscsiListCmd.RunE = WrapCommandFunc(listIscsi)
	iscsiLocateCmd.RunE = WrapCommandFunc(locateIscsi)
	iscsiDeactivateCmd.RunE = WrapCommandFunc(deactivateIscsi)
	iscsiDeleteCmd.RunE = WrapCommandFunc(deleteIscsi)

	iscsiCreateCmd.Flags().Bool("readonly", false, "Ensure the new iSCSI extent is read-only. Ignored for snapshots.")

	iscsiTestCmd.Flags().Bool("setup", false, "Ensure the iSCSI service is started and there exists a portal and initiator for the given host")

	iscsiSetupCmd.Flags().Bool("test", false, "Test the iSCSI portal connection")

	iscsiLocateCmd.Flags().Bool("activate", false, "Activate any shares that could not be located")
	iscsiLocateCmd.Flags().Bool("create", false, "Create any shares that could not be activated or located, then activate them")
	iscsiLocateCmd.Flags().Bool("deactivate", false, "Deactivate any shares that could be located")
	iscsiLocateCmd.Flags().Bool("delete", false, "Deactivate and delete any shares that could be located")
	iscsiLocateCmd.Flags().Bool("readonly", false, "If a share is to be created, ensure that its extent is read-only. Ignored for snapshots.")

	_iscsiCmds := []*cobra.Command {iscsiCreateCmd, iscsiTestCmd, iscsiSetupCmd, iscsiActivateCmd, iscsiLocateCmd, iscsiDeactivateCmd, iscsiDeleteCmd}
	for _, c := range _iscsiCmds {
		c.Flags().StringP("target-prefix", "t", "", "label to prefix the created target")
		c.Flags().Bool("parsable", false, "Parsable (ie. minimal) output")
		c.Flags().StringP("portal", "p", ":", "iSCSI portal [ip]:[port] or id")
		c.Flags().StringP("initiator", "i", "", "iSCSI initiator id or comment")
	}

	iscsiCmd.AddCommand(iscsiCreateCmd)
	iscsiCmd.AddCommand(iscsiActivateCmd)
	iscsiCmd.AddCommand(iscsiTestCmd)
	iscsiCmd.AddCommand(iscsiSetupCmd)
	iscsiCmd.AddCommand(iscsiListCmd)
	iscsiCmd.AddCommand(iscsiLocateCmd)
	iscsiCmd.AddCommand(iscsiDeactivateCmd)
	iscsiCmd.AddCommand(iscsiDeleteCmd)

	AddIscsiCrudCommands(iscsiCmd)

	shareCmd.AddCommand(iscsiCmd)
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

	portalId, err := LookupPortalIdOrCreate(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	initiatorId, err := LookupInitiatorOrCreateBlank(api, options.allFlags["initiator"])
	if err != nil {
		return err
	}

	toUpdateMap := make(map[string]interface{})
	for _, target := range responseTargetQuery.resultsMap {
		if targetAlias, _ := target["alias"].(string); targetAlias != "" {
			toUpdateMap[targetAlias] = core.GetIdFromObject(target)
		}
	}

	targetUpdates := make([]interface{}, 0)
	targetCreates := make([]interface{}, 0)

	for _, vol := range args {
		group := make(map[string]interface{})
		group["portal"] = portalId
		group["initiator"] = initiatorId

		obj := make(map[string]interface{})
		obj["name"] = volumeToMaybeHashedMap[vol]
		obj["alias"] = vol
		obj["groups"] = []map[string]interface{}{group}

		if idObj, exists := toUpdateMap[vol]; exists {
			targetUpdates = append(targetUpdates, []interface{}{idObj, obj})
		} else {
			targetCreates = append(targetCreates, []interface{}{obj})
		}
	}

	if len(targetUpdates) == 0 && len(targetCreates) == 0 {
		if !core.IsStringTrue(options.allFlags, "parsable") {
			fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		}
		return nil
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

	if strings.HasPrefix(cmd.Use, "locate") || !core.IsStringTrue(options.allFlags, "parsable") {
		for _, target := range allTargets {
			vol, _ := target["alias"].(string)
			fmt.Println("created\t" + vol)
		}
	} else {
		for _, target := range allTargets {
			vol, _ := target["alias"].(string)
			fmt.Println(vol)
		}
	}

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

func testIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true
	options, _ := GetCobraFlags(cmd, false, nil)

	checkedServiceState := false
	if core.IsStringTrue(options.allFlags, "setup") {
		if err := setupIscsiImpl(api, options); err != nil {
			return err
		}
		checkedServiceState = true
	}
	return testIscsiImpl(api, options, checkedServiceState)
}

func setupIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true
	options, _ := GetCobraFlags(cmd, false, nil)

	if err := setupIscsiImpl(api, options); err != nil {
		return err
	}
	if core.IsStringTrue(options.allFlags, "test") {
		return testIscsiImpl(api, options, true)
	}
	return nil
}

func testIscsiImpl(api core.Session, options FlagMap, checkedServiceState bool) error {
	if !checkedServiceState {
		msg, err := CheckRemoteIscsiServiceIsRunning(api)
		if err != nil {
			return err
		}
		if msg != "" {
			return fmt.Errorf(msg)
		}
	}

	ipPortalAddr, err := MaybeLookupIpPortFromPortal(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	discoveryOutput, err := TestIscsiDiscovery(api, ipPortalAddr)
	if err != nil {
		return err
	}

	if !core.IsStringTrue(options.allFlags, "parsable") {
		fmt.Println(discoveryOutput)
	}
	return nil
}

func setupIscsiImpl(api core.Session, options FlagMap) error {
	isMinimal := core.IsStringTrue(options.allFlags, "parsable")
	msg, err := CheckRemoteIscsiServiceIsRunning(api)
	if err != nil {
		return err
	}
	if msg != "" {
		isEnable := true
		if err := changeServiceStateImpl(api, "start", nil, isEnable, false, []string {"iscsitarget"}); err != nil {
			return err
		}
		if !isMinimal {
			if isEnable {
				fmt.Println("Started and enabled iscsitarget service")
			} else {
				fmt.Println("Started iscsitarget service")
			}
		}
	}

	portalId, err := LookupPortalIdOrCreate(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}
	if !isMinimal {
		fmt.Println("Portal ID:", portalId)
	}

	initiatorId, err := LookupInitiatorOrCreateBlank(api, options.allFlags["initiator"])
	if err != nil {
		return err
	}
	if !isMinimal {
		fmt.Println("Initiator ID:", initiatorId)
	}

	return nil
}

func listIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	IterateActivatedIscsiShares("", func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
		fullPath := path.Join(root, fullName)
		fmt.Println(fullPath)
	})
	return nil
}

func getIscsiSharesFromSessionAndDiscovery(options FlagMap, api core.Session, args []string, portalAddr string) (map[string]bool, map[string]string, error) {
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	maybeHashedToVolumeMap := make(map[string]string)
	missingShares := make(map[string]string)

	for _, vol := range args {
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
		missingShares[maybeHashed] = vol
	}

	var err error
	if err = CheckIscsiAdminToolExists(); err != nil {
		return nil, nil, err
	}

	err = MaybeLaunchIscsiDaemon()
	if err != nil {
		return nil, nil, err
	}

	isCreate := core.IsStringTrue(options.allFlags, "create")

	sessionTargets, err := GetIscsiTargetsFromSession(api, maybeHashedToVolumeMap)
	if !isCreate && err != nil && !strings.Contains(strings.ToLower(err.Error()), "no active sessions") {
		return nil, nil, err
	}

	discoveryTargets, err := GetIscsiTargetsFromDiscovery(api, maybeHashedToVolumeMap, portalAddr)
	if !isCreate && err != nil {
		return nil, nil, err
	}

	targets := make([]typeIscsiLoginSpec, 0)
	targets = append(targets, sessionTargets...)
	targets = append(targets, discoveryTargets...)

	if len(targets) == 0 {
		return nil, missingShares, nil
	}

	shares := make(map[string]bool)
	for _, t := range targets {
		shares[t.iqn + ":" + t.target] = true
		delete(missingShares, t.target)
	}

	return shares, missingShares, nil
}

func locateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	thisUser, err := user.Current()
	if err == nil {
		if thisUser.Username != "root" {
			return fmt.Errorf("This command must be run as root.")
		}
	}

	options, _ := GetCobraFlags(cmd, false, nil)

	ipPortalAddr, err := MaybeLookupIpPortFromPortal(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	shares, missingShares, err := getIscsiSharesFromSessionAndDiscovery(options, api, args, ipPortalAddr)
	if err != nil {
		return err
	}

	isMinimal := core.IsStringTrue(options.allFlags, "parsable")
	shouldActivate := core.IsStringTrue(options.allFlags, "activate")
	shouldDeactivate := core.IsStringTrue(options.allFlags, "deactivate")
	shouldCreate := core.IsStringTrue(options.allFlags, "create")
	shouldDelete := core.IsStringTrue(options.allFlags, "delete")

	shouldActivate = shouldActivate || shouldCreate
	shouldDeactivate = shouldDeactivate || shouldDelete

	if shares == nil && !shouldCreate && !isMinimal {
		return fmt.Errorf("Could not find any matching iscsi shares")
	}

	if shouldCreate {
		toCreate := make([]string, 0)
		for _, vol := range missingShares {
			toCreate = append(toCreate, vol)
		}
		if len(toCreate) > 0 {
			if err = createIscsi(cmd, api, toCreate); err != nil {
				return err
			}
		}
	}

	toDeactivate := make([]string, 0)
	toDeactivateTargets := make([]string, 0)

	IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipAddr string, iqnTargetName string, targetOnlyName string) {
		if _, exists := shares[iqnTargetName]; !exists {
			return
		}
		if shouldDeactivate {
			toDeactivate = append(toDeactivate, iqnTargetName)
			toDeactivateTargets = append(toDeactivateTargets, targetOnlyName)
		} else {
			fullPath := path.Join(root, fullName)
			fmt.Println("located\t" + fullPath)
		}
		delete(shares, iqnTargetName)
	})

	if shouldActivate {
		var remainingTargets []typeIscsiLoginSpec
		if shouldCreate {
			remainingTargets, _ = GetIscsiTargetsFromDiscovery(api, missingShares, ipPortalAddr)
		} else {
			remainingTargets = make([]typeIscsiLoginSpec, 0)
		}
		for share, _ := range shares {
			parts := strings.Split(share, ":")
			iqn := parts[0]
			var target string
			if len(parts) > 1 {
				target = strings.Join(parts[1:], ":")
			}
			t := typeIscsiLoginSpec {
				remoteIp: ipPortalAddr,
				iqn: iqn,
				target: target,
			}
			remainingTargets = append(remainingTargets, t)
		}
		if len(remainingTargets) > 0 {
			if err = doIscsiActivate(api, remainingTargets, ipPortalAddr, isMinimal, true); err != nil {
				return err
			}
		}
	}

	// deleteIscsi() will deactivate the shares first regardless, so use else if here
	if shouldDelete && len(toDeactivateTargets) > 0 {
		if err = deleteIscsi(cmd, api, toDeactivateTargets); err != nil {
			return err
		}
		for _, t := range toDeactivate {
			fmt.Println("deactivated\t" + t)
		}
	} else if shouldDeactivate {
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
			_, err := RunIscsiAdminTool(api, logoutParams)
			if err != nil {
				fmt.Printf("failed\t%s\t%v\n", t, err)
			} else {
				fmt.Println("deactivated\t" + t)
			}
		}
	}

	return nil
}

type typeIscsiPathAndIqnTarget struct {
	fullPath string
	iqnTargetName string
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	thisUser, err := user.Current()
	if err == nil {
		if thisUser.Username != "root" {
			return fmt.Errorf("This command must be run as root.")
		}
	}

	options, _ := GetCobraFlags(cmd, false, nil)

	ipPortalAddr, err := MaybeLookupIpPortFromPortal(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	shares, _, err := getIscsiSharesFromSessionAndDiscovery(options, api, args, ipPortalAddr)
	if err != nil {
		return err
	}
	if shares == nil {
		return nil
	}

	isMinimal := core.IsStringTrue(options.allFlags, "parsable")

	targets := make([]typeIscsiLoginSpec, 0)
	for share, _ := range shares {
		parts := strings.Split(share, ":")
		iqn := parts[0]
		var target string
		if len(parts) > 1 {
			target = strings.Join(parts[1:], ":")
		}
		targets = append(targets, typeIscsiLoginSpec{
			remoteIp: ipPortalAddr,
			iqn: iqn,
			target: target,
		})
	}

	return doIscsiActivate(api, targets, ipPortalAddr, isMinimal, false)
}

func doIscsiActivate(api core.Session, targets []typeIscsiLoginSpec, ipAddr string, isMinimal bool, shouldPrintStatus bool) error {
	outerMap := make(map[string]bool)

	for _, t := range targets {
		iqnTarget := t.iqn + ":" + t.target
		if t.remoteIp != ipAddr {
			if !isMinimal {
				fmt.Println("IP MISMATCH:", t.remoteIp, "!=", ipAddr)
			}
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
		_, err := RunIscsiAdminTool(api, loginParams)
		if err == nil {
			outerMap[iqnTarget] = true
		} else {
			return fmt.Errorf("failed\t%s\t%v", iqnTarget, err)
		}
	}

	if len(outerMap) == 0 {
		if !isMinimal {
			return fmt.Errorf("No matching iscsi shares were found")
		}
		return nil
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
					if !isMinimal || shouldPrintStatus {
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
						if !isMinimal || shouldPrintStatus {
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
	cmd.SilenceUsage = true

	thisUser, err := user.Current()
	if err == nil {
		if thisUser.Username != "root" {
			return fmt.Errorf("This command must be run as root.")
		}
	}

	options, _ := GetCobraFlags(cmd, false, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	maybeHashedToVolumeMap := make(map[string]string)
	for _, vol := range args {
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
	}

	if err := CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	ipPortalAddr, err := MaybeLookupIpPortFromPortal(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	isMinimal := core.IsStringTrue(options.allFlags, "parsable")

	deactivatedList := DeactivateMatchingIscsiTargets(api, ipPortalAddr, maybeHashedToVolumeMap, isMinimal, false)

	if !isMinimal {
		for _, vol := range deactivatedList {
			delete(maybeHashedToVolumeMap, vol)
		}
		for _, vol := range maybeHashedToVolumeMap {
			fmt.Println("not-found\t" + vol)
		}
	}

	return nil
}

// This command is needed to delete the iscsi extent/target without deleting the underlying dataset.
// However, deleting a dataset will delete the extent and target as well.
// It will deactivate the share before deleting it.
func deleteIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	thisUser, err := user.Current()
	if err == nil {
		if thisUser.Username != "root" {
			return fmt.Errorf("This command must be run as root.")
		}
	}

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

	ipPortalAddr, err := MaybeLookupIpPortFromPortal(api, DEFAULT_ISCSI_PORT, options.allFlags["portal"])
	if err != nil {
		return err
	}

	_ = DeactivateMatchingIscsiTargets(api, ipPortalAddr, maybeHashedToVolumeMap, true, true)

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

	isMinimal := core.IsStringTrue(options.allFlags, "parsable")
	targetIds, targetNames := GetIdsOrderedByArgsFromResponse(responseTarget, "alias", args, argsMapIndex, isMinimal)

	if len(targetIds) == 0 {
		if !isMinimal {
			fmt.Println("Could not find any shares to delete")
		}
		return nil
	}

	targetIdsDelete := make([]interface{}, len(targetIds))
	for i, t := range targetIds {
		targetIdsDelete[i] = []interface{} {t, true, true} // id, force, delete_extents
	}

	timeout := int64(10 + 10 * len(targetIdsDelete))

	extras = typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  false,
		shouldGetUserProps: false,
		shouldRecurse:      true,
	}
	responseDatasets, err := QueryApi(api, "pool.dataset", args, core.StringRepeated("name", len(args)), []string{}, extras)
	if err == nil {
		timeout = int64(10 + 10 * len(responseDatasets.resultsMap))
	}

	_, _, err = MaybeBulkApiCallArray(api, "iscsi.target.delete", int64(timeout), targetIdsDelete, true)
	if err != nil {
		return err
	}

	for _, name := range targetNames {
		fmt.Println("deleted\t" + name)
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

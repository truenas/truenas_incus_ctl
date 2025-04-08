package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
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
	Use:   "create",
	Short: "Create description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiActivateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiListCmd = &cobra.Command{
	Use:   "list",
	Short: "List description",
}

var iscsiLocateCmd = &cobra.Command{
	Use:   "locate",
	Short: "Locate description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete description",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	iscsiCreateCmd.RunE = WrapCommandFunc(createIscsi)
	iscsiActivateCmd.RunE = WrapCommandFunc(activateIscsi)
	iscsiListCmd.RunE = WrapCommandFunc(listIscsi)
	iscsiLocateCmd.RunE = WrapCommandFunc(locateIscsi)
	iscsiDeactivateCmd.RunE = WrapCommandFunc(deactivateIscsi)
	iscsiDeleteCmd.RunE = WrapCommandFunc(deleteIscsi)

	_iscsiCmds := []*cobra.Command {iscsiCreateCmd, iscsiActivateCmd, iscsiListCmd, iscsiLocateCmd, iscsiDeactivateCmd, iscsiDeleteCmd}
	for _, c := range _iscsiCmds {
		c.Flags().StringP("target-prefix", "t", "incus", "label to prefix the created target")
		c.Flags().IntP("port", "p", 3260, "iSCSI portal port")
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
	options, _ := GetCobraFlags(cmd, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)
	cmd.SilenceUsage = true

	changes := make([]typeApiCallRecord, 0)
	defer undoIscsiCreateList(api, &changes)

	toEnsure := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iscsiName := MakeIscsiTargetNameFromVolumePath(prefixName, vol)
		if _, exists := iscsiToVolumeMap[iscsiName]; exists {
			return fmt.Errorf("There are duplicates in the provided list of datasets")
		}
		iscsiToVolumeMap[iscsiName] = vol
		toEnsure = append(toEnsure, iscsiName)
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
	timestampTargets := time.Now().UnixMilli()
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
		obj["name"] = MakeIscsiTargetUuid(prefixName, volName, timestampTargets)
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
	timestampExtents := time.Now().UnixMilli()
	for _, vol := range args {
		if _, exists := extentsByDisk["zvol/" + vol]; !exists {
			extentsCreate = append(extentsCreate, "zvol/" + vol)
			iName := MakeIscsiTargetNameFromVolumePath(prefixName, vol)
			extentsIqnCreate = append(extentsIqnCreate, MakeIscsiTargetUuid(prefixName, iName, timestampExtents))
		}
	}

	if len(extentsCreate) > 0 {
		paramsCreate := make([]interface{}, len(extentsCreate))
		for i, _ := range extentsCreate {
			paramsCreate[i] = []interface{} {
				map[string]interface{} {
					"name": extentsIqnCreate[i],
					"disk": extentsCreate[i],
					"path": extentsCreate[i],
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

	_, _, err = MaybeBulkApiCallArray(api, "iscsi.targetextent.create", 10, teCreateList, true)
	if err != nil {
		return err
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
				idList = append(idList, core.GetIdFromObject(r))
			}
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

func listIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	diskEntries, err := os.ReadDir("/dev/disk/by-path")
	if err != nil {
		return err
	}
	for _, e := range diskEntries {
		name := e.Name()
		/*
			if !strings.Contains(name, ipPortalAddr) {
				continue
			}
		*/
		iqnFindPos := strings.Index(name, "-iscsi-iqn.")
		if iqnFindPos == -1 {
			continue
		}
		iqnStart := iqnFindPos + 7
		iqnSepPos := strings.Index(name[iqnStart:], ":")
		if iqnSepPos == -1 {
			continue
		}
		fmt.Println("/dev/disk/by-path/" + name)
	}

	return nil
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	return activateOrLocateIscsi(cmd, api, args, true)
}

func locateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	return activateOrLocateIscsi(cmd, api, args, false)
}

func activateOrLocateIscsi(cmd *cobra.Command, api core.Session, args []string, isActivate bool) error {
	options, _ := GetCobraFlags(cmd, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	iscsiNames := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iName := MakeIscsiTargetNameFromVolumePath(prefixName, vol)
		iscsiToVolumeMap[iName] = vol
		iscsiNames = append(iscsiNames, iName)
	}

	cmd.SilenceUsage = true

	var err error
	if err = CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	err = MaybeLaunchIscsiDaemon()
	if err != nil {
		return err
	}

	var targets []typeIscsiLoginSpec

	if !isActivate {
		targets, _ = GetIscsiTargetsFromSession(iscsiToVolumeMap)
	}

	if len(targets) == 0 {
		hostUrl, err := url.Parse(api.GetHostUrl())
		if err != nil {
			return err
		}

		portalAddr := hostUrl.Hostname() + ":" + options.allFlags["port"]
		targets, err = GetIscsiTargetsFromDiscovery(iscsiToVolumeMap, portalAddr)
		if err != nil {
			return err
		}
	}

	if isActivate && len(targets) > 0 {
		for _, t := range targets {
			loginParams := []string{
				"--mode",
				"node",
				"--targetname",
				t.iqn + ":" + t.target,
				"--portal",
				t.remoteIp,
				"--login",
			}
			DebugString(strings.Join(loginParams, " "))
			_, err := RunIscsiAdminTool(loginParams)
			if err != nil {
				t.status = "FAILED"
			} else {
				t.status = "activated"
			}
		}

		/*
			RunIscsiAdminTool([]string{
				"--mode",
				"session",
				"-r",
				"1",
				"-P3",
			})
		*/

		time.Sleep(time.Duration(4) * time.Second)
	}

	fmt.Println(strings.Join(LocateIqnTargetsLocally(targets), "\n"))
	return nil
}

func deactivateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	iscsiNames := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iName := MakeIscsiTargetNameFromVolumePath(prefixName, vol)
		iscsiToVolumeMap[iName] = vol
		iscsiNames = append(iscsiNames, iName)
	}

	cmd.SilenceUsage = true

	if err := CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	hostUrl, err := url.Parse(api.GetHostUrl())
	if err != nil {
		return err
	}

	ipAddrs, err := net.LookupIP(hostUrl.Hostname())
	if err != nil {
		return err
	}

	ipPortalAddr := ipAddrs[0].String() + ":" + options.allFlags["port"]

	DeactivateMatchingIscsiTargets(ipPortalAddr, iscsiNames, iscsiToVolumeMap, true)

	for _, iName := range iscsiToVolumeMap {
		fmt.Println("Not found: " + iName)
	}

	return nil
}

// This command is needed to delete the iscsi extent/target without deleting the underlying dataset.
// However, deleting a dataset will delete the extent and dataset as well.
func deleteIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, nil)
	prefixName := GetIscsiTargetPrefixOrExit(options.allFlags)

	iscsiNames := make([]string, 0)
	diskNames := make([]string, 0)
	diskNameIndex := make(map[string]int)
	argsMapIndex := make(map[string]int)
	iscsiToVolumeMap := make(map[string]string)

	for i, vol := range args {
		iName := MakeIscsiTargetNameFromVolumePath(prefixName, vol)
		iscsiToVolumeMap[iName] = vol
		iscsiNames = append(iscsiNames, iName)
		diskNames = append(diskNames, "zvol/" + vol)
		diskNameIndex["zvol/" + vol] = i
		argsMapIndex[vol] = i
	}

	cmd.SilenceUsage = true

	changes := make([]typeApiCallRecord, 0)
	defer undoIscsiDeleteList(api, &changes)

	if err := CheckIscsiAdminToolExists(); err != nil {
		return err
	}

	hostUrl, err := url.Parse(api.GetHostUrl())
	if err != nil {
		return err
	}

	ipAddrs, err := net.LookupIP(hostUrl.Hostname())
	if err != nil {
		return err
	}

	ipPortalAddr := ipAddrs[0].String() + ":" + options.allFlags["port"]

	DeactivateMatchingIscsiTargets(ipPortalAddr, iscsiNames, nil, false)

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

	responseExtent, err := QueryApi(api, "iscsi.extent", diskNames, core.StringRepeated("disk", len(diskNames)), nil, extras)
	if err != nil {
		return nil
	}

	targetIds := GetIdsOrderedByArgsFromResponse(responseTarget, "alias", args, argsMapIndex)
	extentIds := GetIdsOrderedByArgsFromResponse(responseExtent, "disk", diskNames, diskNameIndex)

	DebugString("targets " + fmt.Sprint(targetIds))
	DebugString("extents " + fmt.Sprint(extentIds))

	var teInnerFilter []interface{}
	if len(targetIds) > 0 && len(extentIds) > 0 {
		teInnerFilter = []interface{} {
			[]interface{} {
				"OR",
				[]interface{} {
					[]interface{} {
						"target",
						"in",
						targetIds,
					},
					[]interface{} {
						"extent",
						"in",
						extentIds,
					},
				},
			},
		}
	} else if len(targetIds) > 0 {
		teInnerFilter = []interface{} {
			[]interface{} {
				"target",
				"in",
				targetIds,
			},
		}
	} else if len(extentIds) > 0 {
		teInnerFilter = []interface{} {
			[]interface{} {
				"extent",
				"in",
				extentIds,
			},
		}
	} else {
		fmt.Println("No matching extents or targets were found")
		return nil
	}

	teParams := []interface{} {teInnerFilter, make(map[string]interface{})}
	DebugJson(teParams)
	out, err := core.ApiCall(
		api,
		"iscsi.targetextent.query",
		10,
		teParams,
	)
	if err != nil {
		return err
	}
	DebugString(string(out))

	teResults, teErrors := core.GetResultsAndErrorsFromApiResponseRaw(out)

	teIds := make([]interface{}, len(teResults))
	for i, result := range teResults {
		teIds[i] = []interface{} {core.GetIdFromObject(result)}
	}

	_, _, err = MaybeBulkApiCallArray(api, "iscsi.targetextent.delete", 10, teIds, true)
	if err != nil {
		return err
	}
	changes = append(changes, typeApiCallRecord {
		endpoint: "iscsi.targetextent.delete",
		params: teIds,
		resultList: teResults,
		errorList: teErrors,
	})

	targets := make([]interface{}, 0)
	for _, v := range responseTarget.resultsMap {
		targets = append(targets, v)
	}
	targetIdsDelete := make([]interface{}, len(targetIds))
	for i, t := range targetIds {
		targetIdsDelete[i] = []interface{} {t}
	}

	_, _, err = MaybeBulkApiCallArray(api, "iscsi.target.delete", 10, targetIdsDelete, true)
	if err != nil {
		return err
	}
	changes = append(changes, typeApiCallRecord {
		endpoint: "iscsi.target.delete",
		params: targetIdsDelete,
		resultList: targets,
		errorList: nil,
	})

	extentIdsDelete := make([]interface{}, len(extentIds))
	for i, e := range extentIds {
		extentIdsDelete[i] = []interface{} {e}
	}
	_, _, err = MaybeBulkApiCallArray(api, "iscsi.extent.delete", 10, extentIdsDelete, true)
	if err != nil {
		return err
	}

	changes = make([]typeApiCallRecord, 0)
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

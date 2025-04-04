package cmd

import (
	"os"
	"fmt"
	"net"
	"net/url"
	"time"
	"strconv"
	"strings"
	"encoding/json"
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
	Use:     "activate",
	Short:   "Activate description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeactivateCmd = &cobra.Command{
	Use:     "deactivate",
	Short:   "Activate description",
	Args:  cobra.MinimumNArgs(1),
}

var iscsiDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete description",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

func init() {
	iscsiCreateCmd.RunE = WrapCommandFunc(createIscsi)
	iscsiActivateCmd.RunE = WrapCommandFunc(activateIscsi)
	iscsiDeactivateCmd.RunE = WrapCommandFunc(deactivateIscsi)
	iscsiDeleteCmd.RunE = WrapCommandFunc(deleteIscsi)

	iscsiActivateCmd.Flags().IntP("port", "p", 3260, "iSCSI portal port")

	iscsiDeactivateCmd.Flags().IntP("port", "p", 3260, "iSCSI portal port")

	iscsiCmd.AddCommand(iscsiCreateCmd)
	iscsiCmd.AddCommand(iscsiActivateCmd)
	iscsiCmd.AddCommand(iscsiDeactivateCmd)
	iscsiCmd.AddCommand(iscsiDeleteCmd)

	shareCmd.AddCommand(iscsiCmd)
}

type typeIscsiTargetParams struct {
	verb string
	id interface{}
	portalId int
	initiatorId int
}

func createIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	cmd.SilenceUsage = true

	toEnsure := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iscsiName := MakeIscsiTargetNameFromVolumePath(vol)
		iscsiToVolumeMap[iscsiName] = vol
		toEnsure = append(toEnsure, iscsiName)
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(true),
		shouldGetAllProps:  true,
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}
	response, err := QueryApi(api, "iscsi.target", toEnsure, core.StringRepeated("name", len(toEnsure)), nil, extras)
	if err != nil {
		return err
	}

	toCreateMap := make(map[string]bool)
	for _, t := range toEnsure {
		toCreateMap[t] = true
	}

	//missingInitiators := make(map[string]bool)
	targets := make(map[string]typeIscsiTargetParams)
	shouldFindPortal := false

	for targetId, target := range response.resultsMap {
		targetName, _ := target["name"].(string)
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

				portal := -1
				initiator := 1 // -1
				if portalExists {
					portal = 0
				} else {
					shouldFindPortal = true
				}
				if initiatorExists {
					initiator = 1 // 0
				} else {
					//missingInitiators[targetName] = true
				}

				targets[targetName] = typeIscsiTargetParams{
					verb: "update",
					id: targetId,
					portalId: portal,
					initiatorId: initiator,
				}
			}
		}
		if !anyGroups {
			shouldFindPortal = true
			//missingInitiators[targetName] = true

			targets[targetName] = typeIscsiTargetParams{
				verb: "update",
				id: targetId,
				portalId: -1,
				initiatorId: 1, // -1
			}
		}
	}

	for targetName, _ := range toCreateMap {
		shouldFindPortal = true
		//missingInitiators[targetName] = true

		targets[targetName] = typeIscsiTargetParams{
			verb: "create",
			id: -1,
			portalId: -1,
			initiatorId: 1, // -1
		}
	}

	if len(targets) == 0 {
		fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		return nil
	}

	emptyQueryParams := []interface{} {make([]interface{}, 0), make(map[string]interface{})}

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
	for name, t := range targets {
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
		obj["name"] = name
		obj["alias"] = iscsiToVolumeMap[name]

		if !isGroupEmpty {
			obj["groups"] = []map[string]interface{} {group}
		}

		if t.verb == "create" {
			targetCreates = append(targetCreates, []interface{} {obj})
		} else {
			if id, errNotNumber := strconv.Atoi(fmt.Sprint(t.id)); errNotNumber == nil {
				t.id = id
			}
			targetUpdates = append(targetUpdates, []interface{} {t.id, obj})
		}
	}

	jobIdUpdate := int64(-1)
	jobIdCreate := int64(-1)
	var rawResultsTargetUpdate json.RawMessage
	var rawResultsTargetCreate json.RawMessage

	if len(targetUpdates) > 0 {
		rawResultsTargetUpdate, jobIdUpdate, err = MaybeBulkApiCallArray(api, "iscsi.target.update", 10, targetUpdates, false)
		if err != nil {
			return err
		}
	}
	if len(targetCreates) > 0 {
		rawResultsTargetCreate, jobIdCreate, err = MaybeBulkApiCallArray(api, "iscsi.target.create", 10, targetCreates, false)
		if err != nil {
			return err
		}
	}

	if jobIdUpdate >= 0 {
		rawResultsTargetUpdate, err = api.WaitForJob(jobIdUpdate)
		if err != nil {
			return err
		}
	}
	if jobIdCreate >= 0 {
		rawResultsTargetCreate, err = api.WaitForJob(jobIdCreate)
		if err != nil {
			return err
		}
	}

	resultsTargetUpdate, _ := core.GetResultsAndErrorsFromApiResponseRaw(rawResultsTargetUpdate)
	resultsTargetCreate, _ := core.GetResultsAndErrorsFromApiResponseRaw(rawResultsTargetCreate)

	extentList := make([]string, len(args))
	for i, vol := range extentList {
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
			extentsIqnCreate = append(extentsIqnCreate, MakeIscsiTargetNameFromVolumePath(vol))
		}
	}

	if len(extentsCreate) > 0 {
		params := []interface{} {
			map[string]interface{} {
				"name": extentsIqnCreate[0],
				"disk": extentsCreate[0],
				"path": extentsCreate[0],
			},
		}
		objRemap := map[string][]interface{} {
			"name": core.ToAnyArray(extentsIqnCreate),
			"disk": core.ToAnyArray(extentsCreate),
			"path": core.ToAnyArray(extentsCreate),
		}
		out, _, err := MaybeBulkApiCall(
			api,
			"iscsi.extent.create",
			10,
			params,
			objRemap,
			true,
		)
		if err != nil {
			return err
		}

		resultsExtentCreate, _ := core.GetResultsAndErrorsFromApiResponseRaw(out)
		for _, extent := range resultsExtentCreate {
			if extentMap, ok := extent.(map[string]interface{}); ok {
				extentsByDisk[fmt.Sprint(extentMap["disk"])] = extentMap
			}
		}
	}

	responseTeQuery, err := QueryApi(api, "iscsi.targetextent", nil, nil, nil, extras)
	if err != nil {
		return err
	}
	teList := GetListFromQueryResponse(&responseTeQuery)
	teList = teList

	resultsTargetUpdate = resultsTargetUpdate
	resultsTargetCreate = resultsTargetCreate

	/*
	teCreateList := make([]string, 0)
	for _, te := range teList {
		targetId := te["target"]
		for _, t := range resultsTargetUpdate {
			
		}
		for _, t := range resultsTargetCreate {
			
		}

		extentId := te["extent"]
		for _, e := range extentsByDisk {
			
		}
	}
	*/

	return nil
}

type typeIscsiLoginSpec struct {
	remoteIp string
	iqn string
	target string
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, nil)

	iscsiNames := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iName := MakeIscsiTargetNameFromVolumePath(vol)
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

	portalAddr := hostUrl.Hostname() + ":" + options.allFlags["port"]

	params := []string{"--mode", "discoverydb", "--type", "sendtargets", "--portal", portalAddr, "--discover"}

	err = MaybeLaunchIscsiDaemon()
	if err != nil {
		return err
	}

	DebugString("activateIscsi: " + strings.Join(params, " "))
	out, err := RunIscsiAdminTool(params)
	if err != nil {
		return err
	}

	targets := make([]typeIscsiLoginSpec, 0)
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		spacePos := strings.Index(l, " ")
		if spacePos == -1 {
			continue
		}
		commaPos := strings.Index(l, ",")
		if commaPos == -1 || commaPos > spacePos {
			commaPos = spacePos
		}
		iqnSepPos := strings.Index(l[commaPos:], ":")
		if iqnSepPos == -1 {
			continue
		}

		targetName := l[commaPos+iqnSepPos+1:]
		if _, exists := iscsiToVolumeMap[targetName]; exists {
			t := typeIscsiLoginSpec{}
			t.remoteIp = l[0:commaPos]
			t.iqn = l[spacePos+1:commaPos+iqnSepPos]
			t.target = targetName
			targets = append(targets, t)
		}
	}

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
			return err
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

	diskEntries, err := os.ReadDir("/dev/disk/by-path")
	if err != nil {
		return err
	}
	for _, e := range diskEntries {
		name := e.Name()
		for _, t := range targets {
			//fmt.Println(name)
			if strings.Contains(name, t.iqn + ":" + t.target) {
				fmt.Println("/dev/disk/by-path/" + name)
				break
			}
		}
	}

	return nil
}

func deactivateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	options, _ := GetCobraFlags(cmd, nil)

	iscsiNames := make([]string, 0)
	iscsiToVolumeMap := make(map[string]string)
	for _, vol := range args {
		iName := MakeIscsiTargetNameFromVolumePath(vol)
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

	diskEntries, err := os.ReadDir("/dev/disk/by-path")
	if err != nil {
		return err
	}
	for _, e := range diskEntries {
		name := e.Name()
		if !strings.Contains(name, ipPortalAddr) {
			continue
		}
		iqnFindPos := strings.Index(name, "-iscsi-iqn.")
		if iqnFindPos == -1 {
			continue
		}
		iqnStart := iqnFindPos + 7
		iqnSepPos := strings.Index(name[iqnStart:], ":")
		if iqnSepPos == -1 {
			continue
		}
		iqn := name[iqnStart:iqnStart+iqnSepPos]

		for _, iName := range iscsiNames {
			if strings.Contains(name, iName) {
				logoutParams := []string{
					"--mode",
					"node",
					"--targetname",
					iqn + ":" + iName,
					"--portal",
					ipPortalAddr,
					"--logout",
				}
				DebugString(strings.Join(logoutParams, " "))
				_, err := RunIscsiAdminTool(logoutParams)
				if err != nil {
					return err
				}

				// remove this entry from the map, so that it will contain all iSCSI volumes that we tried to log out but failed to
				delete(iscsiToVolumeMap, iName)
				break
			}
		}
	}

	for _, iName := range iscsiToVolumeMap {
		fmt.Println("Error: " + iName + " was not found")
	}

	return nil
}

func deleteIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	fmt.Println("deleteIscsi")
	return nil
}

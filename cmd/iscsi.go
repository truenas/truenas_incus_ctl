package cmd

import (
	//"os/exec"
	"fmt"
	//"strconv"
	//"strings"
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

var iscsiDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete description",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"rm"},
}

func init() {
	iscsiCreateCmd.RunE = WrapCommandFunc(createIscsi)
	iscsiActivateCmd.RunE = WrapCommandFunc(activateIscsi)
	iscsiDeleteCmd.RunE = WrapCommandFunc(deleteIscsi)

	iscsiCmd.AddCommand(iscsiCreateCmd)
	iscsiCmd.AddCommand(iscsiActivateCmd)
	iscsiCmd.AddCommand(iscsiDeleteCmd)

	shareCmd.AddCommand(iscsiCmd)
}

type typeIscsiTargetParams struct {
	verb string
	id interface{}
	groupIndex int
	portalId int
	initiatorId int
}

func createIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	toEnsure := make([]string, 0)
	for _, vol := range args {
		toEnsure = append(toEnsure, "incus:" + MakeIscsiTargetNameFromVolumePath(vol))
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

	missingInitiators := make(map[string]bool)
	targets := make(map[string]typeIscsiTargetParams)
	shouldFindPortal := false

	for targetId, target := range response.resultsMap {
		targetName, _ := target["name"].(string)
		if targetName == "" {
			return fmt.Errorf("Name could not be found in iSCSI target with ID %d", targetId)
		}
		delete(toCreateMap, targetName)

		anyGroups := false
		if groupsObj, exists := target["groups"]; exists {
			if groups, ok := groupsObj.([]interface{}); ok && len(groups) > 0 {
				anyGroups = true
				for i := 0; i < len(groups); i++ {
					portalExists := false
					initiatorExists := false
					if elem, isElemMap := groups[i].(map[string]interface{}); isElemMap {
						_, portalExists = elem["portal"].(float64)
						_, initiatorExists = elem["initiator"].(float64)
					}

					portal := -1
					initiator := -1
					if portalExists {
						portal = 0
					} else {
						shouldFindPortal = true
					}
					if initiatorExists {
						initiator = 0
					} else {
						missingInitiators[targetName] = true
					}

					targets[targetName] = typeIscsiTargetParams{
						verb: "update",
						id: targetId,
						groupIndex: i,
						portalId: portal,
						initiatorId: initiator,
					}
				}
			}
		}
		if !anyGroups {
			shouldFindPortal = true
			missingInitiators[targetName] = true

			targets[targetName] = typeIscsiTargetParams{
				verb: "update",
				id: targetId,
				groupIndex: -1,
				portalId: -1,
				initiatorId: -1,
			}
		}
	}

	for targetName, _ := range toCreateMap {
		shouldFindPortal = true
		missingInitiators[targetName] = true

		targets[targetName] = typeIscsiTargetParams{
			verb: "create",
			id: -1,
			groupIndex: -1,
			portalId: -1,
			initiatorId: -1,
		}
	}

	if len(targets) == 0 {
		fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		return nil
	}

	defaultPortal := -1
	if shouldFindPortal {
		portalParams := []interface{} {make([]interface{}, 0), make(map[string]interface{})}
		out, err := core.ApiCall(api, "iscsi.portal.query", 10, portalParams)
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
			return fmt.Errorf("No iSCSI portal was found for this host. Use:\n" +
				"<truenas_incus_ctl> share iscsi portal create --ip <IP address> --port <port number>\n" +
				"To create one.\n")
		}
	}

	if len(missingInitiators) > 0 {
		queryList := make([]string, 0)
		for name, _ := range missingInitiators {
			queryList = append(queryList, name)
		}
		responseQuery, err := QueryApi(
			api,
			"iscsi.initiator",
			queryList,
			core.StringRepeated("comment", len(queryList)),
			nil,
			extras,
		)
		if err != nil {
			return err
		}

		initiatorIds := make(map[string]int)
		for _, r := range responseQuery.resultsMap {
			name, err := AddIscsiInitiator(initiatorIds, r)
			if err != nil {
				return err
			}
			delete(missingInitiators, name)
		}

		initiatorsToCreate := make([]string, 0)
		for name, _ := range missingInitiators {
			initiatorsToCreate = append(initiatorsToCreate, name)
		}

		if len(initiatorsToCreate) > 0 {
			paramsInitiator := map[string]interface{} {
				"initiators": make([]interface{}, 0),
				"comment": initiatorsToCreate[0],
			}
			objRemapInitiator := map[string][]interface{}{
				"comment": core.ToAnyArray(initiatorsToCreate),
			}
			out, err := MaybeBulkApiCall(
				api,
				"iscsi.initiator.create",
				10,
				[]interface{}{paramsInitiator},
				objRemapInitiator,
				true,
			)

			var responseCreate map[string]interface{}
			var results []interface{}
			if err := json.Unmarshal(out, &responseCreate); err != nil {
				return err
			}

			var errors []interface{}
			results, errors = core.GetResultsAndErrorsFromApiResponse(responseCreate)
			if len(errors) > 0 {
				return fmt.Errorf("iscsi.initiator.create errors:\n%v", errors)
			}
			for _, r := range results {
				if obj, ok := r.(map[string]interface{}); ok {
					_, err = AddIscsiInitiator(initiatorIds, obj)
					if err != nil {
						return err
					}
				}
			}
		}

		for name, _ := range targets {
			if targets[name].initiatorId == -1 {
				if id, exists := initiatorIds[name]; exists {
					// Go doesn't let you modfiy hashmap entries. Copy out, small change, copy in.
					t := targets[name]
					t.initiatorId = id
					targets[name] = t
				} else {
					return fmt.Errorf("Could not find target \"%s\" in initiator groups", name)
				}
			}
		}
	}

	for _, verb := range []string{"create","update"} {
		nWritten, out, err := IscsiCreateOrUpdateTargets(api, verb, targets)
		if err != nil {
			return err
		}
		if nWritten > 0 {
			fmt.Println(string(out))
		}
	}

	return nil
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	if err := CheckIscsiAdminToolExists(); err != nil {
		cmd.SilenceUsage = true
		return err
	}
	fmt.Println("activateIscsi")
	return nil
}

func deleteIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	fmt.Println("deleteIscsi")
	return nil
}

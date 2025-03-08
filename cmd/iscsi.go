package cmd

import (
	"os/exec"
	//"errors"
	"fmt"
	//"strconv"
	"strings"
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

	defaultPortal := 0
	targetUpdates := make(map[string]typeIscsiTargetParams)
	targetCreates := make(map[string]typeIscsiTargetParams)

	for targetId, target := range response.resultsMap {
		targetName, _ := target["name"].(string)
		if targetName == "" {
			return fmt.Errorf("Name could not be found in iSCSI target with ID %d", targetId)
		}
		delete(toCreateMap, targetName)

		defaultInitiator := 0
		anyGroups := false
		if groupsObj, exists := target["groups"]; exists {
			if groups, ok := groupsObj.([]interface{}); ok && len(groups) > 0 {
				anyGroups = true
				for i := 0; i < len(groups); i++ {
					elem, isElemMap := groups[i].(map[string]interface{})
					_, portalExists := elem["portal"]; 
					_, initiExists := elem["initiator"]; 
					if err = IscsiDetermineTargetUpdates(
						api,
						targetName,
						i,
						&initiatorCreates,
						targetUpdates,
						&defaultPortal,
						&defaultInitiator,
						!isElemMap || !portalExists,
						!isElemMap || !initiExists,
					); err != nil {
						return err
					}
				}
			}
		}
		if !anyGroups {
			if err = IscsiDetermineTargetUpdates(
				api,
				targetName,
				-1,
				&initiatorCreates,
				targetUpdates,
				&defaultPortal,
				&defaultInitiator,
				true,
				true,
			); err != nil {
				return err
			}
		}
	}

	for _, targetName := range toCreateMap {
		if err = IscsiDetermineTargetUpdates(
			api,
			targetName,
			-1,
			&initiatorCreates,
			targetCreates,
			&defaultPortal,
			&defaultInitiator,
			true,
			true,
		); err != nil {
			return err
		}
	}

	if defaultPortal == -1 {
		return fmt.Errorf("No iSCSI portal has been created for this host. Use:\n" +
			"%s share iscsi portal create --ip <IP address> --port <port number>\n" +
			"To create one.\n", os.Args[0])
	}

	if len(targetUpdates) == 0 && len(targetCreates) == 0 {
		fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		return nil
	}

	if len(initiatorCreates) > 0 {
		paramsInitiator := make(map[string]interface{})
		paramsInitiator["initiators"] = make([]interface{}, 0)
		paramsInitiator["comment"] = initiatorCreates[0]

		objRemapInitiator := map[string][]interface{}{"comment": core.ToAnyArray(initiatorCreates)}
		out, err := MaybeBulkApiCall(
			api,
			"iscsi.initiator.create",
			10,
			[]interface{}{paramsInitiator},
			objRemapInitiator,
			true
		)
		var response map[string]interface{}
		if err := json.Unmarshal(data, &response); err != nil {
			results := core.GetResultsListFromApiResponse(response)
		}

		var exists bool
		icResults := make(map[string]int)
		for _, r := range results {
			id := 0
			if idValue, exists := r["id"]; exists {
				if idFloat, ok := idValue.(float64); ok {
					id = int64(idFloat)
				}
			}
			if id <= 0 {
				return fmt.Errorf("Invalid ID in initiator group response: %d", id)
			}
			var name string
			if nameObj, exists := r["comment"]; exists {
				name, _ = nameObj.(string)
			}
			if name == "" {
				return fmt.Errorf("Could not find name in initiator group from iscsi.initiator.create response")
			}
			icResults[name] = id
		}
		for name, _ := range targetUpdates {
			if targetUpdates[name].initiatorId == -1 {
				targetUpdates[name].initiatorId, exists = icResults[name]
				if !exists {
					return fmt.Errorf("Could not find target \"%s\" in initiator group from iscsi.initiator.create response", name)
				}
			}
		}
		for name, _ := range targetCreates {
			if targetCreates[name].initiatorId == -1 {
				targetCreates[name].initiatorId, exists = icResults[name]
				if !exists {
					return fmt.Errorf("Could not find target \"%s\" in initiator group from iscsi.initiator.create response", name)
				}
			}
		}
	}

	if len(targetUpdates) > 0 {
		out, err := IscsiCreateOrUpdateTargets(
			api,
			"update",
			targetUpdates,
		)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	if len(targetCreates) > 0 {
		out, err := IscsiCreateOrUpdateTargets(
			api,
			"create",
			targetCreates,
		)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
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

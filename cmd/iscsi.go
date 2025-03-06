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

type typeUpdateIscsiTargetParams struct {
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

	defaultPortal := 0
	targetUpdates := make([]typeUpdateIscsiTargetParams, 0)

	for targetId, target := range response.resultsMap {
		targetName, _ := target["name"].(string)
		if targetName == "" {
			return fmt.Errorf("Name could not be found in iSCSI target with ID %d", targetId)
		}
		delete(toCreateMap, targetName)

		defaultInitiator := 0
		if groupsObj, exists := target["groups"]; exists {
			if groups, ok := groupsObj.([]interface{}); ok && len(groups) > 0 {
				for i := 0; i < len(groups); i++ {
					elem, isElemMap := groups[i].(map[string]interface{})
					portal := 0
					if _, ok := elem["portal"]; !isElemMap || !ok {
						if defaultPortal == 0 {
							defaultPortal, err = IscsiGetFirstPortal(api)
							if err != nil {
								return err
							}
						}
						portal = defaultPortal
					}
					initiator := 0
					if _, ok := elem["initiator"]; !isElemMap || !ok {
						if defaultInitiator == 0 {
							defaultInitiator, err = IscsiGetMatchingInitiatorGroup(api, targetName)
							if err != nil {
								return err
							}
							if defaultInitiator == -1 {
								initiatorCreates = append(initiatorCreates, targetName)
							}
						}
						initiator = defaultInitiator
					}
					if portal != 0 || initiator != 0 {
						targetUpdates = append(targetUpdates, typeUpdateIscsiTargetParams{
							id: targetId,
							groupIndex: i,
							portalId: portal,
							initiatorId: initiator,
						})
					}
				}
			}
		}
	}

	for _, targetName := range toCreateMap {
		if defaultPortal == 0 {
			defaultPortal, err = IscsiGetFirstPortal(api)
			if err != nil {
				return err
			}
		}
		defaultInitiator, err = IscsiGetMatchingInitiatorGroup(api, targetName)
		if err != nil {
			return err
		}
		if defaultInitiator == -1 {
			initiatorCreates = append(initiatorCreates, targetName)
		}
		targetCreates = append(targetCreates, typeUpdateIscsiTargetParams{
			id: targetId,
			groupIndex: i,
			portalId: defaultPortal,
			initiatorId: defaultInitiator,
		})
	}

	if len(targetUpdates) == 0 && len(targetCreates) == 0 {
		fmt.Println("iSCSI targets, portal and initiator groups are up to date for", args)
		return nil
	}

	if defaultPortal == -1 {
		defaultPortal, err = MakeIscsiPortal(api)
		if err != nil {
			return err
		}
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
			false
		)
		var response map[string]interface{}
		if err := json.Unmarshal(data, &response); err != nil {
			results := core.GetResultsListFromApiResponse(response)
		}
		/*
		idx := 0
		for i, p := range targetUpdates {
			if p.initiatorId == -1 {
				p.initiatorId = idx
			}
		}
		*/
	}

	if len(targetUpdates) > 0 {
		out, err := MaybeBulkApiCall(
			api,
			"iscsi.target.update",
			10,
			[]interface{}{paramsInitiator},
			objRemapInitiator,
			false
		)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	if len(targetCreates) > 0 {
		out, err := MaybeBulkApiCall(
			api,
			"iscsi.target.create",
			10,
			[]interface{}{paramsInitiator},
			objRemapInitiator,
			false
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

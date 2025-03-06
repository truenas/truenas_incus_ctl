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
	Aliases: []string{"set"},
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
						}
						initiator = defaultInitiator
					}
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

	fmt.Println(response.resultsMap)
	return nil
}

func activateIscsi(cmd *cobra.Command, api core.Session, args []string) error {
	if err := checkIscsiAdminToolExists(); err != nil {
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

func MakeIscsiTargetNameFromVolumePath(vol string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(strings.ToLower(vol), ":", "-"),
			".", "-"),
		"_", "-"),
	"/", ":")
}

func IscsiGetFirstPortal(api core.Session) (int, error) {
	return 0, nil
}

func IscsiGetMatchingInitiatorGroup(api core.Session, targetName string) (int, error) {
	return 0, nil
}

func checkIscsiAdminToolExists() error {
	_, err := exec.LookPath("iscsiadm")
	if err != nil {
		fmt.Println("Could not find iscsiadm in $PATH.\nMake sure that the open-iscsi package is installed on your system.")
	}
	return err
}

func runIscsiAdminTool(args []string) error {
	return exec.Command("iscsiadm", args...).Run()
}

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

type typeIscsiTargetParams struct {
	id interface{}
	groupIndex int
	portalId int
	initiatorId int
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

func IscsiGetTargetParams(
	api core.Session,
	targetName string,
	groupIndex int,
	initiatorCreates *[]string,
	targetParams map[string]typeIscsiTargetParams,
	defaultPortal *int,
	defaultInitiator *int,
	shouldLookupPortal bool,
	shouldLookupInitiator bool,
) error {
	portal := 0
	if shouldLookupPortal {
		if *defaultPortal == 0 {
			*defaultPortal, err = IscsiGetFirstPortal(api)
			if err != nil {
				return err
			}
		}
		portal = *defaultPortal
	}
	initiator := 0
	if shouldLookupInitiator {
		if *defaultInitiator == 0 {
			*defaultInitiator, err = IscsiGetMatchingInitiatorGroup(api, targetName)
			if err != nil {
				return err
			}
			if defaultInitiator == -1 {
				*initiatorCreates = append(*initiatorCreates, targetName)
			}
		}
		initiator = *defaultInitiator
	}
	if portal != 0 || initiator != 0 {
		targetParams[targetName] = typeIscsiTargetParams{
			id: targetId,
			groupIndex: groupIndex,
			portalId: portal,
			initiatorId: initiator,
		}
	}
	return nil
}

func IscsiCreateOrUpdateTargets() {
	return MaybeBulkApiCall(
		"iscsi.target." + verb,
		10,
		[]interface{}{paramsInitiator},
		objRemapInitiator,
		false,
	)
}

func CheckIscsiAdminToolExists() error {
	_, err := exec.LookPath("iscsiadm")
	if err != nil {
		fmt.Println("Could not find iscsiadm in $PATH.\nMake sure that the open-iscsi package is installed on your system.")
	}
	return err
}

func RunIscsiAdminTool(args []string) error {
	return exec.Command("iscsiadm", args...).Run()
}

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

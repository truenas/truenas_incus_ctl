package cmd

import (
	"os/exec"
	//"errors"
	"fmt"
	//"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	//"github.com/spf13/cobra"
)

func MakeIscsiTargetNameFromVolumePath(vol string) string {
	return "incus:" + strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(strings.ToLower(vol), ":", "-"),
			".", "-"),
		"_", "-"),
	"/", ":")
}

func AddIscsiInitiator(initiators map[string]int, resultRow map[string]interface{}) (string, error) {
	id := 0
	if idValue, exists := resultRow["id"]; exists {
		if idFloat, ok := idValue.(float64); ok {
			id = int(idFloat)
		}
	}
	if id <= 0 {
		return "", fmt.Errorf("Invalid ID in initiator group response: %d", id)
	}
	var name string
	if nameObj, exists := resultRow["comment"]; exists {
		name, _ = nameObj.(string)
	}
	if name == "" {
		return "", fmt.Errorf("Could not find any name in initiator group %d", id)
	}
	initiators[name] = id
	return name, nil
}

func CheckIscsiAdminToolExists() error {
	_, err := exec.LookPath("iscsiadm")
	if err != nil {
		fmt.Println("Could not find iscsiadm in $PATH.\nMake sure that the open-iscsi package is installed on your system.")
	}
	return err
}

func MaybeLaunchIscsiDaemon() error {
	// assuming a stable internet connection, iscsid as a command does not block.
	// it instead starts the actual daemon before returning immediately, without any console output.
	// returns 1 if the daemon could not be run or is already running, 0 otherwise.
	// since there's not enough information to determine if the daemon is actually active after this call, we might as well return nil.
	_ = exec.Command("iscsid").Run()
	return nil
}

func RunIscsiAdminTool(args []string) (string, error) {
	return core.RunCommand("iscsiadm", args...)
}

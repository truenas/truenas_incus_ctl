package cmd

import (
	"os"
	"os/exec"

	//"errors"
	"fmt"
	"log"
	//"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"
	//"github.com/spf13/cobra"
)

type typeIscsiLoginSpec struct {
	status   string
	remoteIp string
	iqn      string
	target   string
}

type typeApiCallRecord struct {
	endpoint string
	params []interface{}
	resultList []interface{}
	errorList []interface{}
}

func GetIscsiTargetPrefixOrExit(options map[string]string) string {
	prefix := options["target_prefix"]
	if prefix == "" {
		log.Fatal("Target prefix was not set")
	}
	const MAX_LENGTH = 27
	if len(prefix) > MAX_LENGTH {
		log.Fatal(fmt.Errorf("Target prefix exceeded maximum length of %d (was length %d)", MAX_LENGTH, len(prefix)))
	}
	return prefix
}

func MakeIscsiTargetNameFromVolumePath(prefix string, vol string) string {
	return prefix + ":" + strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(strings.ToLower(vol), ":", "-"),
				".", "-"),
			"_", "-"),
		"/", ":")
}

func MakeIscsiTargetUuid(prefix, targetName string) string {
	if targetName == "" {
		return ""
	}
	return prefix + ":" + core.MakeHashedUuid(targetName)
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

func LocateIqnTargetsLocally(targets []typeIscsiLoginSpec) []string {
	output := make([]string, 0)
	diskEntries, err := os.ReadDir("/dev/disk/by-path")
	if err != nil {
		return output
	}
	for _, e := range diskEntries {
		name := e.Name()
		for _, t := range targets {
			if strings.HasSuffix(name, t.iqn+":"+t.target+"-lun-0") {
				path := "/dev/disk/by-path/" + name
				var line string
				if t.status != "" {
					line = t.status + ": " + path
				} else {
					line = path
				}
				output = append(output, line)
			}
		}
	}
	return output
}

func GetIscsiTargetsFromDiscovery(iscsiToVolumeMap map[string]string, portalAddr string) ([]typeIscsiLoginSpec, error) {
	out, err := RunIscsiAdminTool([]string{"--mode", "discoverydb", "--type", "sendtargets", "--portal", portalAddr, "--discover"})
	if err != nil {
		return nil, err
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
			t.iqn = l[spacePos+1 : commaPos+iqnSepPos]
			t.target = targetName
			targets = append(targets, t)
		}
	}

	return targets, nil
}

func GetIscsiTargetsFromSession(iscsiToVolumeMap map[string]string) ([]typeIscsiLoginSpec, error) {
	out, err := RunIscsiAdminTool([]string{"--mode", "session"})
	if err != nil {
		return nil, err
	}

	targets := make([]typeIscsiLoginSpec, 0)
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		firstSpacePos := strings.Index(l, " ")
		if firstSpacePos == -1 {
			continue
		}
		lastSpacePos := strings.LastIndex(l, " ")
		if lastSpacePos == firstSpacePos {
			lastSpacePos = len(l)
		}
		fullName := l[firstSpacePos+1 : lastSpacePos]
		firstColon := strings.Index(fullName, ":")
		if firstColon == -1 {
			continue
		}
		targetName := fullName[firstColon+1:]
		if _, exists := iscsiToVolumeMap[targetName]; exists {
			iqnName := fullName[0:firstColon]
			targets = append(targets, typeIscsiLoginSpec{
				remoteIp: "",
				iqn:      iqnName,
				target:   targetName,
			})
		}
	}

	return targets, nil
}

func CheckIscsiAdminToolExists() error {
	_, err := exec.LookPath("iscsiadm")
	if err != nil {
		fmt.Println("Could not find iscsiadm in $PATH.\nMake sure that the open-iscsi package is installed on your system.")
	}
	return err
}

// TODO: Wait for daemon to ACTUALLY finish launching.
// running an iscsiadm command immediately after launching the daemon results in an error for some reason.
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

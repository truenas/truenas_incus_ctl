package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
	"truenas/truenas_incus_ctl/core"
)

type typeIscsiLoginSpec struct {
	remoteIp string
	iqn      string
	target   string
}

type typeApiCallRecord struct {
	endpoint   string
	params     []interface{}
	resultList []interface{}
	errorList  []interface{}
	priorQuery *typeQueryResponse
}

func LookupPortalByObject(api core.Session, toMatch interface{}) (int, error) {
	queryFilter := []interface{}{[]interface{}{"listen", "=", toMatch}}
	queryParams := []interface{}{
		queryFilter,
		make(map[string]interface{}),
	}
	out, err := core.ApiCall(api, "iscsi.portal.query", 10, queryParams)
	if err != nil {
		return -1, err
	}
	var response map[string]interface{}
	if err = json.Unmarshal(out, &response); err != nil {
		return -1, err
	}
	results, _ := response["result"].([]interface{})
	for i := 0; i < len(results); i++ {
		idObj := core.GetIdFromObject(results[i])
		if n, errNotNumber := strconv.Atoi(fmt.Sprint(idObj)); errNotNumber == nil {
			return n, nil
		}
	}
	return -1, nil
}

func LookupPortalIdOrCreate(api core.Session, defaultPort int, spec string) (int, error) {
	if spec == "" {
		return -1, fmt.Errorf("Portal was not specified (use ':' for the default portal)")
	}
	if asInt, errNotNumber := strconv.Atoi(spec); errNotNumber == nil {
		return asInt, nil
	}

	ipPortStr := core.IpPortToJsonString(spec, api.GetHostName(), defaultPort)
	var ipPortObj interface{}
	if err := json.Unmarshal([]byte(ipPortStr), &ipPortObj); err != nil {
		return -1, err
	}
	portalId, err := LookupPortalByObject(api, ipPortObj)
	if err != nil {
		return -1, err
	}

	if portalId == -1 {
		if ipPortArray, ok := ipPortObj.([]interface{}); ok && len(ipPortArray) > 0 {
			if ipPortMap, ok := ipPortArray[0].(map[string]interface{}); ok {
				delete(ipPortMap, "port")
				ipPortArray[0] = ipPortMap
				ipPortObj = ipPortArray
			}
		}
		paramsCreate := []interface{}{map[string]interface{}{"listen": ipPortObj}}
		DebugJson(paramsCreate)

		out, err := core.ApiCall(api, "iscsi.portal.create", 10, paramsCreate)
		if err != nil {
			return -1, err
		}
		resCreate, _ := core.GetResultsAndErrorsFromApiResponseRaw(out)
		if len(resCreate) > 0 {
			idObj := core.GetIdFromObject(resCreate[0])
			if n, errNotNumber := strconv.Atoi(fmt.Sprint(idObj)); errNotNumber == nil {
				portalId = n
			}
		}
		if portalId == -1 {
			return LookupPortalByObject(api, ipPortObj)
		}
	}

	return portalId, nil
}

func MaybeLookupIpPortFromPortal(api core.Session, defaultPort int, spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("Portal was not specified (use ':' for the default portal)")
	}

	var ipPortObjMap map[string]interface{}
	if asInt, errNotNumber := strconv.Atoi(spec); errNotNumber == nil {
		queryFilter := []interface{}{[]interface{}{"id", "=", asInt}}
		queryParams := []interface{}{
			queryFilter,
			make(map[string]interface{}),
		}
		out, err := core.ApiCall(api, "iscsi.portal.query", 10, queryParams)
		if err != nil {
			return "", err
		}
		var response map[string]interface{}
		if err = json.Unmarshal(out, &response); err != nil {
			return "", err
		}
		results, _ := response["result"].([]interface{})
		for i := 0; i < len(results); i++ {
			if obj, ok := results[i].(map[string]interface{}); ok {
				if listenArray, ok := obj["listen"].([]interface{}); ok && len(listenArray) > 0 {
					ipPortObjMap, _ = listenArray[0].(map[string]interface{})
					break
				} else if listenMap, ok := obj["listen"].(map[string]interface{}); ok {
					ipPortObjMap = listenMap
					break
				}
			}
		}
	}

	if ipPortObjMap == nil {
		ipPortStr := core.IpPortToJsonString(spec, api.GetHostName(), defaultPort)
		var obj interface{}
		if err := json.Unmarshal([]byte(ipPortStr), &obj); err != nil {
			return "", err
		}
		if objArray, isArray := obj.([]interface{}); isArray {
			if len(objArray) > 0 {
				obj = objArray[0]
			} else {
				return "", fmt.Errorf("listen object was empty")
			}
		}
		if objMap, isMap := obj.(map[string]interface{}); isMap {
			ipPortObjMap = objMap
		} else {
			return "", fmt.Errorf("listen object was not a map or array of map")
		}
	}

	ip, exists := ipPortObjMap["ip"]
	if !exists {
		ip = core.ResolvedIpv4OrVerbatim(api.GetHostName())
	}
	port, exists := ipPortObjMap["port"]
	if !exists {
		port = defaultPort
	}

	return fmt.Sprintf("%v:%v", ip, port), nil
}

func LookupInitiatorByFilter(api core.Session, queryFilter []interface{}) (int, error) {
	queryParams := []interface{}{
		queryFilter,
		make(map[string]interface{}),
	}
	out, err := core.ApiCall(api, "iscsi.initiator.query", 10, queryParams)
	if err != nil {
		return -1, err
	}
	var response map[string]interface{}
	if err = json.Unmarshal(out, &response); err != nil {
		return -1, err
	}
	results, _ := response["result"].([]interface{})
	for i := 0; i < len(results); i++ {
		idObj := core.GetIdFromObject(results[i])
		if n, errNotNumber := strconv.Atoi(fmt.Sprint(idObj)); errNotNumber == nil {
			return n, nil
		}
	}
	return -1, nil
}

func LookupInitiatorOrCreateBlank(api core.Session, spec string) (int, error) {
	if asInt, errNotNumber := strconv.Atoi(spec); errNotNumber == nil {
		return asInt, nil
	}
	queryFilter := make([]interface{}, 0)
	if spec != "" {
		queryFilter = append(queryFilter, []interface{}{"comment", "=", spec})
	}
	initiatorId, err := LookupInitiatorByFilter(api, queryFilter)
	if err != nil {
		return -1, err
	}

	if initiatorId == -1 {
		out, err := core.ApiCall(api, "iscsi.initiator.create", 10, []interface{}{map[string]interface{}{"comment": spec}})
		if err != nil {
			return -1, err
		}
		resCreate, _ := core.GetResultsAndErrorsFromApiResponseRaw(out)
		if len(resCreate) > 0 {
			idObj := core.GetIdFromObject(resCreate[0])
			if n, errNotNumber := strconv.Atoi(fmt.Sprint(idObj)); errNotNumber == nil {
				initiatorId = n
			}
		}
		if initiatorId == -1 {
			return LookupInitiatorByFilter(api, queryFilter)
		}
	}

	return initiatorId, nil
}

func TryDeferIscsiApiCallArray(deferNotSupportedRef *bool, api core.Session, method string, timeoutSeconds int64, paramsArray []interface{}) ([]interface{}, []interface{}, error) {
	nCalls := len(paramsArray)
	if nCalls == 0 {
		return nil, nil, errors.New("TryDeferIscsiApiCallArray: Nothing to do")
	}
	if deferNotSupportedRef != nil && *deferNotSupportedRef {
		DebugString("defer not supported - aware")
		out, _, err := MaybeBulkApiCallArray(api, method, timeoutSeconds, paramsArray, true)
		if err != nil {
			return nil, nil, err
		}
		res, errs := core.GetResultsAndErrorsFromApiResponseRaw(out)
		return res, errs, nil
	}

	deferParamsArray, _ := core.DeepCopy(paramsArray).([]interface{})
	if strings.HasSuffix(method, ".delete") {
		for i := 0; i < nCalls; i++ {
			if paramsList, ok := deferParamsArray[i].([]interface{}); ok {
				deferParamsArray[i] = append(paramsList, true)
			}
		}
	} else if strings.HasSuffix(method, ".update") {
		for i := 0; i < nCalls; i++ {
			if paramsList, ok := deferParamsArray[i].([]interface{}); ok {
				if paramsMap, ok := paramsList[1].(map[string]interface{}); ok {
					paramsMap["defer"] = true
					paramsList[1] = paramsMap
				}
				deferParamsArray[i] = paramsList
			}
		}
	} else {
		for i := 0; i < nCalls; i++ {
			if paramsList, ok := deferParamsArray[i].([]interface{}); ok {
				if paramsMap, ok := paramsList[0].(map[string]interface{}); ok {
					paramsMap["defer"] = true
					paramsList[0] = paramsMap
				}
				deferParamsArray[i] = paramsList
			}
		}
	}

	if err := core.MaybeLogin(api); err != nil {
		return nil, nil, err
	}
	out, err := api.CallRaw(method, timeoutSeconds, deferParamsArray[0])
	if err != nil {
		return nil, nil, err
	}

	resultList, errorList := core.GetResultsAndErrorsFromApiResponseRaw(out)

	//DebugJson(resultList)
	//DebugJson(errorList)

	if len(errorList) > 0 {
		lower := strings.ToLower(core.ExtractApiErrorJsonGivenError(errorList[0]))
		if strings.Contains(lower, "too many arguments") || strings.Contains(lower, "extra inputs are not") {
			DebugString("defer not supported - was not aware")
			if deferNotSupportedRef != nil {
				*deferNotSupportedRef = true
			}
			out, _, err = MaybeBulkApiCallArray(api, method, timeoutSeconds, paramsArray, true)
			if err != nil {
				return nil, nil, err
			}
			res, errs := core.GetResultsAndErrorsFromApiResponseRaw(out)
			return res, errs, nil
		}
	}
	if nCalls >= 2 {
		out, _, err = MaybeBulkApiCallArray(api, method, timeoutSeconds, deferParamsArray[1:nCalls], true)
		if err != nil {
			return nil, nil, err
		}
		subResults, subErrors := core.GetResultsAndErrorsFromApiResponseRaw(out)
		resultList = append(resultList, subResults)
		errorList = append(errorList, subErrors)
	}
	return resultList, errorList, nil
}

func GetIscsiTargetPrefixOrExit(options map[string]string) string {
	prefixRaw := options["target_prefix"]
	prefix := strings.TrimSpace(prefixRaw)
	const MAX_LENGTH = 24
	if len(prefix) > MAX_LENGTH {
		log.Fatal(fmt.Errorf("Target prefix exceeded maximum length of %d (was length %d)", MAX_LENGTH, len(prefix)))
	}
	return prefix
}

func MakeIscsiTargetNameFromVolumePath(prefix, vol string) string {
	var substituted strings.Builder
	for _, r := range vol {
		if r == ':' || r == '.' || r == '_' {
			r = '-'
		}
		if r == '/' || r == '@' {
			r = ':'
		}
		substituted.WriteRune(r)
	}
	if prefix == "" {
		return strings.ToLower(substituted.String())
	}
	return strings.ToLower(prefix + ":" + substituted.String())
}

func MaybeHashIscsiNameFromVolumePath(prefix, vol string) string {
	iscsiName := MakeIscsiTargetNameFromVolumePath(prefix, vol)
	if len(iscsiName) > 64 {
		var begin string
		if prefix == "" {
			begin = "-:"
		} else {
			begin = prefix + ":-:"
		}
		return begin + core.MakeHashedString(vol, 64-len(begin))
	}
	return iscsiName
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

func GetIscsiSharesFromSessionAndDiscovery(api core.Session, prefixName string, isCreate bool, args []string, portalAddr string) (map[string]bool, map[string]string, error) {
	maybeHashedToVolumeMap := make(map[string]string)
	missingShares := make(map[string]string)

	for _, vol := range args {
		maybeHashed := MaybeHashIscsiNameFromVolumePath(prefixName, vol)
		maybeHashedToVolumeMap[maybeHashed] = vol
		missingShares[maybeHashed] = vol
	}

	var err error
	if err = CheckIscsiAdminToolExists(); err != nil {
		return nil, nil, err
	}

	err = MaybeLaunchIscsiDaemon()
	if err != nil {
		return nil, nil, err
	}

	discoveryTargets, err := GetIscsiTargetsFromDiscovery(api, maybeHashedToVolumeMap, portalAddr)
	//fmt.Println(discoveryTargets, err)
	if !isCreate && err != nil {
		return nil, nil, err
	}

	sessionTargets, err := GetIscsiTargetsFromSession(api, maybeHashedToVolumeMap)
	//fmt.Println(sessionTargets, err)
	if !isCreate && err != nil && !strings.Contains(strings.ToLower(err.Error()), "no active sessions") {
		return nil, nil, err
	}

	targets := make([]typeIscsiLoginSpec, 0)
	targets = append(targets, sessionTargets...)
	targets = append(targets, discoveryTargets...)

	if len(targets) == 0 {
		return nil, missingShares, nil
	}

	shares := make(map[string]bool)
	for _, t := range targets {
		shares[t.iqn+":"+t.target] = true
		delete(missingShares, t.target)
	}

	return shares, missingShares, nil
}

func IterateActivatedIscsiShares(optIpPortalAddr string, callback func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string)) {
	diskEntries, err := os.ReadDir("/dev/disk/by-path")
	if err != nil {
		return
	}
	for _, e := range diskEntries {
		name := e.Name()
		suffix := "-lun-0"
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		pathPrefix := "ip-" + optIpPortalAddr
		if !strings.HasPrefix(name, pathPrefix) {
			continue
		}

		iqnPathStart := "-iscsi-iqn."
		var pathStartPos int
		var ipPortalAddr string

		if len(optIpPortalAddr) == 0 {
			pathStartPos = strings.Index(name, iqnPathStart)
			if pathStartPos == -1 {
				continue
			}
			ipPortalAddr = name[3:pathStartPos]
		} else {
			pathStartPos = len(pathPrefix)
			if !strings.HasPrefix(name[pathStartPos:], iqnPathStart) {
				continue
			}
			ipPortalAddr = optIpPortalAddr
		}

		iqnStart := pathStartPos + len(iqnPathStart) - 4
		iqnTargetName := name[iqnStart : len(name)-len(suffix)]
		targetOnlyName := iqnTargetName[strings.Index(iqnTargetName, ":")+1:]

		callback("/dev/disk/by-path", name, ipPortalAddr, iqnTargetName, targetOnlyName)
	}
}

func DeactivateIscsiTargetList(api core.Session, ipPortalAddr string, toDeactivate []string, shouldWait bool) ([]string, []error) {
	results := make([]string, 0)
	errs := make([]error, 0)

	toDeactivateMap := make(map[string]bool)
	for _, t := range toDeactivate {
		toDeactivateMap[t] = true
	}

	toSyncList := make([]string, 0)
	IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
		if _, exists := toDeactivateMap[iqnTargetName]; exists {
			fullPath := path.Join(root, fullName)
			toSyncList = append(toSyncList, fullPath)
		}
	})

	if len(toSyncList) > 0 {
		_, _ = core.RunCommand("sync", append([]string{"-f"}, toSyncList...)...)
	}

	for _, t := range toDeactivate {
		if err := RunIscsiDeactivate(api, t, ipPortalAddr); err != nil {
			errs = append(errs, fmt.Errorf("%s\t%v", t, err.Error()))
		} else {
			results = append(results, t)
		}
	}
	if len(results) == 0 {
		results = nil
	}
	if len(errs) == 0 {
		errs = nil
	}

	if !shouldWait || results == nil {
		return results, errs
	}

	statusCh := make(chan bool)
	innerMap := make(map[string]bool)
	outerMap := make(map[string]bool)
	for _, t := range results {
		innerMap[t] = true
		outerMap[t] = true
	}

	go func() {
		core.WaitForCreatedDeletedFiles("/dev/disk/by-path", func(fname string, wasCreate, wasDelete bool) bool {
			isDone := true
			IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
				if _, exists := innerMap[iqnTargetName]; exists {
					isDone = false
				}
			})
			statusCh <- isDone
			return isDone
		})
	}()

	const maxTries = 30
	for i := 0; i < maxTries; i++ {
		select {
		case isDone := <-statusCh:
			if isDone {
				return results, errs
			}
		case <-time.After(time.Duration(1000) * time.Millisecond):
			isDone := true
			IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
				if _, exists := outerMap[iqnTargetName]; exists {
					isDone = false
				}
			})
			if isDone {
				return results, errs
			}
		}
	}

	return results, errs
}

func DeactivateMatchingIscsiTargets(
	api core.Session,
	ipPortalAddr string,
	maybeHashedToVolumeMap map[string]string,
	shouldWait bool,
	isMinimal bool,
	shouldPrintStatus bool,
) []string {
	toDeactivateMap := make(map[string]string)
	toDeactivateIqnTargets := make([]string, 0)
	IterateActivatedIscsiShares(ipPortalAddr, func(root string, fullName string, ipPortalAddr string, iqnTargetName string, targetOnlyName string) {
		if _, exists := maybeHashedToVolumeMap[targetOnlyName]; exists {
			toDeactivateMap[iqnTargetName] = targetOnlyName
			toDeactivateIqnTargets = append(toDeactivateIqnTargets, iqnTargetName)
		} else if !isMinimal {
			fmt.Println("not-found\t" + targetOnlyName)
		}
	})

	deactivatedIqnTargetList, errorList := DeactivateIscsiTargetList(api, ipPortalAddr, toDeactivateIqnTargets, shouldWait)
	for _, e := range errorList {
		fmt.Printf("failed\t%v\n", e)
	}

	deactivatedTargets := make([]string, 0)
	for _, t := range deactivatedIqnTargetList {
		if shouldPrintStatus || !isMinimal {
			fmt.Println("deactivated\t", t)
		} else {
			fmt.Println(t)
		}
		deactivatedTargets = append(deactivatedTargets, toDeactivateMap[t])
	}

	return deactivatedTargets
}

func GetIscsiTargetsFromDiscovery(api core.Session, maybeHashedToVolumeMap map[string]string, portalAddr string) ([]typeIscsiLoginSpec, error) {
	out, err := RunIscsiDiscover(api, portalAddr)
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
		if _, exists := maybeHashedToVolumeMap[targetName]; exists {
			t := typeIscsiLoginSpec{}
			t.remoteIp = l[0:commaPos]
			t.iqn = l[spacePos+1 : commaPos+iqnSepPos]
			t.target = targetName
			targets = append(targets, t)
		}
	}

	return targets, nil
}

func GetIscsiTargetsFromSession(api core.Session, maybeHashedToVolumeMap map[string]string) ([]typeIscsiLoginSpec, error) {
	out, err := RunIscsiAdminTool(api, []string{"--mode", "session"})
	if err != nil {
		return nil, err
	}

	targets := make([]typeIscsiLoginSpec, 0)
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		firstEndBracket := strings.Index(l, "]")
		if firstEndBracket == -1 {
			continue
		}
		addrStart := firstEndBracket + 2
		firstSpacePos := strings.Index(l[addrStart:], " ")
		if firstSpacePos == -1 {
			continue
		}
		firstCommaPos := strings.Index(l[addrStart:], ",")
		if firstCommaPos == -1 || firstCommaPos > firstSpacePos {
			firstCommaPos = firstSpacePos
		}
		lastSpacePos := strings.LastIndex(l, " ")
		if lastSpacePos == firstSpacePos {
			lastSpacePos = len(l)
		}
		ipPortalAddr := l[addrStart:firstCommaPos]
		fullName := l[addrStart+firstSpacePos+1 : lastSpacePos]
		firstColon := strings.Index(fullName, ":")
		if firstColon == -1 {
			continue
		}
		targetName := fullName[firstColon+1:]

		if _, exists := maybeHashedToVolumeMap[targetName]; exists {
			iqnName := fullName[0:firstColon]
			targets = append(targets, typeIscsiLoginSpec{
				remoteIp: ipPortalAddr,
				iqn:      iqnName,
				target:   targetName,
			})
		}
	}

	return targets, nil
}

func CheckRemoteIscsiServiceIsRunning(api core.Session) (string, error) {
	out, err := core.ApiCall(api, "service.started", 10, []interface{}{"iscsitarget"})
	if err != nil {
		return "", err
	}
	var response map[string]interface{}
	if err = json.Unmarshal(out, &response); err != nil {
		return "", err
	}
	if !core.IsValueTrue(response, "result") {
		return "The iSCSI service has not been started\nRun this tool with:\nservice start --enable iscsitarget\nTo start the service", nil
	}
	return "", nil
}

func RunIscsiActivate(api core.Session, iqnTargetName string, ipPortalAddr string) error {
	loginParams := []string{
		"--mode",
		"node",
		"--targetname",
		iqnTargetName,
		"--portal",
		ipPortalAddr,
		"--login",
	}
	DebugString(strings.Join(loginParams, " "))
	_, err := RunIscsiAdminTool(api, loginParams)
	return err
}

func RunIscsiDeactivate(api core.Session, iqnTargetName string, ipPortalAddr string) error {
	logoutParams := []string{
		"--mode",
		"node",
		"--targetname",
		iqnTargetName,
		"--portal",
		ipPortalAddr,
		"--logout",
	}
	DebugString(strings.Join(logoutParams, " "))
	_, err := RunIscsiAdminTool(api, logoutParams)
	return err
}

func RunIscsiDiscover(api core.Session, portalAddr string) (string, error) {
	return RunIscsiAdminTool(api, []string{"--mode", "discoverydb", "--type", "sendtargets", "--discover", "--portal", portalAddr})
}

func TestIscsiDiscovery(api core.Session, portalAddr string) (string, error) {
	RunIscsiDiscover(api, portalAddr)
	return RunIscsiAdminTool(api, []string{"--mode", "discovery", "--portal", portalAddr})
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

func RunIscsiAdminTool(api core.Session, args []string) (string, error) {
	retriesLeft := 10
begin:
	out, err := core.RunCommand("iscsiadm", args...)
	// "Could not stat" seems to happen when iscsiadm decides to delete a node... and another instance deletes the node, a retry should resolve.
	if err != nil && (strings.HasPrefix(err.Error(), "iscsiadm: Could not scan /sys/class/iscsi_transport") || strings.HasPrefix(err.Error(), "iscsiadm: Could not stat") || strings.HasPrefix(err.Error(), "iscsiadm: Could not lookup devpath") || strings.HasPrefix(err.Error(), "iSCSI ERROR: realpath() failed") || strings.HasPrefix(err.Error(), "iSCSI ERROR: Got unexpected(should be")) {
		time.Sleep(time.Duration(500) * time.Millisecond)
		retriesLeft--
		if retriesLeft > 0 {
			goto begin
		}
	}
	if err != nil {
		msg, apiErr := CheckRemoteIscsiServiceIsRunning(api)
		if apiErr == nil {
			if msg != "" {
				err = errors.New(err.Error() + "\n" + msg)
			} else {
				err = errors.New(err.Error() + "\nThe iscsitarget service is running. It may need to be restarted with:\nservice restart iscsitarget")
			}
		}
	}
	return out, err
}

package cmd

import (
	"testing"
)

func TestSnapshotClone(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		snapshotCloneCmd,
		cloneSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test4@readonly","dozer/testing/test5@readonly"},
		"[{\"dataset_dst\":\"dozer/testing/test5@readonly\",\"snapshot\":\"dozer/testing/test4@readonly\"}]",
	))
}

func TestSnapshotCreate(t *testing.T) {
	/*
	MaybeCopyProperty(outMap, options.allFlags, "recursive")
	MaybeCopyProperty(outMap, options.usedFlags, "suspend_vms")
	MaybeCopyProperty(outMap, options.usedFlags, "vmware_sync")

	if excludeStr := options.allFlags["exclude"]; excludeStr != "" {
		outMap["exclude"] = strings.Split(excludeStr, ",")
	}

	outProps := make(map[string]interface{})
	_ = WriteKvArrayToMap(outProps, ConvertParamsStringToKvArray(options.allFlags["option"]), nil)
	outMap["properties"] = outProps
	*/
}

func TestSnapshotDelete(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		snapshotDeleteCmd,
		deleteOrRollbackSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test3@readonly"},
		"[\"dozer/testing/test3@readonly\",{}]",
	))
}

func TestSnapshotList(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{},
		[]string{},
		[]string{"[[],{\"extra\":{\"flat\":false,\"properties\":[],\"retrieve_children\":true,\"user_properties\":false}}]"}, // expected
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"},"+ //response
			"{\"id\":\"dozer/testing/test5@readonly\",\"name\":\"dozer/testing/test5@readonly\"}],\"id\":2}"},
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n" +
		" dozer/testing/test5@readonly \n",
	))
}

func TestSnapshotListParameter(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test4@readonly"},
		[]string{"[[[\"name\",\"in\",[\"dozer/testing/test4@readonly\"]]],"+ // expected
			"{\"extra\":{\"flat\":false,\"properties\":[],\"retrieve_children\":false,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"}],\"id\":2}"}, // response
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n",
	))
}

func TestSnapshotListTwoParameters(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test4@readonly","dozer/testing/test5@readonly"},
		[]string{"[[[\"name\",\"in\",[\"dozer/testing/test4@readonly\",\"dozer/testing/test5@readonly\"]]],"+ // expected
			"{\"extra\":{\"flat\":false,\"properties\":[],\"retrieve_children\":false,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"},"+ //response
			"{\"id\":\"dozer/testing/test5@readonly\",\"name\":\"dozer/testing/test5@readonly\"}],\"id\":2}"},
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n" +
		" dozer/testing/test5@readonly \n",
	))
}

func TestSnapshotListDataset(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test4"},
		[]string{"[[[\"dataset\",\"in\",[\"dozer/testing/test4\"]]],{\"extra\":{\"flat\":false,"+ // expected
			"\"properties\":[],\"retrieve_children\":false,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"}],\"id\":2}"}, // response
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n",
	))
}

func TestSnapshotListSnapshot(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{},
		[]string{"@readonly"},
		[]string{"[[[\"snapshot_name\",\"in\",[\"readonly\"]]],{\"extra\":{\"flat\":false,"+ // expected
			"\"properties\":[],\"retrieve_children\":false,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"},"+ //response
			"{\"id\":\"dozer/testing/test5@readonly\",\"name\":\"dozer/testing/test5@readonly\"}],\"id\":2}"},
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n" +
		" dozer/testing/test5@readonly \n",
	))
}

func TestSnapshotListRecursive(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{"recursive":true},
		[]string{"dozer/testing"},
		[]string{"[[[\"OR\",[[\"dataset\",\"=\",\"dozer/testing\"],[\"dataset\",\"^\",\"dozer/testing/\"]]]],"+ // expected
			"{\"extra\":{\"flat\":false,\"properties\":[],\"retrieve_children\":true,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\"},"+ //response
			"{\"id\":\"dozer/testing/test5@readonly\",\"name\":\"dozer/testing/test5@readonly\"}],\"id\":2}"},
		"             name             \n" + // table
		"------------------------------\n" +
		" dozer/testing/test4@readonly \n" +
		" dozer/testing/test5@readonly \n",
	))
}

func TestSnapshotListWithProperties(t *testing.T) {
	FailIf(t, DoTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{"no-headers":true,"parseable":true,"output":"name,clones"},
		[]string{"dozer/testing/test4@readonly"},
		[]string{"[[[\"name\",\"in\",[\"dozer/testing/test4@readonly\"]]],{\"extra\":{\"flat\":false,"+
			"\"properties\":[\"name\",\"clones\"],\"retrieve_children\":false,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\","+
			"\"properties\":{\"clones\":{\"rawvalue\":\"dozer/testing/test\",\"value\":\"dozer/testing/test\",\"parsed\":\"dozer/testing/test\"}}}],\"id\":2}"},
		"dozer/testing/test4@readonly\tdozer/testing/test\n",
	))
}

func TestSnapshotRename(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		snapshotRenameCmd,
		renameSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test@readonly", "dozer/testing/test3@readonly"},
		"[\"dozer/testing/test@readonly\",{\"new_name\":\"dozer/testing/test3@readonly\"}]",
	))
}

func TestSnapshotRollback(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		snapshotRollbackCmd,
		deleteOrRollbackSnapshot,
		map[string]interface{}{},
		[]string{"dozer/testing/test3@readonly"},
		"[\"dozer/testing/test3@readonly\",{}]",
	))
}

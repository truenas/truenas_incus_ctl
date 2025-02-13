package cmd

import (
	"testing"
)

func TestSnapshotClone(t *testing.T) {

}

func TestSnapshotCreate(t *testing.T) {

}

func TestSnapshotDelete(t *testing.T) {

}

func TestSnapshotList(t *testing.T) {

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

}

func TestSnapshotRollback(t *testing.T) {

}

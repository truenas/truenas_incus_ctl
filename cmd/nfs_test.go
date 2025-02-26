package cmd

import (
	"testing"
)

func TestNfsCreate(t *testing.T) {

}

func TestNfsCreateBulk(t *testing.T) {
	
}

func TestNfsUpdate(t *testing.T) {

}

func TestNfsUpdateBulk(t *testing.T) {

}

func TestNfsUpdateOrCreate(t *testing.T) {

}

func TestNfsUpdateOrCreateBulk(t *testing.T) {

}

func TestNfsDelete(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		nfsDeleteCmd,
		deleteNfs,
		map[string]interface{}{},
		[]string{"3"},
		"[3]",
	))
}

func TestNfsDeleteWithLookup(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsDeleteCmd,
		deleteNfs,
		map[string]interface{}{},
		[]string{"dozer/testing/test4"},
		[]string{
			"[[[\"path\",\"in\",[\"/mnt/dozer/testing/test4\"]]]]",
			"[4]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}",
			"{}",
		},
		"",
	))
}

func TestNfsDeleteBulk(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		nfsDeleteCmd,
		deleteNfs,
		map[string]interface{}{},
		[]string{"3","4"},
		"[\"sharing.nfs.delete\",[[3],[4]]]",
	))
}

func TestNfsDeleteBulkWithLookup(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsDeleteCmd,
		deleteNfs,
		map[string]interface{}{},
		[]string{"3","dozer/testing/test4"},
		[]string{
			"[[[\"OR\",[[\"id\",\"in\",[3]],[\"path\",\"in\",[\"/mnt/dozer/testing/test4\"]]]]]]",
			"[\"sharing.nfs.delete\",[[3],[4]]]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":3,\"path\":\"/mnt/dozer/testing/test3\"},{\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}",
			"{}",
		},
		"",
	))
}

func TestNfsList(t *testing.T) {

}

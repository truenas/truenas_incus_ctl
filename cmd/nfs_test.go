package cmd

import (
	"testing"
)

func TestNfsCreate(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		nfsCreateCmd,
		createNfs,
		map[string]interface{}{},
		[]string{"dozer/testing/test3"},
		"[{\"path\":\"/mnt/dozer/testing/test3\"}]",
	))
}

func TestNfsCreateWithId(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		nfsCreateCmd,
		createNfs,
		map[string]interface{}{},
		[]string{"3"},
		"Unrecognized nfs create spec \"3\"",
	))
}

func TestNfsCreateBulk(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		nfsCreateCmd,
		createNfs,
		map[string]interface{}{"read-only":true},
		[]string{"dozer/testing/test3","/mnt/dozer/testing/test4"},
		"[\"sharing.nfs.create\",[[{\"path\":\"/mnt/dozer/testing/test3\",\"ro\":true}],[{\"path\":\"/mnt/dozer/testing/test4\",\"ro\":true}]]]",
	))
}

func TestNfsUpdate(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsUpdateCmd,
		updateNfs,
		map[string]interface{}{"comment":"bar"},
		[]string{"dozer/testing/test4"},
		[]string{
			"[[[\"path\",\"in\",[\"/mnt/dozer/testing/test4\"]]]]",
			"[4,{\"comment\":\"bar\"}]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"comment\":\"foo\",\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}",
			"{}",
		},
		"",
	))
}

func TestNfsUpdateBulk(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsUpdateCmd,
		updateNfs,
		map[string]interface{}{"comment":"bar"},
		[]string{"3","dozer/testing/test4","/mnt/dozer/testing/test5"},
		[]string{
			"[[[\"OR\",[[\"id\",\"in\",[3]],[\"path\",\"in\",[\"/mnt/dozer/testing/test4\",\"/mnt/dozer/testing/test5\"]]]]]]",
			"[\"sharing.nfs.update\",[[3,{\"comment\":\"bar\"}],[5,{\"comment\":\"bar\"}]]]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"comment\":\"foo\",\"id\":3,\"path\":\"/mnt/dozer/testing/test3\"},"+
				"{\"comment\":\"bar\",\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"},"+
				"{\"comment\":\"foo\",\"id\":5,\"path\":\"/mnt/dozer/testing/test5\"}],\"id\":2}",
			"{}",
		},
		"",
	))
}

func TestNfsUpdateOrCreate(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsUpdateCmd,
		updateNfs,
		map[string]interface{}{"create":true,"comment":"bar"},
		[]string{"dozer/testing/test4","/mnt/dozer/testing/test5"},
		[]string{
			"[[[\"path\",\"in\",[\"/mnt/dozer/testing/test4\",\"/mnt/dozer/testing/test5\"]]]]",
			"[4,{\"comment\":\"bar\"}]",
			"[{\"comment\":\"bar\",\"path\":\"/mnt/dozer/testing/test5\"}]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"comment\":\"foo\",\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}",
			"{}",
			"{}",
		},
		"",
	))
}

func TestNfsUpdateOrCreateBulk(t *testing.T) {
	FailIf(t, DoTest(
		t,
		nfsUpdateCmd,
		updateNfs,
		map[string]interface{}{"create":true,"comment":"bar"},
		[]string{"/mnt/dozer/testing/test3","4","dozer/testing/test5","/mnt/dozer/testing/test6"},
		[]string{
			"[[[\"OR\",[[\"path\",\"in\",[\"/mnt/dozer/testing/test3\",\"/mnt/dozer/testing/test5\","+
				"\"/mnt/dozer/testing/test6\"]],[\"id\",\"in\",[4]]]]]]",
			"[\"sharing.nfs.update\",[[3,{\"comment\":\"bar\"}],[4,{\"comment\":\"bar\"}]]]",
			"[\"sharing.nfs.create\",[[{\"comment\":\"bar\",\"path\":\"/mnt/dozer/testing/test5\"}],"+
				"[{\"comment\":\"bar\",\"path\":\"/mnt/dozer/testing/test6\"}]]]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"comment\":\"foo\",\"id\":3,\"path\":\"/mnt/dozer/testing/test3\"},"+
				"{\"comment\":\"foo\",\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}",
			"{}",
			"{}",
		},
		"",
	))
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
	FailIf(t, DoTest(
		t,
		nfsListCmd,
		listNfs,
		map[string]interface{}{"json":true},
		[]string{"/mnt/dozer/testing/test3","dozer/testing/test4"},
		[]string{"[[[\"path\",\"in\",[\"/mnt/dozer/testing/test3\",\"/mnt/dozer/testing/test4\"]]]]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":3,\"path\":\"/mnt/dozer/testing/test3\"},{\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}],\"id\":2}"},
		"{\"shares\":{\"3\":{\"id\":3,\"path\":\"/mnt/dozer/testing/test3\"},\"4\":{\"id\":4,\"path\":\"/mnt/dozer/testing/test4\"}}}\n",
	))
}

package cmd

import (
	"testing"
)

func TestGenericList(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"no-headers":true,"parsable":true,"output":"id,clones"},
		[]string{},
		[]string{"[[],{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\"],\"retrieve_children\":true,\"user_properties\":false}}]"},
		[]string{"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\","+
			"\"properties\":{\"clones\":{\"rawvalue\":\"dozer/testing/test5\",\"value\":\"dozer/testing/test5\",\"parsed\":\"dozer/testing/test5\"}}}],\"id\":2}"},
		"dozer/testing/test4@readonly\tdozer/testing/test5\n",
	))
}

func TestGenericListTypes(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"types":"vol,snap","no-headers":true,"parsable":true,"output":"type,id,clones"},
		[]string{},
		[]string{ // expected
			"[[],{\"extra\":{\"flat\":false,\"properties\":[\"type\",\"id\",\"clones\"],\"retrieve_children\":true,\"user_properties\":false}}]",
			"[[],{\"extra\":{\"flat\":false,\"properties\":[\"type\",\"id\",\"clones\",\"createtxg\"],\"retrieve_children\":true,\"user_properties\":false}}]",
		},
		[]string{ // response
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test5\",\"name\":\"dozer/testing/test5\",\"type\":\"volume\"}],\"id\":2}",
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\",\"properties\":{\"clones\":"+
				"{\"rawvalue\":\"dozer/testing/test5\",\"value\":\"dozer/testing/test5\",\"parsed\":\"dozer/testing/test5\"}},\"type\":\"snapshot\"}],\"id\":2}",
		},
		"snapshot\tdozer/testing/test4@readonly\tdozer/testing/test5\n"+
		"volume\tdozer/testing/test5\t-\n",
	))
}

func TestGenericListParameters(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"recursive":true,"no-headers":true,"parsable":true,"output":"id,clones"},
		[]string{"dozer/testing"},
		[]string{ // expected
			"[[[\"name\",\"in\",[\"dozer/testing\"]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\"],\"retrieve_children\":true,\"user_properties\":false}}]",
			"[[[\"OR\",[[\"dataset\",\"=\",\"dozer/testing\"],[\"dataset\",\"^\",\"dozer/testing/\"]]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\",\"createtxg\"],\"retrieve_children\":true,\"user_properties\":false}}]",
		},
		[]string{ // response
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test\",\"name\":\"dozer/testing/test\"},"+
				"{\"id\":\"dozer/testing/test5\",\"name\":\"dozer/testing/test5\"}],\"id\":2}",
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\","+
				"\"properties\":{\"clones\":{\"rawvalue\":\"dozer/testing/test5\",\"value\":\"dozer/testing/test5\",\"parsed\":\"dozer/testing/test5\"}}}],\"id\":2}",
		},
		"dozer/testing/test\t-\n"+
		"dozer/testing/test4@readonly\tdozer/testing/test5\n"+
		"dozer/testing/test5\t-\n",
	))
}

func TestGenericListParametersRecursive(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"recursive":true,"no-headers":true,"parsable":true,"output":"id,clones"},
		[]string{"dozer/testing"},
		[]string{ // expected
			"[[[\"name\",\"in\",[\"dozer/testing\"]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\"],\"retrieve_children\":true,\"user_properties\":false}}]",
			"[[[\"OR\",[[\"dataset\",\"=\",\"dozer/testing\"],[\"dataset\",\"^\",\"dozer/testing/\"]]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\",\"createtxg\"],\"retrieve_children\":true,\"user_properties\":false}}]",
		},
		[]string{ // response
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test\",\"name\":\"dozer/testing/test\"},"+
				"{\"id\":\"dozer/testing/test5\",\"name\":\"dozer/testing/test5\"}],\"id\":2}",
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\","+
				"\"properties\":{\"clones\":{\"rawvalue\":\"dozer/testing/test5\",\"value\":\"dozer/testing/test5\",\"parsed\":\"dozer/testing/test5\"}}}],\"id\":2}",
		},
		"dozer/testing/test\t-\n"+
		"dozer/testing/test4@readonly\tdozer/testing/test5\n"+
		"dozer/testing/test5\t-\n",
	))
}

func TestGenericListTypesAndParameters(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"recursive":true,"no-headers":true,"parsable":true,"output":"id,clones"},
		[]string{"dozer/testing"},
		[]string{ // expected
			"[[[\"name\",\"in\",[\"dozer/testing\"]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\"],\"retrieve_children\":true,\"user_properties\":false}}]",
			"[[[\"OR\",[[\"dataset\",\"=\",\"dozer/testing\"],[\"dataset\",\"^\",\"dozer/testing/\"]]]],"+
				"{\"extra\":{\"flat\":false,\"properties\":[\"id\",\"clones\",\"type\",\"createtxg\"],\"retrieve_children\":true,\"user_properties\":false}}]",
		},
		[]string{ // response
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test\",\"name\":\"dozer/testing/test\"},"+
				"{\"id\":\"dozer/testing/test5\",\"name\":\"dozer/testing/test5\"}],\"id\":2}",
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test4@readonly\",\"name\":\"dozer/testing/test4@readonly\","+
				"\"properties\":{\"clones\":{\"rawvalue\":\"dozer/testing/test5\",\"value\":\"dozer/testing/test5\",\"parsed\":\"dozer/testing/test5\"}}}],\"id\":2}",
		},
		"dozer/testing/test\t-\n"+
		"dozer/testing/test4@readonly\tdozer/testing/test5\n"+
		"dozer/testing/test5\t-\n",
	))
}

func TestGenericListOrder(t *testing.T) {
	FailIf(t, DoTest(
		t,
		listCmd,
		doList,
		map[string]interface{}{"recursive":true,"no-headers":true},
		[]string{"dozer/testing"},
		[]string{
			"[[[\"name\",\"in\",[\"dozer/testing\"]]],{\"extra\":{\"flat\":false,"+
				"\"properties\":[],\"retrieve_children\":true,\"user_properties\":false}}]",
			"[[[\"OR\",[[\"dataset\",\"=\",\"dozer/testing\"],[\"dataset\",\"^\",\"dozer/testing/\"]]]],{\"extra\":{\"flat\":false,"+
				"\"properties\":[\"createtxg\"],\"retrieve_children\":true,\"user_properties\":false}}]",
		},
		[]string{
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test5\",\"name\":\"dozer/testing/test5\"},"+
				"{\"id\":\"dozer/testing/test4\",\"name\":\"dozer/testing/test4\"}],\"id\":2}",
			"{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":\"dozer/testing/test5@readonly\",\"name\":\"dozer/testing/test5@readonly\",\"createtxg\":1001},"+
				"{\"id\":\"dozer/testing/test4@snap1\",\"name\":\"dozer/testing/test4@snap1\",\"createtxg\":1003},"+
				"{\"id\":\"dozer/testing/test4@snap2\",\"name\":\"dozer/testing/test4@snap2\",\"createtxg\":1002}],\"id\":2}",
		},
		"dozer/testing/test4\n"+
		"dozer/testing/test4@snap2\n"+
		"dozer/testing/test4@snap1\n"+
		"dozer/testing/test5\n"+
		"dozer/testing/test5@readonly\n",
	))
}

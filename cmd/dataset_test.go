package cmd

import (
	"testing"
)

func TestDatasetCreateWithParentsTrue(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":true},
		[]string{"dozer/testing/test"},
		"[{\"create_ancestors\":true,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]",
	))
}

func TestDatasetCreateWithParentsFalse(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":false},
		[]string{"dozer/testing/test"},
		"[{\"create_ancestors\":false,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]",
	))
}

func TestDatasetCreateWithParentsEmpty(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":""},
		[]string{"dozer/testing/test"},
		"aux flag create_parents: type mismatch (existing: bool, type of given value: string)",
	))
}

func TestDatasetCreateWithComma(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{},
		[]string{"dozer/testing/test,comma"},
		"[{\"name\":\"dozer/testing/test,comma\",\"type\":\"FILESYSTEM\"}]",
	))
}

func TestDatasetCreateVolume(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"volume":1024},
		[]string{"dozer/testing/test2"},
		"[{\"name\":\"dozer/testing/test2\",\"type\":\"VOLUME\",\"volsize\":1024}]",
	))
}

func TestDatasetCreateWithComments(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{
			"option":"exec=on,atime=off,acltype=posix,aclmode=discard",
			"managedby":"incus.truenas",
			"comments":"Managed by Incus.TrueNAS",
		},
		[]string{"dozer/testing/test"},
		"[{\"aclmode\":\"DISCARD\",\"acltype\":\"POSIX\",\"atime\":\"OFF\","+
			"\"comments\":\"Managed by Incus.TrueNAS\",\"exec\":\"ON\","+
			"\"managedby\":\"incus.truenas\",\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]",
	))
}

func TestDatasetUpdate(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetUpdateCmd,
		createOrUpdateDataset,
		map[string]interface{}{
			"option":"exec=off,atime=off,acltype=posix,aclmode=discard",
			"managedby":"incus.truenas",
			"comments":"Managed by Incus.TrueNAS",
		},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\",{\"aclmode\":\"DISCARD\",\"acltype\":\"POSIX\","+
			"\"atime\":\"OFF\",\"comments\":\"Managed by Incus.TrueNAS\","+
			"\"exec\":\"OFF\",\"managedby\":\"incus.truenas\"}]",
	))
}

func TestDatasetDelete(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetDeleteCmd,
		deleteDataset,
		map[string]interface{}{},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\",{}]",
	))
}

func TestDatasetDeleteRecursive(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetDeleteCmd,
		deleteDataset,
		map[string]interface{}{"recursive":true},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\",{\"recursive\":true}]",
	))
}

func TestDatasetDeleteForce(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetDeleteCmd,
		deleteDataset,
		map[string]interface{}{"force":true},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\",{\"force\":true}]",
	))
}

func TestDatasetDeleteRecursiveForce(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetDeleteCmd,
		deleteDataset,
		map[string]interface{}{"recursive":true,"force":true},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\",{\"force\":true,\"recursive\":true}]",
	))
}

func TestDatasetList(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetListCmd,
		listDataset,
		map[string]interface{}{},
		[]string{"dozer/testing/test"},
		"[[[\"name\",\"in\",[\"dozer/testing/test\"]]],{\"extra\":{\"flat\":false,"+
			"\"properties\":[],\"retrieve_children\":false,\"user_properties\":false}}]",
	))
}

func TestDatasetListWithProperties(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetListCmd,
		listDataset,
		map[string]interface{}{"parseable":true,"output":"name,atime,relatime"},
		[]string{"dozer/testing/test"},
		"[[[\"name\",\"in\",[\"dozer/testing/test\"]]],{\"extra\":{\"flat\":false,"+
			"\"properties\":[\"name\",\"atime\",\"relatime\"],\"retrieve_children\":false,\"user_properties\":false}}]",
	))
}

func TestDatasetPromote(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetPromoteCmd,
		promoteDataset,
		map[string]interface{}{},
		[]string{"dozer/testing/test"},
		"[\"dozer/testing/test\"]",
	))
}

func TestDatasetRename(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		datasetRenameCmd,
		renameDataset,
		map[string]interface{}{},
		[]string{"dozer/testing/test", "dozer/testing/test3"},
		"[\"dozer/testing/test\",{\"new_name\":\"dozer/testing/test3\"}]",
	))
}

func TestDatasetRenameUpdateShares(t *testing.T) {
	
}

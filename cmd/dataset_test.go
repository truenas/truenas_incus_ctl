package cmd

import (
	"testing"
)

func TestDatasetCreateWithParentsTrue(t *testing.T) {
	DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":true},
		[]string{"dozer/testing/test"},
		"[{\"create_ancestors\":true,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]",
	)
}

func TestDatasetCreateWithParentsFalse(t *testing.T) {
	DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":false},
		[]string{"dozer/testing/test"},
		"[{\"create_ancestors\":false,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]",
	)
}

func TestDatasetCreateWithParentsEmpty(t *testing.T) {
	DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"create-parents":""},
		[]string{"dozer/testing/test"},
		"aux flag create_parents: type mismatch (existing: bool, type of given value: string)",
	)
}

func TestDatasetCreateVolume(t *testing.T) {
	DoSimpleTest(
		t,
		datasetCreateCmd,
		createOrUpdateDataset,
		map[string]interface{}{"volume":1024},
		[]string{"dozer/testing/test2"},
		"[{\"name\":\"dozer/testing/test2\",\"type\":\"VOLUME\",\"volsize\":1024}]",
	)
}

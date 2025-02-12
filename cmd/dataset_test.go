package cmd

import (
	"testing"
)

func TestDatasetCreate(t *testing.T) {
	expect := "[{\"create_ancestors\":true,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]"
	response := "{}"
	api := setupTest(t, expect, response)
	SetAuxCobraFlag(datasetCreateCmd, "create-parents", true)
	failIf(t, createOrUpdateDataset(datasetCreateCmd, api, []string{"dozer/testing/test"}))
	ResetAuxCobraFlags(datasetCreateCmd)

	api.expect = "[{\"name\":\"dozer/testing/test2\",\"type\":\"VOLUME\",\"volsize\":1024}]"
	SetAuxCobraFlag(datasetCreateCmd, "volume", 1024)
	failIf(t, createOrUpdateDataset(datasetCreateCmd, api, []string{"dozer/testing/test2"}))
	ResetAuxCobraFlags(datasetCreateCmd)
}

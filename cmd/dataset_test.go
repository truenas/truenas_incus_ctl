package cmd

import (
	"testing"
)

func TestDatasetCreate(t *testing.T) {
	expect := "[{\"create_ancestors\":true,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]"
	response := "{}"
	api := setupTest(t, expect, response)
	failIf(t, SetCobraFlag(datasetCreateCmd, "create-parents", "true"))
	failIf(t, createOrUpdateDataset(datasetCreateCmd, api, []string{"dozer/testing/test"}))
}

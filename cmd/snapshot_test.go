package cmd

import (
	"testing"
)

func TestSnapshotListWithProperties(t *testing.T) {
	DoSimpleTest(
		t,
		snapshotListCmd,
		listSnapshot,
		map[string]interface{}{"no-headers":true,"parseable":true,"output":"clones"},
		[]string{"dozer/testing/test@readonly"},
		"[[[\"name\",\"in\",[\"dozer/testing/test@readonly\"]]],{\"extra\":{\"flat\":false,"+
			"\"properties\":[\"clones\"],\"retrieve_children\":false,\"user_properties\":false}}]",
	)
}

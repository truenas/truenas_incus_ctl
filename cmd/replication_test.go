package cmd

import (
	"testing"
)

func TestReplicationStart(t *testing.T) {
	FailIf(t, DoSimpleTest(
		t,
		replStartCmd,
		startReplication,
		map[string]interface{}{"options":"a=true,bar=4,custom=something","direction":"push","retention_policy":"none"},
		[]string{"dozer/testing/test1","dozer/testing/test2"},
		"[{\"a\":true,\"bar\":4,\"custom\":\"something\",\"direction\":\"PUSH\",\"recursive\":false,\"retention_policy\":\"NONE\","+
			"\"source_datasets\":[\"dozer/testing/test1\"],\"target_dataset\":\"dozer/testing/test2\",\"transport\":\"LOCAL\"}]",
	))
}

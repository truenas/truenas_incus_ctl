package cmd

import (
	"encoding/json"
	"fmt"
	"testing"
)

func failIf(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

type DummySession struct {
	test *testing.T
	expect string
	response string
}

func (s *DummySession) Login() error { return nil }
func (s *DummySession) Close() error { return nil }

func (s *DummySession) CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(params)
	failIf(s.test, err)
	if string(data) != s.expect {
		s.test.Error(fmt.Errorf("\"%s\" != \"%s\"", string(data), s.expect))
	}
	return []byte(s.response), nil
}

func setupTest(t *testing.T, expect, response string) *DummySession {
	api := &DummySession{}
	//api.Login()
	api.test = t
	api.expect = expect
	api.response = response
	return api
}

func TestDatasetCreate(t *testing.T) {
	expect := "[{\"create_ancestors\":true,\"name\":\"dozer/testing/test\",\"type\":\"FILESYSTEM\"}]"
	response := "{}"
	api := setupTest(t, expect, response)
	failIf(t, SetCobraFlag(datasetCreateCmd, "create-parents", "true"))
	createOrUpdateDataset(datasetCreateCmd, api, []string{"dozer/testing/test"})
}

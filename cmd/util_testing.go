package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"truenas/truenas-admin/core"

	"github.com/spf13/cobra"
)

type DummySession struct {
	test *testing.T
	expect string
	response string
}

func (s *DummySession) Login() error { return nil }
func (s *DummySession) Close() error { return nil }

func (s *DummySession) CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(params)
	FailIf(s.test, err)
	if string(data) != s.expect {
		s.test.Error(fmt.Errorf("\"%s\" != \"%s\"", string(data), s.expect))
	}
	return []byte(s.response), errors.New(s.expect)
}

func FailIf(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func SetupTest(t *testing.T, expect, response string) *DummySession {
	api := &DummySession{}
	//api.Login()
	api.test = t
	api.expect = expect
	api.response = response
	return api
}

func DoSimpleTest(t *testing.T, cmd *cobra.Command, commandFunc func(*cobra.Command,core.Session,[]string)error, props map[string]interface{}, args []string, expect string) {
	response := "{}"
	for key, value := range props {
		SetAuxCobraFlag(cmd, key, value)
	}
	err := commandFunc(cmd, SetupTest(t, expect, response), args)
	ResetAuxCobraFlags(cmd)
	if err != nil {
		errMsg := err.Error()
		if errMsg != expect {
			fmt.Println("\"" + errMsg + "\" != \"" + expect + "\"")
			t.Error(err)
		}
	}
}

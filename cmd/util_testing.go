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
	expects []string
	responses []string
	callIdx int
}

func (s *DummySession) Login() error { return nil }
func (s *DummySession) Close() error { return nil }

func (s *DummySession) CallRaw(method string, timeoutStr string, params interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(params)
	FailIf(s.test, err)
	expect := s.expects[s.callIdx]
	if string(data) != expect {
		return nil, fmt.Errorf("\"%s\" != \"%s\"", string(data), expect)
	}
	response := []byte(s.responses[s.callIdx])
	s.callIdx++
	return response, errors.New(expect)
}

func FailIf(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func SetupSimpleTest(t *testing.T, expect, response string) *DummySession {
	api := &DummySession{}
	//api.Login()
	api.test = t
	api.expects = []string{expect}
	api.responses = []string{response}
	return api
}

func DoSimpleTest(
	t *testing.T,
	cmd *cobra.Command,
	commandFunc func(*cobra.Command,core.Session,[]string)error,
	props map[string]interface{},
	args []string,
	expect string,
) error {
	response := "{}"
	for key, value := range props {
		SetAuxCobraFlag(cmd, key, value)
	}
	defer ResetAuxCobraFlags(cmd)
	err := commandFunc(cmd, SetupSimpleTest(t, expect, response), args)
	if err != nil {
		errMsg := err.Error()
		if errMsg != expect {
			fmt.Println("\"" + errMsg + "\" != \"" + expect + "\"")
			return err
		}
	}
	return nil
}

func SetupMultiTest(t *testing.T, expectList, responseList []string) *DummySession {
	if len(expectList) != len(responseList) || len(expectList) < 1 {
		return nil
	}
	api := &DummySession{}
	//api.Login()
	api.test = t
	api.expects = expectList
	api.responses = responseList
	return api
}

func DoTest(
	t *testing.T,
	cmd *cobra.Command,
	commandFunc func(*cobra.Command,core.Session,[]string)error,
	props map[string]interface{},
	args []string,
	expectList []string,
	responseList []string,
) error {
	for key, value := range props {
		SetAuxCobraFlag(cmd, key, value)
	}
	defer ResetAuxCobraFlags(cmd)
	api := SetupMultiTest(t, expectList, responseList)
	if api == nil {
		return errors.New("Failed to set up test correctly. Check expectList and responseList.")
	}
	err := commandFunc(cmd, api, args)
	if err != nil {
		errMsg := err.Error()
		expectMsg := expectList[api.callIdx]
		if expectMsg != errMsg {
			fmt.Println("\"" + errMsg + "\" != \"" + expectMsg + "\"")
			return err
		}
	}
	return nil
}

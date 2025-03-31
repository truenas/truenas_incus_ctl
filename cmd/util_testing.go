package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

type UnitTestSession struct {
	test *testing.T
	expects []string
	responses []string
	tableExpected string
	callIdx int
	shouldIncCallIdx bool
}

func (s *UnitTestSession) Login() error { return nil }
func (s *UnitTestSession) IsLoggedIn() bool { return true }
func (s *UnitTestSession) WaitForJob(jobId int64) (json.RawMessage, error) { return nil, nil }
func (s *UnitTestSession) Close(internalError error) error { return nil }

func (s *UnitTestSession) CallRaw(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error) {
	if s.shouldIncCallIdx {
		s.callIdx++
	}
	data, err := json.Marshal(params)
	FailIf(s.test, err)
	expect := s.expects[s.callIdx]
	if string(data) != expect {
		msg := fmt.Errorf("\"%s\" != \"%s\"", string(data), expect)
		return nil, msg
	}
	response := []byte(s.responses[s.callIdx])
	s.shouldIncCallIdx = true
	return response, nil
}

func (s *UnitTestSession) CallAsyncRaw(method string, params interface{}, awaitThisJob bool) (int64, error) {
	_, err := s.CallRaw(method, 10, params)
	return -1, err
}

func PrintTable(api core.Session, str string) {
	if unit, isUnitTest := api.(*UnitTestSession); isUnitTest {
		if unit.tableExpected != str {
			unit.test.Error(errors.New("table:\n" + str + "did not match expected:\n" + unit.tableExpected))
		}
	} else {
		os.Stdout.WriteString(str)
	}
}

func SetupSimpleTest(t *testing.T, expect, response string) *UnitTestSession {
	api := &UnitTestSession{}
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

func SetupMultiTest(t *testing.T, expectList, responseList []string, tableExpected string) *UnitTestSession {
	if len(expectList) != len(responseList) || len(expectList) < 1 {
		return nil
	}
	api := &UnitTestSession{}
	//api.Login()
	api.test = t
	api.expects = expectList
	api.responses = responseList
	api.tableExpected = tableExpected
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
	tableExpected string,
) error {
	for key, value := range props {
		SetAuxCobraFlag(cmd, key, value)
	}
	defer ResetAuxCobraFlags(cmd)
	api := SetupMultiTest(t, expectList, responseList, tableExpected)
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

func FailIf(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func FailUnless(t *testing.T, err error) {
	if err == nil {
		t.Error(errors.New("test expected an error, none was received"))
	}
}

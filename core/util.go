package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"slices"
)

func IdentifyObject(obj string) (string, string) {
	if obj == "" {
		return "", ""
	} else if _, errNotNumber := strconv.Atoi(obj); errNotNumber == nil {
		return "id", obj
	} else if obj[0] == '/' {
		return "share", obj
	} else if obj[0] == '@' {
		return "snapshot_only", obj[1:]
	} else if pos := strings.Index(obj, "@"); pos >= 1 {
		if pos == len(obj)-1 {
			return "error", obj
		}
		return "snapshot", obj
	} else if pos := strings.LastIndex(obj, "/"); pos >= 1 {
		if pos == len(obj)-1 {
			return IdentifyObject(obj[0:len(obj)-1])
		}
		return "dataset", obj
	}
	return "pool", obj
}

func StringRepeated(str string, count int) []string {
	if count <= 0 {
		return nil
	}
	arr := make([]string, count)
	for i := 0; i < count; i++ {
		arr[i] = str
	}
	return arr
}

func AppendIfMissing[T comparable](arr []T, value T) []T {
	for _, elem := range arr {
		if elem == value {
			return arr
		}
	}
	return append(arr, value)
}

func MakeErrorFromList(errorList []error) error {
	if len(errorList) == 0 {
		return nil
	}

	var combinedErrMsg strings.Builder
	for _, e := range errorList {
		combinedErrMsg.WriteString("\n")
		combinedErrMsg.WriteString(e.Error())
	}

	return errors.New(combinedErrMsg.String())
}

func GetKeysSorted[T any](dict map[string]T) []string {
	var keys []string
	size := len(dict)
	if size > 0 {
		keys = make([]string, 0, size)
		for k, _ := range dict {
			keys = append(keys, k)
		}
		slices.Sort(keys)
	}
	return keys
}

func ToAnyArray[T any](arr []T) []interface{} {
	if arr == nil {
		return nil
	}
	n := len(arr)
	out := make([]interface{}, n, n)
	for i := 0; i < n; i++ {
		out[i] = arr[i]
	}
	return out
}

// Slow, but easy
func DeepCopy(input interface{}) interface{} {
	if input == nil {
		return nil
	}
	data, _ := json.Marshal(input)
	var output interface{}
	json.Unmarshal(data, &output)
	return output
}

func GetResultsAndErrorsFromApiResponseRaw(response json.RawMessage) ([]interface{}, []interface{}) {
	var unmarshalled map[string]interface{}
	if err := json.Unmarshal(response, &unmarshalled); err != nil {
		return nil, nil
	}
	return GetResultsAndErrorsFromApiResponse(unmarshalled)
}

func GetResultsAndErrorsFromApiResponse(response map[string]interface{}) ([]interface{}, []interface{}) {
	if response == nil {
		return nil, nil
	}

	var errorList []interface{}
	if errorObj, exists := response["error"]; exists {
		if errValue, ok := errorObj.(map[string]interface{}); ok {
			if len(errValue) > 0 {
				errorList = []interface{} {errValue}
			}
		} else if errValue, ok := errorObj.([]interface{}); ok {
			if len(errValue) > 0 {
				errorList = errValue
			}
		}
	}

	var resultList []interface{}
	if resultsObj, exists := response["result"]; exists {
		if resultsMap, ok := resultsObj.(map[string]interface{}); ok {
			resultList = []interface{} {resultsMap}
		} else if resultsArray, ok := resultsObj.([]interface{}); ok && len(resultsArray) > 0 {
			resultList = resultsArray
		}
	}
	if len(resultList) == 0 {
		return nil, errorList
	}

	isCoreBulk := false
	if firstResult, ok := resultList[0].(map[string]interface{}); ok {
		outerMethod, _ := firstResult["method"].(string)
		isCoreBulk = outerMethod == "core.bulk"
	}
	if !isCoreBulk {
		return resultList, errorList
	}

	outResults := make([]interface{}, 0)
	outErrors := make([]interface{}, 0)
	for _, r := range resultList {
		if obj, ok := r.(map[string]interface{}); ok {
			subResults, subErrors := GetResultsAndErrorsFromApiResponse(obj)
			if len(subResults) > 0 {
				outResults = append(outResults, subResults...)
			}
			if len(subErrors) > 0 {
				outErrors = append(outErrors, subErrors...)
			}
		}
	}

	if len(outResults) == 0 {
		outResults = nil
	}
	if len(outErrors) == 0 {
		outErrors = nil
	}
	return outResults, outErrors
}

func ExtractJsonArrayOfMaps(obj map[string]interface{}, key string) ([]map[string]interface{}, string) {
	if value, ok := obj[key]; ok {
		if array, ok := value.([]interface{}); ok {
			if len(array) == 0 {
				return nil, ""
			}
			list := make([]map[string]interface{}, 0)
			for i := 0; i < len(array); i++ {
				if elem, ok := array[i].(map[string]interface{}); ok {
					list = append(list, elem)
				} else {
					return nil, "contained a non-object entry"
				}
			}
			return list, ""
		}
		return nil, "was not an array"
	}
	return nil, "did not contain a list"
}

func IsValueTrue(dict map[string]string, key string) bool {
	if valueStr, exists := dict[key]; exists {
		return valueStr == "true"
	}
	return false
}

func GetIntegerFromJsonObjectOr(data map[string]interface{}, key string, ifNotFound int64) int64 {
	if value, exists := data[key]; exists {
		if valueF, ok := value.(float64); ok {
			return int64(valueF)
		} else if valueStr, ok := value.(string); ok {
			if valueI, errNotNumber := strconv.ParseInt(valueStr, 0, 64); errNotNumber == nil {
				return valueI
			}
			return ifNotFound
		} else if valueI, ok := value.(int64); ok {
			return valueI
		}
	}
	return ifNotFound
}

func ExtractApiError(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}

	var response interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return "\n" + err.Error()
	}

	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return ""
	}

	return ExtractApiErrorJson(responseMap)
}

func ExtractApiErrorJson(responseMap map[string]interface{}) string {
	errorValue, exists := responseMap["error"]
	if !exists {
		return ""
	}

	var codeStr string
	var messageStr string
	var reasonStr string

	if errorObj, ok := errorValue.(map[string]interface{}); ok {
		if codeValue, exists := errorObj["code"]; exists {
			codeStr = fmt.Sprint(codeValue)
		}
		if messageValue, exists := errorObj["message"]; exists {
			messageStr = fmt.Sprint(messageValue)
		}
		if dataValue, exists := errorObj["data"]; exists {
			if dataObj, ok := dataValue.(map[string]interface{}); ok {
				if reasonValue, exists := dataObj["reason"]; exists {
					reasonStr = fmt.Sprint(reasonValue)
				}
			}
		}
	}

	var builder strings.Builder
	builder.WriteString("\nError ")
	if codeStr != "" {
		builder.WriteString(codeStr)
		builder.WriteString("\n")
	}
	if messageStr != "" {
		builder.WriteString(messageStr)
		builder.WriteString("\n")
	}
	if reasonStr != "" {
		builder.WriteString(reasonStr)
	}

	return builder.String()
}

func GetJobNumber(data json.RawMessage) (int, error) {
	var responseJson interface{}
	if err := json.Unmarshal(data, &responseJson); err != nil {
		return -1, err
	}
	return GetJobNumberFromObject(responseJson)
}

func GetJobNumberFromObject(responseJson interface{}) (int, error) {
	if responseJson == nil {
		return -1, errors.New("response was nil")
	}
	if obj, ok := responseJson.(map[string]interface{}); ok {
		if resultObj, exists := obj["result"]; exists {
			if resultNumberFloat, ok := resultObj.(float64); ok {
				return int(resultNumberFloat), nil
			} else if resultNumber64, ok := resultObj.(int64); ok {
				return int(resultNumber64), nil
			} else if resultNumber, ok := resultObj.(int); ok {
				return resultNumber, nil
			} else {
				return -1, errors.New("result in response was not a job number")
			}
		} else {
			return -1, errors.New("result was not found in response")
		}
	} else {
		return -1, errors.New("response was not a json object")
	}
}

func RunCommandRaw(prog string, args ...string) (string, string, error) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.Command(prog, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func RunCommand(prog string, args ...string) (string, error) {
	out, warn, err := RunCommandRaw(prog, args...)
	var errMsg strings.Builder
	isError := false
	if warn != "" {
		errMsg.WriteString(warn)
		if warn[len(warn)-1] != '\n' {
			errMsg.WriteString("\n")
		}
		isError = true
	}
	if err != nil {
		errMsg.WriteString(err.Error())
		isError = true
	}
	if isError {
		return "", errors.New(errMsg.String())
	}
	return out, nil
}

func FlushString(str string) {
	os.Stdout.WriteString(str)
	os.Stdout.Sync()
}

type ReadAllWriteAll interface {
	ReadAll() ([]byte, error)
	WriteAll([]byte) error
}

type FileRawa struct {
	FileName string
}

type MemoryRawa struct {
	Current []byte
}

func (rw *FileRawa) ReadAll() ([]byte, error) {
	return os.ReadFile(rw.FileName)
}
func (rw *FileRawa) WriteAll(content []byte) error {
	return os.WriteFile(rw.FileName, content, 0666)
}

func (rw *MemoryRawa) ReadAll() ([]byte, error) {
	var buf []byte
	size := len(rw.Current)
	if size > 0 {
		buf = make([]byte, size)
		copy(buf, rw.Current)
	}
	return buf, nil
}
func (rw *MemoryRawa) WriteAll(content []byte) error {
	rw.Current = content
	return nil
}

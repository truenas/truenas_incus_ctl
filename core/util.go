package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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

func IsStringTrue(dict map[string]string, key string) bool {
	if valueStr, exists := dict[key]; exists {
		return valueStr == "true"
	}
	return false
}

func IsValueTrue(dict map[string]interface{}, key string) bool {
	if valueObj, exists := dict[key]; exists {
		if valueBool, ok := valueObj.(bool); ok {
			return valueBool
		} else if valueStr, ok := valueObj.(string); ok {
			return strings.ToLower(valueStr) == "true"
		}
	}
	return false
}

func IpPortToJsonString(str string, defaultHostname string, defaultPort int) string {
	if str == "" || str[0] == '[' || str[0] == '{' {
		return str
	}

	pos := strings.Index(str, ":")
	hostname := defaultHostname
	port := defaultPort
	if pos < 0 {
		hostname = str
	} else {
		if pos > 0 {
			hostname = str[0:pos]
		}
		if pos < len(str) - 1 {
			if n, errNotNumber := strconv.Atoi(str[pos+1:]); errNotNumber == nil {
				port = n
			}
		}
	}

	resolved := ResolvedIpv4OrVerbatim(hostname)
	return "[{\"ip\":\"" + resolved + "\",\"port\":" + fmt.Sprint(port) + "}]"
}

func GetIdFromObject(obj interface{}) interface{} {
	if obj == nil {
		return nil
	} else if objF, ok := obj.(float64); ok {
		return objF
	} else if objI64, ok := obj.(int64); ok {
		return objI64
	} else if objI, ok := obj.(int); ok {
		return objI
	} else if objMap, ok := obj.(map[string]interface{}); ok {
		return objMap["id"]
	} else if objArr, ok := obj.([]interface{}); ok {
		return GetIdFromObject(objArr[0])
	}
	return nil
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
	return ExtractApiErrorJsonGivenError(errorValue)
}

func ExtractApiErrorJsonGivenError(errorValue interface{}) string {
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

func GetJobNumber(data json.RawMessage) (int64, error) {
	var responseJson interface{}
	if err := json.Unmarshal(data, &responseJson); err != nil {
		return -1, err
	}
	return GetJobNumberFromObject(responseJson)
}

func GetJobNumberFromObject(responseJson interface{}) (int64, error) {
	if responseJson == nil {
		return -1, errors.New("response was nil")
	}
	if obj, ok := responseJson.(map[string]interface{}); ok {
		if resultObj, exists := obj["result"]; exists {
			if resultNumberFloat, ok := resultObj.(float64); ok {
				return int64(resultNumberFloat), nil
			} else if resultNumber64, ok := resultObj.(int64); ok {
				return resultNumber64, nil
			} else if resultNumber, ok := resultObj.(int); ok {
				return int64(resultNumber), nil
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

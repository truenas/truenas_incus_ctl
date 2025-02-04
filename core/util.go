package core

import (
	"os"
	"strings"
	"slices"
)

func EncloseAndEscape(original string, ends string) string {
	var builder strings.Builder
	WriteEncloseAndEscape(&builder, original, ends)
	return builder.String()
}

func WriteEncloseAndEscape(builder *strings.Builder, original string, ends string) {
	builder.WriteString(ends)
	off := 0
	for off < len(original) {
		idx := strings.Index(original[off:], ends)
		if idx < 0 {
			builder.WriteString(original[off:])
			break
		}
		if idx != 0 {
			builder.WriteString(original[off:off+idx])
		}
		builder.WriteString("\\")
		builder.WriteString(ends)
		off += idx + 1
	}
	builder.WriteString(ends)
}

func ExposeString(original string, ends string) string {
	size := len(original)
	endSize := len(ends)
	if size <= 2*endSize || original[0:endSize] != ends && original[size-endSize:] != ends {
		return original
	}
	return strings.ReplaceAll(original[endSize:size-endSize], "\\" + ends, ends)
}

func WriteJsonStringArray(builder *strings.Builder, valueList []string) {
	for i, elem := range valueList {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("\"")
		builder.WriteString(elem)
		builder.WriteString("\"")
	}
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

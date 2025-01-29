package core

import (
	"errors"
	"strings"
	"slices"
)

func EncloseWith(original string, ends string) (string, error) {
	if strings.Index(original, ends) >= 0 {
		return "", errors.New("string already contains '" + ends + "'")
	}
	return ends + original + ends, nil
}

func WriteEncloseWith(builder *strings.Builder, original string, ends string) error {
	str, err := EncloseWith(original, ends)
	if err != nil {
		return err
	}
	builder.WriteString(str)
	return nil
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

package cmd

import (
	"errors"
	"fmt"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FlagMap struct {
	flagKeys  []string
	usedFlags map[string]string
	allFlags  map[string]string
	allTypes  map[string]string
}

func GetCobraFlags(cmd *cobra.Command, cmdEnums map[string][]string) (FlagMap, error) {
	fm := FlagMap{}
	fm.usedFlags = make(map[string]string)
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		fm.usedFlags[flag.Name] = flag.Value.String()
	})

	fm.flagKeys = make([]string, 0)
	fm.allFlags = make(map[string]string)
	fm.allTypes = make(map[string]string)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		fm.flagKeys = append(fm.flagKeys, flag.Name)
		fm.allFlags[flag.Name] = flag.Value.String()
		fm.allTypes[flag.Name] = flag.Value.Type()
	})

	RemoveGlobalFlags(fm.usedFlags)
	RemoveGlobalFlags(fm.allFlags)
	RemoveGlobalFlags(fm.allTypes)

	if err := ValidateFlagEnums(&fm.allFlags, cmdEnums); err != nil {
		return FlagMap{}, err
	}
	return fm, nil
}

func ValidateFlagEnums(flags *map[string]string, cmdEnums map[string][]string) error {
	var builder strings.Builder
	for key, value := range *flags {
		if enumList, exists := cmdEnums[key]; exists {
			valueUpper := strings.ToUpper(value)
			found := false
			for _, elem := range enumList {
				if strings.ToUpper(elem) == valueUpper {
					found = true
					(*flags)[key] = valueUpper
					break
				}
			}
			if !found {
				builder.WriteString("Error: flag \"")
				builder.WriteString(key)
				builder.WriteString("\" - value \"")
				builder.WriteString(value)
				builder.WriteString("\" was not in valid set ")
				builder.WriteString(fmt.Sprint(enumList))
				builder.WriteString("\n")
			}
		}
	}
	str := builder.String()
	if str != "" {
		return errors.New(str)
	}
	return nil
}

func ValidateEnumArray(content string, enumList []string) ([]string, error) {
	var output []string
	if content == "" {
		return output, nil
	}

	var builder strings.Builder
	input := strings.Split(content, ",")

	for _, value := range input {
		valueUpper := strings.ToUpper(value)
		found := false
		for _, elem := range enumList {
			if strings.ToUpper(elem) == valueUpper {
				if output == nil {
					output = make([]string, 0)
				}
				output = append(output, valueUpper)
				found = true
				break
			}
		}
		if !found {
			builder.WriteString("Error: value \"")
			builder.WriteString(value)
			builder.WriteString("\" was not valid\n")
		}
	}
	if builder.Len() > 0 {
		builder.WriteString("Acceptable values: (")
		core.WriteJsonStringArray(&builder, enumList)
		builder.WriteString(")")
		return output, errors.New(builder.String())
	}

	return output, nil
}

func AddFlagsEnum(enumMap *map[string][]string, flagName string, newEnum []string) string {
	(*enumMap)[flagName] = newEnum
	var builder strings.Builder
	builder.WriteString("(")
	core.WriteJsonStringArray(&builder, newEnum)
	builder.WriteString(")")
	return builder.String()
}

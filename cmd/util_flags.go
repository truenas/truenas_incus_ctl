package cmd

import (
	"errors"
	"fmt"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FlagMap struct {
	flagKeys  []string
	usedFlags map[string]string
	allFlags  map[string]string
	allTypes  map[string]string
}

var auxiliaryFlags map[*cobra.Command]map[string]interface{}

func SetAuxCobraFlag(cmd *cobra.Command, rawKey string, parsedValue interface{}) {
	if auxiliaryFlags == nil {
		auxiliaryFlags = make(map[*cobra.Command]map[string]interface{})
	}
	if flags, exists := auxiliaryFlags[cmd]; !exists || flags == nil {
		auxiliaryFlags[cmd] = make(map[string]interface{})
	}
	key := strings.ReplaceAll(rawKey, "-", "_")
	auxiliaryFlags[cmd][key] = parsedValue
}

func ResetAuxCobraFlags(cmd *cobra.Command) {
	if auxiliaryFlags != nil {
		auxiliaryFlags[cmd] = nil
	}
}

func GetCobraFlags(cmd *cobra.Command, keepGlobals bool, cmdEnums map[string][]string) (FlagMap, error) {
	fm := FlagMap{}
	fm.usedFlags = make(map[string]string)
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		key := strings.ReplaceAll(flag.Name, "-", "_")
		fm.usedFlags[key] = flag.Value.String()
	})

	fm.flagKeys = make([]string, 0)
	fm.allFlags = make(map[string]string)
	fm.allTypes = make(map[string]string)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		key := strings.ReplaceAll(flag.Name, "-", "_")
		fm.flagKeys = append(fm.flagKeys, key)
		fm.allFlags[key] = flag.Value.String()
		fm.allTypes[key] = flag.Value.Type()
	})

	if aux, exists := auxiliaryFlags[cmd]; exists {
		for key, value := range aux {
			knownType, exists := fm.allTypes[key]
			if !exists {
				return FlagMap{}, fmt.Errorf("aux flag %s was not found in command \"%s\"", key, cmd.Use)
			}
			var typeStr string
			if _, ok := value.(string); ok {
				typeStr = "string"
			} else if _, ok := value.(bool); ok {
				typeStr = "bool"
			} else if _, ok := value.(int); ok {
				typeStr = "int"
			} else if _, ok := value.(int64); ok {
				typeStr = "int64"
			} else {
				typeStr = "any"
			}
			if knownType[:3] != typeStr[:3] {
				return FlagMap{}, fmt.Errorf("aux flag %s: type mismatch (existing: %s, type of given value: %s)", key, knownType, typeStr)
			}
			valueStr := fmt.Sprint(value)
			fm.allFlags[key] = valueStr
			fm.usedFlags[key] = valueStr
		}
	}

	if !keepGlobals {
		RemoveGlobalFlags(fm.usedFlags)
		RemoveGlobalFlags(fm.allFlags)
		RemoveGlobalFlags(fm.allTypes)
	}

	if err := ValidateFlagEnums(&fm.usedFlags, cmdEnums); err != nil {
		return FlagMap{}, err
	}
	if err := ValidateFlagEnums(&fm.allFlags, cmdEnums); err != nil {
		return FlagMap{}, err
	}
	return fm, nil
}

func InsertNonCobraFlag(fm FlagMap, flagType, flagName, flagValue string) {
	key := strings.ReplaceAll(flagName, "-", "_")
	fm.flagKeys = append(fm.flagKeys, key)
	fm.usedFlags[key] = flagValue
	fm.allFlags[key] = flagValue
	fm.allTypes[key] = flagType
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
				builder.WriteString("\": value \"")
				builder.WriteString(value)
				builder.WriteString("\" was not in the valid set (")
				builder.WriteString(strings.Join(enumList, ", "))
				builder.WriteString(")\n")
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
		builder.WriteString(strings.Join(enumList, ", "))
		builder.WriteString(")")
		return output, errors.New(builder.String())
	}

	return output, nil
}

func AddFlagsEnum(enumMap *map[string][]string, flagName string, newEnum []string) string {
	if *enumMap == nil {
		*enumMap = make(map[string][]string)
	}

	key := strings.ReplaceAll(flagName, "-", "_")
	(*enumMap)[key] = newEnum

	var builder strings.Builder
	builder.WriteString("(")
	builder.WriteString(strings.Join(newEnum, ", "))
	builder.WriteString(")")
	return builder.String()
}

func WrapCommandFunc(cmdFunc func(*cobra.Command,core.Session,[]string)error) func(*cobra.Command,[]string)error {
	return func(cmd *cobra.Command, args []string) error {
		api := InitializeApiClient()
		if api == nil {
			return nil
		}
		err := cmdFunc(cmd, api, args)
		return api.Close(err)
	}
}

func WrapCommandFuncWithoutApi(cmdFunc func(*cobra.Command,core.Session,[]string)error) func(*cobra.Command,[]string)error {
	return func(cmd *cobra.Command, args []string) error {
		return cmdFunc(cmd, nil, args)
	}
}

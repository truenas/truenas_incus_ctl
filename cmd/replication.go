package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:     "replication",
	Short:   "Replicate a dataset from one pool to another, locally or across any network",
	Aliases: []string{"backup", "repl"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var replStartCmd = &cobra.Command{
	Use:   "start <sources>... <destination> <-n|-N|-R> <name filter>",
	Short: "Start replicating a dataset from one pool to another, locally or across any network",
	Long:  "Start replicating a dataset from one pool to another, locally or across any network.\n"+
		"A replication specifier can either be \"<host>:<dataset>\" or just \"<dataset>\" if local.\n"+
		"Currently, only local to local replication is supported.",
	Args:  cobra.MinimumNArgs(2),
}

var g_replStartEnums map[string][]string

func init() {
	replStartCmd.RunE = WrapCommandFunc(startReplication)

	replStartCmd.Flags().StringP("exclude", "e", "", "")
	replStartCmd.Flags().BoolP("recursive", "r", false, "")
	replStartCmd.Flags().StringP("options", "o", "", "")
	replStartCmd.Flags().StringP("direction", "d", "push", ""+
		AddFlagsEnum(&g_replStartEnums, "direction", []string{"push", "pull"}))
	replStartCmd.Flags().StringP("retention-policy", "p", "none", ""+
		AddFlagsEnum(&g_replStartEnums, "retention-policy", []string{"source", "custom", "none"}))
	replStartCmd.Flags().StringP("naming-schema-main", "n", "", "")
	replStartCmd.Flags().StringP("naming-schema-aux", "N", "", "")
	replStartCmd.Flags().StringP("name-regex", "R", "", "")

	// TODO: implement non-local replication
	/*
		replStartCmd.Flags().String("transport", "", ""+
			AddFlagsEnum(&g_replStartEnums, "transport", []string{"ssh","ssh+netcat","local"}))
	*/

	replStartCmd.Flags().Int("ssh-credentials", 0, "")
	replStartCmd.Flags().String("netcat-active-side", "local", ""+
		 AddFlagsEnum(&g_replStartEnums, "netcat-active-side", []string{"local","remote"}))
	replStartCmd.Flags().String("netcat-active-side-listen-address", "", "")
	replStartCmd.Flags().Int("netcat-active-side-port-min", 0, "")
	replStartCmd.Flags().Int("netcat-active-side-port-max", 0, "")
	replStartCmd.Flags().String("netcat-passive-side-connect-address", "", "")
	replStartCmd.Flags().Bool("sudo", false, "")
	replStartCmd.Flags().Bool("aux-properties", true, "") // aux-properties -> properties
	replStartCmd.Flags().String("properties-exclude", "", "") // array of strings
	replStartCmd.Flags().String("properties-override", "", "") // array of key=value, ala --options
	replStartCmd.Flags().Bool("replicate", false, "")
	replStartCmd.Flags().Bool("encryption", false, "")
	replStartCmd.Flags().Bool("encryption-inherit", false, "")
	replStartCmd.Flags().String("encryption-key", "", "")
	replStartCmd.Flags().String("encryption-key-format", "passphrase", ""+
		AddFlagsEnum(&g_replStartEnums, "encryption-key-format", []string{"hex","passphrase"}))
	replStartCmd.Flags().String("encryption-key-location", "", "")
	replStartCmd.Flags().String("periodic-snapshot-tasks", "", "") // int array
	replStartCmd.Flags().String("restrict-schedule", "", "") // array of key=value, ala --options
	replStartCmd.Flags().Bool("allow-from-scratch", false, "")
	replStartCmd.Flags().String("readonly-policy", "set", ""+
		AddFlagsEnum(&g_replStartEnums, "readonly-policy", []string{"set","require","ignore"})) // "readonly-policy" -> "readonly"
	replStartCmd.Flags().Bool("hold-pending-snapshots", false, "")
	replStartCmd.Flags().Int("lifetime-value", 0, "")
	replStartCmd.Flags().String("lifetime-unit", "hour", ""+
		AddFlagsEnum(&g_replStartEnums, "lifetime-unit", []string{"hour","day","week","month","year"}))
	// TODO: "lifetimes" array
	replStartCmd.Flags().String("compression", "lz4", ""+
		AddFlagsEnum(&g_replStartEnums, "compression", []string{"lz4","pigz","plzip"}))
	replStartCmd.Flags().Int64("speed-limit", 0, "") // is this bytes per second?
	replStartCmd.Flags().Bool("large-block", true, "")
	replStartCmd.Flags().Bool("embed", false, "")
	replStartCmd.Flags().Bool("compressed", true, "")
	replStartCmd.Flags().Int("retries", 5, "")
	replStartCmd.Flags().String("logging-level", "warning", ""+
		AddFlagsEnum(&g_replStartEnums, "logging-level", []string{"debug","info","warning","error"}))
	replStartCmd.Flags().Bool("exclude-mountpoint-property", true, "")
	replStartCmd.Flags().Bool("only-from-scratch", false, "")

	replCmd.AddCommand(replStartCmd)
	rootCmd.AddCommand(replCmd)
}

func startReplication(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, g_replStartEnums)
	if err != nil {
		return err
	}

	mainSchemaStr := options.allFlags["naming_schema_main"]
	auxSchemaStr := options.allFlags["naming_schema_aux"]
	regexStr := options.allFlags["name_regex"]

	if mainSchemaStr == "" && auxSchemaStr == "" && regexStr == "" {
		return errors.New("At least one of -n, -N and -R must be provided. To include all, pass -R \".*\".")
	}

	_, sources, err := getHostAndDatasetSpecs(args[0 : len(args)-1])
	if err != nil {
		return err
	}

	_, targets, err := getHostAndDatasetSpecs([]string{args[len(args)-1]})
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return errors.New("No dataset source(s) were given")
	}
	if len(targets) == 0 {
		return errors.New("No dataset target was given")
	}
	if len(targets) != 1 {
		return errors.New("Only one dataset target is allowed")
	}

	// TODO: Implement non-local replication
	transportType := "LOCAL"

	outMap := make(map[string]interface{})

	outMap["direction"] = options.allFlags["direction"]
	outMap["transport"] = transportType
	outMap["source_datasets"] = sources
	outMap["target_dataset"] = targets[0]
	outMap["recursive"] = core.IsValueTrue(options.allFlags, "recursive")
	outMap["retention_policy"] = options.allFlags["retention_policy"]

	if mainSchemaStr != "" {
		outMap["naming_schema"] = strings.Split(mainSchemaStr, ",")
		delete(options.usedFlags, "naming_schema_main")
	}
	if auxSchemaStr != "" {
		outMap["also_include_naming_schema"] = strings.Split(auxSchemaStr, ",")
		delete(options.usedFlags, "naming_schema_aux")
	}
	if regexStr != "" {
		outMap["name_regex"] = regexStr
	}

	_, excludes, err := getHostAndDatasetSpecsFromString(options.allFlags["exclude"])
	if err != nil {
		return err
	}
	if len(excludes) > 0 {
		outMap["exclude"] = excludes
	}

	if optStr := options.allFlags["options"]; optStr != "" {
		err = WriteKvArrayToMap(outMap, ConvertParamsStringToKvArray(optStr), g_replStartEnums)
		if err != nil {
			return err
		}
		delete(options.usedFlags, "options")
	}

	for key, valueStr := range options.usedFlags {
		if _, exists := outMap[key]; exists {
			continue
		}
		outKey := key
		wroteValue := false
		switch key {
		case "aux_properties":
			outKey = "properties"
		case "readonly_policy":
			outKey = "readonly"
		case "properties_exclude":
			outMap[outKey] = strings.Split(valueStr, ",")
			wroteValue = true
		case "properties_override":
			fallthrough
		case "restrict_schedule":
			innerMap := make(map[string]interface{})
			if err = WriteKvArrayToMap(innerMap, ConvertParamsStringToKvArray(valueStr), g_replStartEnums); err != nil {
				return err
			}
			outMap[outKey] = innerMap
			wroteValue = true
		case "periodic_snapshot_tasks":
			valueList := strings.Split(valueStr, ",")
			idList := make([]float64, 0)
			for _, v := range valueList {
				vInt, errNotNumber := strconv.Atoi(strings.Trim(v, " \t\n"))
				if errNotNumber != nil {
					return fmt.Errorf("--periodic-snapshot-tasks requires a list of integers (not \"%s\")", valueStr)
				}
				idList = append(idList, float64(vInt))
			}
			outMap[outKey] = idList
			wroteValue = true
		}
		if !wroteValue {
			value, err := ParseStringAndValidate(key, valueStr, g_replStartEnums)
			if err != nil {
				return err
			}
			outMap[outKey] = value
		}
	}

	params := []interface{}{outMap}
	DebugJson(params)

	cmd.SilenceUsage = true

	jobId, err := core.ApiCallAsync(api, "replication.run_onetime", params, false)
	if err != nil {
		return err
	}

	fmt.Println(jobId)
	return nil
}

func getHostAndDatasetSpecsFromString(str string) ([]string, []string, error) {
	return getHostAndDatasetSpecs(strings.Split(str, ","))
}

func getHostAndDatasetSpecs(array []string) ([]string, []string, error) {
	hosts := make([]string, 0)
	datasets := make([]string, 0)

	for _, s := range array {
		if s == "" || s == " " {
			continue
		}

		var host string
		var obj string

		div := strings.Index(s, ":")
		if div < 0 {
			host = ""
			obj = s
		} else if div == 0 || div == len(s)-1 {
			return nil, nil, fmt.Errorf("Invalid spec \"%s\": must conform to <host>:<dataset> or <dataset>", s)
		} else {
			host = s[0:div]
			obj = s[div+1:]
		}

		t, spec := core.IdentifyObject(obj)
		if t != "dataset" {
			return nil, nil, fmt.Errorf("\"%s\" is not a dataset (%s)", spec, s)
		}

		hosts = append(hosts, host)
		datasets = append(datasets, spec)
	}

	return hosts, datasets, nil
}

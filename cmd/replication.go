package cmd

import (
	"errors"
	"fmt"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:     "replication",
	Short:   "Start replicating a dataset from one pool to another, locally or across any network",
	Aliases: []string{"backup", "repl"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var replStartCmd = &cobra.Command{
	Use:  "start <src dataset>... <dst dataset>",
	Args: cobra.MinimumNArgs(2),
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
	replStartCmd.Flags().String("netcat-active-side", "", "") // "enum": ["LOCAL", "REMOTE"]
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
	replStartCmd.Flags().String("encryption-key-format", "", "") // enum: [ "HEX", "PASSPHRASE" ]
	replStartCmd.Flags().String("encryption-key-location", "", "")
	replStartCmd.Flags().String("periodic-snapshot-tasks", "", "") // int array
	/*
	"naming-schema",
	"also-include-naming-schema",
	"name-regex",
	"restrict-schedule",
	"allow-from-scratch",
	"readonly",
	"hold-pending-snapshots",
	"lifetime-value",
	"lifetime-unit",
	"lifetimes",
	"compression",
	"speed-limit",
	"large-block",
	"embed",
	"compressed",
	"retries",
	"logging-level",
	"exclude-mountpoint-property",
	"only-from-scratch"
	*/

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
	}
	if auxSchemaStr != "" {
		outMap["also_include_naming_schema"] = strings.Split(auxSchemaStr, ",")
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
	}

	params := []interface{}{outMap}
	DebugJson(params)

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

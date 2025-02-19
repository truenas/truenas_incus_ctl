package cmd

import (
	"errors"
	"strings"
	"fmt"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:   "replication",
	Short: "Start replicating a dataset from one pool to another, locally or across any network",
	Aliases: []string{"backup", "repl"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var replStartCmd = &cobra.Command{
	Use:  "start",
	Args: cobra.MinimumNArgs(2),
}

var g_replStartEnums map[string][]string

func init() {
	replStartCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return startReplication(cmd, ValidateAndLogin(), args)
	}

	replStartCmd.Flags().StringP("exclude", "e", "", "")
	replStartCmd.Flags().BoolP("recursive", "r", false, "")
	replStartCmd.Flags().StringP("options", "o", "", "")
	replStartCmd.Flags().StringP("direction", "d", "", ""+
		AddFlagsEnum(&g_replStartEnums, "direction", []string{"push", "pull"}))
	replStartCmd.Flags().StringP("retention-policy", "p", "", ""+
		AddFlagsEnum(&g_replStartEnums, "retention-policy", []string{"source", "custom", "none"}))
	/*
	replStartCmd.Flags().String("transport", "", ""+
		AddFlagsEnum(&g_replStartEnums, "transport", []string{"ssh","ssh+netcat","local"}))
	replStartCmd.Flags().Int("ssh_credentials", 0, "")
	"netcat_active_side",
	"netcat_active_side_listen_address",
	"netcat_active_side_port_min",
	"netcat_active_side_port_max",
	"netcat_passive_side_connect_address",
	"sudo",
	"properties",
	"properties_exclude",
	"properties_override",
	"replicate",
	"encryption",
	"encryption_inherit",
	"encryption_key",
	"encryption_key_format",
	"encryption_key_location",
	"periodic_snapshot_tasks",
	"naming_schema",
	"also_include_naming_schema",
	"name_regex",
	"restrict_schedule",
	"allow_from_scratch",
	"readonly",
	"hold_pending_snapshots",
	"lifetime_value",
	"lifetime_unit",
	"lifetimes",
	"compression",
	"speed_limit",
	"large_block",
	"embed",
	"compressed",
	"retries",
	"logging_level",
	"exclude_mountpoint_property",
	"only_from_scratch"
	*/

	replCmd.AddCommand(replStartCmd)
	rootCmd.AddCommand(replCmd)
}

func startReplication(cmd *cobra.Command, api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	options, err := GetCobraFlags(cmd, g_replStartEnums)
	if err != nil {
		return err
	}

	_, sources, err := getHostAndDatasetSpecs(args[0:len(args)-1])
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
	outMap["recursive"] = options.allFlags["recursive"]
	outMap["retention_policy"] = options.allFlags["retention_policy"]

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

	err = core.ApiCallAsync(api, "replication.run_onetime", params)
	if err != nil {
		return err
	}

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

package cmd

import (
	"fmt"
	"strings"
	"truenas/truenas_incus_ctl/core"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Control the operation of services on the server (eg. iSCSI)",
}

var serviceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "Start one or more services",
	Aliases: []string {"ls"},
}

var serviceReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload the configuration of one or more services",
	Args:  cobra.MinimumNArgs(1),
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart one or more services",
	Args:  cobra.MinimumNArgs(1),
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start one or more services",
	Args:  cobra.MinimumNArgs(1),
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop one or more services",
	Args:  cobra.MinimumNArgs(1),
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable one or more services, allowing them to run on startup",
	Args:  cobra.MinimumNArgs(1),
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable one or more services, preventing them from running on startup",
	Args:  cobra.MinimumNArgs(1),
}

var g_serviceListEnums map[string][]string

func init() {
	serviceListCmd.RunE = WrapCommandFunc(listService)
	serviceReloadCmd.RunE = WrapCommandFunc(changeServiceState)
	serviceRestartCmd.RunE = WrapCommandFunc(changeServiceState)
	serviceStartCmd.RunE = WrapCommandFunc(changeServiceState)
	serviceStopCmd.RunE = WrapCommandFunc(changeServiceState)
	serviceEnableCmd.RunE = WrapCommandFunc(enableOrDisableService)
	serviceDisableCmd.RunE = WrapCommandFunc(enableOrDisableService)

	serviceListCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	serviceListCmd.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
	serviceListCmd.Flags().String("format", "table", "Output table format "+
		AddFlagsEnum(&g_serviceListEnums, "format", []string{"csv", "json", "table", "compact"}))
	serviceListCmd.Flags().StringP("output", "o", "", "Output property list")
	serviceListCmd.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
	serviceListCmd.Flags().BoolP("all", "a", false, "Output all properties")

	_svStateCommands := []*cobra.Command { serviceReloadCmd, serviceRestartCmd, serviceStartCmd, serviceStopCmd }
	for _, cmd := range _svStateCommands {
		cmd.Flags().BoolP("enable", "e", false, "And enable these services")
		cmd.Flags().BoolP("disable", "d", false, "And disable these services")
		cmd.Flags().Bool("silent", false, "pass silent flag to server")
		cmd.Flags().Bool("ha-propagate", false, "pass ha_propagate flag to server")
	}

	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceReloadCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	rootCmd.AddCommand(serviceCmd)
}

func listService(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, false, g_serviceListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	properties := EnumerateOutputProperties(options.allFlags)

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsStringTrue(options.allFlags, "parsable")),
		shouldGetAllProps:  core.IsStringTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      false,
	}

	response, err := QueryApi(api, "service", args, core.StringRepeated("service", len(args)), properties, extras)
	if err != nil {
		return err
	}

	results := GetListFromQueryResponse(&response)

	required := []string { "id", "service", "enable", "state" }
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(results, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, "services", columnsList, results)
	PrintTable(api, str)
	return err
}

func changeServiceState(cmd *cobra.Command, api core.Session, args []string) error {
	cmdType := strings.Split(cmd.Use, " ")[0]

	options, _ := GetCobraFlags(cmd, false, nil)
	cmd.SilenceUsage = true

	isEnable := core.IsStringTrue(options.allFlags, "enable")
	RemoveFlag(options, "enable")
	isDisable := core.IsStringTrue(options.allFlags, "disable")
	RemoveFlag(options, "disable")

	return changeServiceStateImpl(api, cmdType, options.usedFlags, isEnable, isDisable, args)
}

func enableOrDisableService(cmd *cobra.Command, api core.Session, args []string) error {
	cmdType := strings.Split(cmd.Use, " ")[0]
	isEnable := cmdType == "enable"
	if !isEnable && cmdType != "disable" {
		return fmt.Errorf("cmdType was not enable or disable")
	}
	cmd.SilenceUsage = true

	return toggleService(api, args, isEnable, !isEnable)
}

func toggleService(api core.Session, args []string, shouldEnable bool, shouldDisable bool) error {
	if !shouldEnable && !shouldDisable {
		return nil
	}
	if shouldEnable && shouldDisable {
		return fmt.Errorf("--enable and --disable are incompatible")
	}

	paramsArray := make([]interface{}, len(args))
	for i, arg := range args {
		paramsArray[i] = []interface{} { arg, map[string]interface{} { "enable": shouldEnable } }
	}

	_, _, err := MaybeBulkApiCallArray(api, "service.update", int64(10 + 10 * len(paramsArray)), paramsArray, false)
	return err
}

func changeServiceStateImpl(api core.Session, newState string, optionalFlags map[string]string, isEnable, isDisable bool, serviceList []string) error {
	if newState != "reload" && newState != "restart" && newState != "start" && newState != "stop" {
		return fmt.Errorf("The intended state was not reload, restart, start or stop")
	}

	if err := toggleService(api, serviceList, isEnable, isDisable); err != nil {
		return err
	}

	outMap := make(map[string]interface{})

	if optionalFlags != nil {
		for propName, valueStr := range optionalFlags {
			value, _ := ParseStringAndValidate(propName, valueStr, nil)
			outMap[propName] = value
		}
	}

	paramsArray := make([]interface{}, len(serviceList))
	for i, arg := range serviceList {
		paramsArray[i] = []interface{} { arg, outMap }
	}

	out, _, err := MaybeBulkApiCallArray(api, "service." + newState, int64(10 + 10 * len(paramsArray)), paramsArray, true)
	if err != nil {
		return err
	}
	DebugString(string(out))
	return nil
}

package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "Prints a table of datasets/snapshots/shares, given a source and an optional set of properties.",
	Aliases: []string{"ls"},
}

var g_genericListEnums map[string][]string

func init() {
	listCmd.RunE = WrapCommandFunc(doList)

	g_genericListEnums = make(map[string][]string)

	listCmd.Flags().StringP("types", "t", "fs,vol", "Array of types of data to retrieve. By default, types are deduced from arguments, else fs,vol. (fs,vol,snap,nfs)")
	listCmd.Flags().BoolP("recursive", "r", false, "Retrieves properties for children")
	listCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	listCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	listCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_genericListEnums, "format", []string{"csv", "json", "table", "compact"}))
	listCmd.Flags().StringP("output", "o", "", "Output property list")
	listCmd.Flags().BoolP("parseable", "p", false, "Show raw values instead of the already parsed values")
	listCmd.Flags().BoolP("all", "a", false, "Output all properties")

	rootCmd.AddCommand(listCmd)
}

func doList(cmd *cobra.Command, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, g_genericListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	properties := EnumerateOutputProperties(options.allFlags)

	givenTypes := strings.Split(options.allFlags["types"], ",")
	if len(givenTypes) == 0 {
		return errors.New("At least one object type must be provided")
	}

	typesToQuery := make(map[string]bool)
	shouldQueryFs := false
	shouldQueryVol := false
	_, shouldExclude := options.usedFlags["types"]
	if len(args) == 0 {
		shouldExclude = true
	}

	for i := 0; i < len(givenTypes); i++ {
		t := givenTypes[i]
		if t == "fs" || t == "filesystem" {
			t = "dataset"
			shouldQueryFs = true
		} else if t == "vol" || t == "volume" {
			t = "dataset"
			shouldQueryVol = true
		} else if t == "snap" {
			t = "snapshot"
		}
		if t != "dataset" && t != "snapshot" && t != "nfs" {
			return errors.New("Unrecognised object type \"" + t + "\"")
		}
		typesToQuery[t] = true
	}

	qEntriesMap := make(map[string][]string)
	qEntryTypesMap := make(map[string][]string)

	for i := 0; i < len(args); i++ {
		obj, value := core.IdentifyObject(args[i])
		var qType string
		if obj == "id" || obj == "share" {
			qType = "nfs"
		} else if obj == "snapshot" || obj == "snapshot_only" {
			qType = "snapshot"
		} else if obj == "dataset" {
			qType = "dataset"
		} else if obj == "pool" {
			qType = "pool"
		} else {
			return errors.New("Unrecognised namespec \"" + obj + "\"")
		}
		if _, exists := qEntriesMap[qType]; !exists {
			qEntriesMap[qType] = make([]string, 0)
			qEntryTypesMap[qType] = make([]string, 0)
		}
		qEntriesMap[qType] = append(qEntriesMap[qType], value)
		qEntryTypesMap[qType] = append(qEntryTypesMap[qType], obj)
	}

	// NOTE: datasets are added to snapshots before pools are added to datasets
	if _, exists := typesToQuery["snapshot"]; exists || !shouldExclude {
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "dataset", "snapshot", len(args) == 0)
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "pool", "snapshot", len(args) == 0)
	} else if shouldExclude {
		delete(qEntriesMap, "snapshot")
		delete(qEntryTypesMap, "snapshot")
	}

	if _, exists := typesToQuery["dataset"]; exists || !shouldExclude {
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "pool", "dataset", len(args) == 0)
	} else if shouldExclude {
		delete(qEntriesMap, "dataset")
		delete(qEntryTypesMap, "dataset")
	}

	if _, exists := typesToQuery["nfs"]; exists || !shouldExclude {
		if len(args) == 0 {
			qEntriesMap["nfs"] = make([]string, 0)
			qEntryTypesMap["nfs"] = make([]string, 0)
		}
	} else {
		delete(qEntriesMap, "nfs")
		delete(qEntryTypesMap, "nfs")
	}

	delete(qEntriesMap, "pool")
	delete(qEntryTypesMap, "pool")

	tDs := qEntryTypesMap["dataset"]
	for i, _ := range tDs {
		if tDs[i] == "dataset" {
			tDs[i] = "name"
		}
	}

	tSnaps := qEntryTypesMap["snapshot"]
	for i, _ := range tSnaps {
		if tSnaps[i] == "snapshot" {
			tSnaps[i] = "name"
		} else if tSnaps[i] == "snapshot_only" {
			tSnaps[i] = "snapshot_name"
		}
	}

	tShares := qEntryTypesMap["nfs"]
	for i, _ := range tShares {
		if tShares[i] == "share" {
			tShares[i] = "path"
		}
	}

	DebugString(fmt.Sprint(typesToQuery))
	DebugString(fmt.Sprint(qEntriesMap))
	DebugString(fmt.Sprint(qEntryTypesMap))

	var allTypes []string
	for qType, _ := range qEntriesMap {
		if allTypes == nil {
			allTypes = make([]string, 0)
		}
		allTypes = append(allTypes, qType)
	}

	if len(allTypes) == 0 {
		return errors.New("No types could be queried. Try passing a different value to the --types option.")
	}

	slices.Sort(allTypes)

	cmd.SilenceUsage = true

	var outProps []string
	if properties != nil {
		outProps = make([]string, len(properties))
		copy(outProps, properties)
		hasType := false
		for _, p := range properties {
			if p == "type" {
				hasType = true
				break
			}
		}
		if !hasType {
			properties = append(properties, "type")
		}
	}

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsValueTrue(options.allFlags, "parseable")),
		shouldSkipKeyBuild: true,
		shouldGetAllProps:  core.IsValueTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	combinedResponse := typeQueryResponse{}
	combinedResponse.resultsMap = make(map[string]map[string]interface{})
	combinedResponse.intKeys = make([]int, 0)
	combinedResponse.strKeys = make([]string, 0)

	for _, qType := range allTypes {
		var category string
		switch qType {
		case "dataset":
			category = "pool.dataset"
		case "snapshot":
			category = "zfs.snapshot"
		case "nfs":
			category = "sharing.nfs"
		}
		response, err := QueryApi(api, category, qEntriesMap[qType], qEntryTypesMap[qType], properties, extras)
		if err != nil {
			return err
		}

		filterType := ""
		if qType == "dataset" {
			if shouldQueryFs && !shouldQueryVol {
				filterType = "filesystem"
			} else if !shouldQueryFs && shouldQueryVol {
				filterType = "volume"
			}
		}

		for key, r := range response.resultsMap {
			shouldAdd := true
			if qType == "nfs" {
				r["type"] = "nfs"
			} else if t, exists := r["type"]; filterType != "" && exists {
				if tStr, ok := t.(string); ok && strings.ToLower(tStr) != filterType {
					shouldAdd = false
				}
			}
			if shouldAdd {
				combinedResponse.resultsMap[key] = r
				if number, errNotNumber := strconv.Atoi(key); errNotNumber == nil {
					combinedResponse.intKeys = append(combinedResponse.intKeys, number)
				} else {
					combinedResponse.strKeys = append(combinedResponse.strKeys, key)
				}
			}
		}
	}

	allResults := GetListFromQueryResponse(&combinedResponse)

	required := []string{"id"}
	if _, exists := qEntriesMap["nfs"]; exists {
		required = append(required, "path")
	}

	LowerCaseValuesFromEnums(allResults, g_datasetCreateUpdateEnums)
	//LowerCaseValuesFromEnums(allResults, g_snapshotCreateUpdateEnums)
	LowerCaseValuesFromEnums(allResults, g_nfsCreateUpdateEnums)

	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(allResults, required)
	} else if len(properties) > 0 {
		columnsList = outProps
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, "all", columnsList, allResults)
	PrintTable(api, str)
	return err
}

func addEntriesFromInto(allValues, allTypes map[string][]string, srcKey, dstKey string, shouldCreateAnyway bool) {
	if _, exists := allValues[dstKey]; !exists && shouldCreateAnyway {
		allValues[dstKey] = make([]string, 0)
		allTypes[dstKey] = make([]string, 0)
	}
	if values, exists := allValues[srcKey]; exists {
		if _, exists := allValues[dstKey]; !exists {
			allValues[dstKey] = make([]string, 0)
			allTypes[dstKey] = make([]string, 0)
		}
		allValues[dstKey] = append(allValues[dstKey], values...)
		allTypes[dstKey] = append(allTypes[dstKey], allTypes[srcKey]...)
	}
}

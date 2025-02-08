package cmd

import (
	"errors"
	"fmt"
	"strings"
	"truenas/admin-tool/core"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Prints a table of datasets/snapshots/shares, given a source and an optional set of properties.",
}

var g_genericListEnums map[string][]string

func init() {
	listCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return doList(ValidateAndLogin(), args)
	}

	g_genericListEnums = make(map[string][]string)

	listCmd.Flags().StringP("types", "t", "fs,vol", "Array of types of data to retrieve. Defaults to \"fs,vol\". (fs,vol,snap,nfs)")
	listCmd.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
	listCmd.Flags().BoolP("no-headers", "H", false, "Equivalent to --format=compact. More easily parsed by scripts")
	listCmd.Flags().String("format", "table", "Output table format. Defaults to \"table\" "+
		AddFlagsEnum(&g_genericListEnums, "format", []string{"csv", "json", "table", "compact"}))
	listCmd.Flags().StringP("output", "o", "", "Output property list")
	listCmd.Flags().BoolP("all", "a", false, "Output all properties")

	rootCmd.AddCommand(listCmd)
}

func doList(api core.Session, args []string) error {
	if api == nil {
		return nil
	}
	defer api.Close()

	options, err := GetCobraFlags(listCmd, g_genericListEnums)
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

	shouldQueryFs := false
	shouldQueryVol := false
	_, shouldExclude := options.usedFlags["types"]
	typesToQuery := make(map[string]bool)

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
			return errors.New("Unrecognised object type \"" + obj + "\"")
		}
		if _, exists := qEntriesMap[qType]; !exists {
			qEntriesMap[qType] = make([]string, 0)
			qEntryTypesMap[qType] = make([]string, 0)
		}
		qEntriesMap[qType] = append(qEntriesMap[qType], value)
		qEntryTypesMap[qType] = append(qEntryTypesMap[qType], obj)
	}

	listCmd.SilenceUsage = true

	// NOTE: datasets are added to snapshots before pools are added to datasets
	if _, exists := typesToQuery["snapshot"]; exists || !shouldExclude {
		fmt.Println("adding to snapshot")
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "dataset", "snapshot")
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "pool", "snapshot")
	} else if shouldExclude {
		delete(qEntriesMap, "snapshot")
		delete(qEntryTypesMap, "snapshot")
	}
	if _, exists := typesToQuery["dataset"]; exists || !shouldExclude {
		fmt.Println("adding to dataset")
		addEntriesFromInto(qEntriesMap, qEntryTypesMap, "pool", "dataset")
	} else if shouldExclude {
		delete(qEntriesMap, "dataset")
		delete(qEntryTypesMap, "dataset")
	}
	if _, exists := typesToQuery["nfs"]; !exists && shouldExclude {
		delete(qEntriesMap, "nfs")
		delete(qEntryTypesMap, "nfs")
	}

	delete(qEntriesMap, "pool")
	delete(qEntryTypesMap, "pool")

	DebugString(fmt.Sprint(typesToQuery))
	DebugString(fmt.Sprint(qEntriesMap))
	DebugString(fmt.Sprint(qEntryTypesMap))
	if true {
		return nil
	}

	shouldQueryFs = shouldQueryFs
	shouldQueryVol = shouldQueryVol

	extras := typeRetrieveParams{
		retrieveType:       "nfs",
		shouldGetAllProps:  format == "json" || core.IsValueTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsValueTrue(options.allFlags, "recursive"),
	}

	var idTypes []string
	results, err := QueryApi(api, args, idTypes, properties, extras)
	if err != nil {
		return err
	}

	LowerCaseValuesFromEnums(results, g_datasetCreateUpdateEnums)
	//LowerCaseValuesFromEnums(results, g_snapshotCreateUpdateEnums)
	LowerCaseValuesFromEnums(results, g_nfsCreateUpdateEnums)

	required := []string{"id"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(results, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	core.PrintTableDataList(format, "all", columnsList, results)
	return nil
}

func addEntriesFromInto(allValues, allTypes map[string][]string, srcKey, dstKey string) {
	if values, exists := allValues[srcKey]; exists {
		if _, exists := allValues[dstKey]; !exists {
			allValues[dstKey] = make([]string, 0)
			allTypes[dstKey] = make([]string, 0)
		}
		allValues[dstKey] = append(allValues[dstKey], values...)
		allTypes[dstKey] = append(allTypes[dstKey], allTypes[srcKey]...)
	}
}

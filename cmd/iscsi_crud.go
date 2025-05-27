package cmd

import (
	//"encoding/json"
	//"fmt"
	"log"
	//"path"
	//"strconv"
	//"strings"
	//"time"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

type iscsiCrudFeature struct {
	name string
	kind string
	defValue interface{}
	description string
}

var iscsiCrudObjects = []string { "target", "extent", "targetextent", "initiator", "auth" }

var iscsiCrudIdentifierMap = map[string][]string {
	"target": []string {"id", "name", "alias"},
	"extent": []string {"id", "name", "disk"},
	"targetextent": []string {"id", "target", "extent"},
	"initiator": []string {"id", "comment"},
	"auth": []string {"id", "user"},
}

var iscsiCrudFeatureMap = map[string][]iscsiCrudFeature {
	"target": []iscsiCrudFeature {
		iscsiCrudFeature { name: "alias", kind: "String", defValue: "", description: "Path of attached extent" },
	},
	"extent": []iscsiCrudFeature {
		iscsiCrudFeature { name: "disk", kind: "String", defValue: "", description: "Path to zvol" },
	},
	"targetextent": []iscsiCrudFeature {
		iscsiCrudFeature { name: "target", kind: "String", defValue: "", description: "ID or name of target" },
		iscsiCrudFeature { name: "lunid", kind: "Int64", defValue: 0, description: "LUN ID" },
		iscsiCrudFeature { name: "extent", kind: "String", defValue: "", description: "ID or name of extent" },
	},
	"initiator": []iscsiCrudFeature {
		iscsiCrudFeature { name: "initiators", kind: "String", defValue: "", description: "List of initiators in this group" },
		iscsiCrudFeature { name: "comment", kind: "String", defValue: "", description: "Initiator group description/comment" },
	},
	"auth": []iscsiCrudFeature {
		iscsiCrudFeature { name: "user", kind: "String", defValue: "", description: "User name" },
	},
}

func WrapIscsiCrudFunc(cmdFunc func(*cobra.Command,string,core.Session,[]string)error, objectType string) func(*cobra.Command,[]string) error {
	return func(cmd *cobra.Command, args []string) error {
		api := InitializeApiClient()
		if api == nil {
			return nil
		}
		err := cmdFunc(cmd, objectType, api, args)
		return api.Close(err)
	}
}

func AddIscsiCrudCommandFlag(cmd *cobra.Command, feature iscsiCrudFeature) {
	switch feature.kind {
		case "String":
			cmd.Flags().String(feature.name, feature.defValue.(string), feature.description)
		case "Int64":
			if intValue, ok := feature.defValue.(int); ok {
				cmd.Flags().Int(feature.name, intValue, feature.description)
			} else if int64Value, ok := feature.defValue.(int64); ok {
				cmd.Flags().Int64(feature.name, int64Value, feature.description)
			}
		default:
			log.Fatal("Flag type \"" + feature.kind + "\" is currently not supported by AddIscsiCrudCommandFlag()")
	}
}

var iscsiCrudListEnums map[string][]string

func AddIscsiCrudCommands(parentCmd *cobra.Command) {
	listFormatDesc := AddFlagsEnum(&iscsiCrudListEnums, "format", []string{"csv", "json", "table", "compact"})

	for _, object := range iscsiCrudObjects {
		cmdList := &cobra.Command { Use: "list", RunE: WrapIscsiCrudFunc(iscsiCrudList, object) }
		cmdCreate := &cobra.Command { Use: "create", RunE: WrapIscsiCrudFunc(iscsiCrudUpdateCreate, object) }
		cmdUpdate := &cobra.Command { Use: "update", RunE: WrapIscsiCrudFunc(iscsiCrudUpdateCreate, object) }
		cmdDelete := &cobra.Command { Use: "delete", RunE: WrapIscsiCrudFunc(iscsiCrudDelete, object) }

		features := iscsiCrudFeatureMap[object]
		for _, f := range features {
			AddIscsiCrudCommandFlag(cmdCreate, f)
			AddIscsiCrudCommandFlag(cmdUpdate, f)
		}

		cmdList.Flags().BoolP("recursive", "r", false, "")
		cmdList.Flags().BoolP("user-properties", "u", false, "Include user-properties")
		cmdList.Flags().BoolP("json", "j", false, "Equivalent to --format=json")
		cmdList.Flags().BoolP("no-headers", "c", false, "Equivalent to --format=compact. More easily parsed by scripts")
		cmdList.Flags().String("format", "table", "Output table format. Defaults to \"table\" " + listFormatDesc)
		cmdList.Flags().StringP("output", "o", "", "Output property list")
		cmdList.Flags().BoolP("parsable", "p", false, "Show raw values instead of the already parsed values")
		cmdList.Flags().Bool("all", false, "Output all properties")

		cmd := &cobra.Command { Use: object }
		cmd.AddCommand(cmdList)
		cmd.AddCommand(cmdCreate)
		cmd.AddCommand(cmdUpdate)
		cmd.AddCommand(cmdDelete)
		parentCmd.AddCommand(cmd)
	}
}

func addIscsiQuery(api core.Session, object string, results []map[string]interface{}, values []string, properties []string, extras typeQueryParams) []map[string]interface{} {
	params := iscsiCrudIdentifierMap[object]
	for _, kind := range params {
		response, err := QueryApi(api, "iscsi." + object, values, core.StringRepeated(kind, len(values)), properties, extras)
		if err != nil {
			continue
		}
		subResults := GetListFromQueryResponse(&response)
		if len(subResults) > 0 {
			results = append(results, subResults...)
		}
	}
	return results
}

func iscsiCrudList(cmd *cobra.Command, object string, api core.Session, args []string) error {
	options, err := GetCobraFlags(cmd, false, iscsiCrudListEnums)
	if err != nil {
		return err
	}

	format, err := GetTableFormat(options.allFlags)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	extras := typeQueryParams{
		valueOrder:         BuildValueOrder(core.IsStringTrue(options.allFlags, "parsable")),
		shouldGetAllProps:  core.IsStringTrue(options.allFlags, "all"),
		shouldGetUserProps: false,
		shouldRecurse:      len(args) == 0 || core.IsStringTrue(options.allFlags, "recursive"),
	}

	var properties []string
	results := make([]map[string]interface{}, 0)

	if object == "targetextent" {
		results = addIscsiQuery(api, "targetextent", results, args, properties, extras)
		results = addIscsiQuery(api, "target", results, args, properties, extras)
		results = addIscsiQuery(api, "extent", results, args, properties, extras)
	} else {
		properties = EnumerateOutputProperties(options.allFlags)
		results = addIscsiQuery(api, object, results, args, properties, extras)
	}

	required := []string{"name"}
	var columnsList []string
	if extras.shouldGetAllProps {
		columnsList = GetUsedPropertyColumns(results, required)
	} else if len(properties) > 0 {
		columnsList = properties
	} else {
		columnsList = required
	}

	str, err := core.BuildTableData(format, object + "s", columnsList, results)
	PrintTable(api, str)
	return err
}

func iscsiCrudUpdateCreate(cmd *cobra.Command, object string, api core.Session, args []string) error {
	return nil
}

func iscsiCrudDelete(cmd *cobra.Command, object string, api core.Session, args []string) error {
	return nil
}

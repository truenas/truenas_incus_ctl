package cmd

import (
	"errors"
	"log"
	"os"
	"slices"
	"truenas/admin-tool/core"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "admin-tool",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var g_useMock bool
var g_url string
var g_apiKey string
var g_keyFile string

func init() {
	rootCmd.PersistentFlags().BoolVar(&g_useMock, "mock", false, "Use the mock API instead of a TrueNAS server")
	rootCmd.PersistentFlags().StringVarP(&g_url, "url", "U", "", "Server URL")
	rootCmd.PersistentFlags().StringVarP(&g_apiKey, "api-key", "K", "", "API key")
	rootCmd.PersistentFlags().StringVar(&g_keyFile, "key-file", "", "Text file containing server URL on the first line, API key on the second")
}

func RemoveGlobalFlags(flags map[string]string) {
	delete(flags, "mock")
	delete(flags, "url")
	delete(flags, "api-key")
	delete(flags, "api_key")
	delete(flags, "key-file")
	delete(flags, "key_file")
}

func ValidateAndLogin() core.Session {
	var api core.Session
	if g_useMock {
		api = &core.MockSession{
			DatasetSource: &core.FileRawa{FileName: "datasets.tsv"},
		}
	} else {
		api = &core.RealSession{
			HostUrl:     g_url,
			ApiKey:      g_apiKey,
			KeyFileName: g_keyFile,
		}
	}

	err := api.Login()
	if err != nil {
		api.Close()
		log.Fatal(err)
	}

	return api
}

func MakePropertyColumns(required []string, additional []string) []string {
	columnSet := make(map[string]bool)
	uniqAdditional := make([]string, 0, 0)

	for _, c := range required {
		columnSet[c] = true
	}
	for _, c := range additional {
		if _, exists := columnSet[c]; !exists {
			uniqAdditional = append(uniqAdditional, c)
		}
		columnSet[c] = true
	}

	slices.Sort(uniqAdditional)
	return append(required, uniqAdditional...)
}

func GetUsedPropertyColumns(data []map[string]interface{}, required []string) []string {
	columnsMap := make(map[string]bool)
	columnsList := make([]string, 0)

	for _, c := range required {
		columnsMap[c] = true
	}

	for _, d := range data {
		for key, _ := range d {
			if _, exists := columnsMap[key]; !exists {
				columnsMap[key] = true
				columnsList = append(columnsList, key)
			}
		}
	}

	slices.Sort(columnsList)
	return append(required, columnsList...)
}

func GetTableFormat(properties map[string]string) (string, error) {
	isJson := core.IsValueTrue(properties, "json")
	isCompact := core.IsValueTrue(properties, "no-headers")
	if isJson && isCompact {
		return "", errors.New("--json and --no-headers cannot be used together")
	} else if isJson {
		return "json", nil
	} else if isCompact {
		return "compact", nil
	}

	return properties["format"], nil
}

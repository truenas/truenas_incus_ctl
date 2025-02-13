package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"truenas/truenas-admin/core"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "truenas-admin",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var g_useMock bool
var g_debug bool
var g_url string
var g_apiKey string
var g_keyFile string

func init() {
	rootCmd.PersistentFlags().BoolVar(&g_debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().BoolVar(&g_useMock, "mock", false, "Use the mock API instead of a TrueNAS server")
	rootCmd.PersistentFlags().StringVarP(&g_url, "url", "U", "", "Server URL")
	rootCmd.PersistentFlags().StringVarP(&g_apiKey, "api-key", "K", "", "API key")
	rootCmd.PersistentFlags().StringVar(&g_keyFile, "key-file", "", "Text file containing server URL on the first line, API key on the second")
}

func RemoveGlobalFlags(flags map[string]string) {
	delete(flags, "debug")
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

func DebugString(str string) {
	if g_debug {
		fmt.Println(str)
	}
}

func DebugJson(obj interface{}) {
	if g_debug {
		data, err := json.Marshal(obj)
		if err != nil {
			fmt.Printf("%v (%v)", obj, err)
		}
		fmt.Println(string(data))
	}
}

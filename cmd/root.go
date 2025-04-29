package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "truenas_incus_ctl",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var g_useMock bool
var g_debug bool
var g_async bool
var g_configFileName string
var g_configHost string
var g_url string
var g_apiKey string

func init() {
	rootCmd.PersistentFlags().BoolVar(&g_debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().BoolVar(&g_useMock, "mock", false, "Use the mock API instead of a TrueNAS server")
	rootCmd.PersistentFlags().BoolVar(&g_async, "nowait", false, "Disable waiting until every job completes")
	rootCmd.PersistentFlags().StringVar(&g_configFileName, "config", "", "Override config filename (~/.truenas_incus_ctl/config.json)")
	rootCmd.PersistentFlags().StringVar(&g_configHost, "host", "", "Name of config to look up in config.json, defaults to first entry")
	rootCmd.PersistentFlags().StringVarP(&g_url, "url", "U", "", "Server URL")
	rootCmd.PersistentFlags().StringVarP(&g_apiKey, "api-key", "K", "", "API key")
}

func RemoveGlobalFlags(flags map[string]string) {
	delete(flags, "debug")
	delete(flags, "mock")
	delete(flags, "nowait")
	delete(flags, "config")
	delete(flags, "host")
	delete(flags, "url")
	delete(flags, "api-key")
	delete(flags, "api_key")
}

func InitializeApiClient() core.Session {
	var api core.Session
	if g_useMock {
		api = &core.MockSession{
			DatasetSource: &core.FileRawa{FileName: "datasets.tsv"},
		}
	} else {
		if g_url == "" && g_apiKey == "" {
			var err error
			g_url, g_apiKey, err = loadConfig(g_configFileName, g_configHost)
			if err != nil {
				log.Fatal(fmt.Errorf("Failed to parse config: %v", err))
			}
		}
		api = &core.RealSession{
			HostUrl:     g_url,
			ApiKey:      g_apiKey,
			ShouldWait:  !g_async,
			IsDebug:     g_debug,
		}
	}

	/*
	err := api.Login()
	if err != nil {
		api.Close(err)
		log.Fatal(err)
	}
	*/

	return api
}

func loadConfig(fileName, hostName string) (string, string, error) {
	var data []byte
	var err error

	if fileName == "" {
		fileName = getDefaultConfigPath()
		data, err = os.ReadFile(fileName)
	} else {
		data, err = os.ReadFile(fileName)
		if err != nil {
			fileName = getDefaultConfigPath()
			data, err = os.ReadFile(fileName)
		}
	}

	if err != nil {
		return "", "", err
	}

	var obj interface{}
	if err = json.Unmarshal(data, &obj); err != nil {
		return "", "", fmt.Errorf("\"%s\": %v", fileName, err)
	}

	config, ok := obj.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("Config was not a JSON object \"%s\"", fileName)
	}

	hosts, err := getMapFromMapAny(config, "hosts", fileName)
	if err != nil {
		return "", "", err
	}

	if hostName == "" {
		for key, _ := range hosts {
			if hostName == "" || key < hostName {
				hostName = key
			}
		}
		if hostName == "" {
			return "", "", fmt.Errorf("Could not find any hosts in config \"%s\"", fileName)
		}
	}

	host, err := getMapFromMapAny(hosts, hostName, fileName)
	if err != nil {
		return "", "", err
	}

	apiKey, err := getNonEmptyStringFromMapAny(host, "api_key", fileName)
	if err != nil {
		return "", "", err
	}

	url, err := getNonEmptyStringFromMapAny(host, "url", fileName)
	if err != nil {
		return "", "", err
	}

	return url, apiKey, nil
}

func getDefaultConfigPath() string {
	p, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return p + "/.truenas_incus_ctl/config.json"
}

func getMapFromMapAny(dict map[string]interface{}, key, fileName string) (map[string]interface{}, error) {
	var inner map[string]interface{}
	if innerObj, exists := dict[key]; exists {
		var ok bool
		inner, ok = innerObj.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("\"%s\" in config \"%s\" was not a JSON object", key, fileName)
		}
	} else {
		return nil, fmt.Errorf("Could not find \"%s\" in config \"%s\"", key, fileName)
	}
	return inner, nil
}

func getNonEmptyStringFromMapAny(dict map[string]interface{}, key, fileName string) (string, error) {
	var str string
	if strObj, exists := dict[key]; exists {
		var ok bool
		str, ok = strObj.(string)
		if !ok {
			return "", fmt.Errorf("\"%s\" in config \"%s\" was not a string", key, fileName)
		}
	} else {
		return "", fmt.Errorf("Could not find \"%s\" in config \"%s\"", key, fileName)
	}
	if str == "" {
		return "", fmt.Errorf("\"%s\" in config \"%s\" was left blank", key, fileName)
	}
	return str, nil
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

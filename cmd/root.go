package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

const USE_DAEMON = true

var rootCmd = &cobra.Command{
	Use: "truenas_incus_ctl",
}

var daemonCmd = &cobra.Command{
	Use:  "daemon",
	Args: cobra.MinimumNArgs(1),
	Run:  runDaemon,
}

var g_debug bool
var g_allowInsecure bool
var g_configFileName string
var g_configName string
var g_hostName string
var g_apiKey string

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&g_debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().BoolVar(&g_allowInsecure, "allow-insecure", false, "Allow self-signed or non-trusted SSL certificates")
	rootCmd.PersistentFlags().StringVarP(&g_configFileName, "config-file", "F", "", "Override config filename (~/.truenas_incus_ctl/config.json)")
	rootCmd.PersistentFlags().StringVarP(&g_configName, "config", "C", "", "Name of config to look up in config.json, defaults to first entry")
	rootCmd.PersistentFlags().StringVarP(&g_hostName, "host", "H", "", "Server hostname or URL")
	rootCmd.PersistentFlags().StringVarP(&g_apiKey, "api-key", "K", "", "API key")

	daemonCmd.Flags().StringP("timeout", "t", "", "Exit the daemon if no communication occurs after this duration")

	rootCmd.AddCommand(daemonCmd)
}

func RemoveGlobalFlags(flags map[string]string) {
	core.DeleteSnakeKebab(flags, "debug")
	core.DeleteSnakeKebab(flags, "mock")
	core.DeleteSnakeKebab(flags, "allow-insecure")
	core.DeleteSnakeKebab(flags, "config-file")
	core.DeleteSnakeKebab(flags, "config")
	core.DeleteSnakeKebab(flags, "host")
	core.DeleteSnakeKebab(flags, "api-key")
}

func runDaemon(cmd *cobra.Command, args []string) {
	var globalTimeoutStr string
	f := cmd.Flags().Lookup("timeout")
	if f != nil {
		globalTimeoutStr = f.Value.String()
	}
	serverSockAddr := args[0]
	if serverSockAddr == "" {
		log.Fatal("Error: path to server socket was not provided")
	}
	core.RunDaemon(serverSockAddr, globalTimeoutStr, g_allowInsecure)
}

func InitializeApiClient() core.Session {
	var api core.Session
	if g_hostName == "" || g_apiKey == "" {
		host, key, config, err := findCredsFromConfig(g_configFileName, g_configName, g_hostName, g_apiKey)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to parse config: %v", err))
		}
		g_hostName = host
		g_apiKey = key
		if _, exists := config["debug"]; exists {
			g_debug = core.IsValueTrue(config, "debug")
		}
		if _, exists := config["allow_insecure"]; exists {
			g_allowInsecure = core.IsValueTrue(config, "allow_insecure")
		}
	}
	if USE_DAEMON {
		p, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		api = &core.ClientSession{
			HostName:      g_hostName,
			ApiKey:        g_apiKey,
			SocketPath:    path.Join(p, "tncdaemon.sock"),
			AllowInsecure: g_allowInsecure,
		}
	} else {
		api = &core.RealSession{
			HostName:      g_hostName,
			ApiKey:        g_apiKey,
			IsDebug:       g_debug,
			AllowInsecure: g_allowInsecure,
		}
	}

	return api
}

// This method is called assuming that we're missing either a hostname or api key.
// Additionally, we might not know the config path (in which case we use the default),
// or the name (in which case we just pick the first config in the list)
func findCredsFromConfig(fileName, name, existingHost, existingApiKey string) (string, string, map[string]interface{}, error) {
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
		return "", "", nil, err
	}

	var obj interface{}
	if err = json.Unmarshal(data, &obj); err != nil {
		return "", "", nil, fmt.Errorf("\"%s\": %v", fileName, err)
	}

	jsonObj, ok := obj.(map[string]interface{})
	if !ok {
		return "", "", nil, fmt.Errorf("Config was not a JSON object \"%s\"", fileName)
	}

	hosts, err := getMapFromMapAny(jsonObj, "hosts", fileName)
	if err != nil {
		return "", "", nil, err
	}

	if name == "" {
		if existingHost != "" {
			existingHostCondensed := core.GetHostNameFromApiUrl(existingHost)
			for key, value := range hosts {
				if valueMap, ok := value.(map[string]interface{}); ok {
					url, _ := valueMap["url"].(string)
					if url != "" && core.GetHostNameFromApiUrl(url) == existingHostCondensed {
						name = key
						break
					}
				}
			}
		} else if existingApiKey != "" {
			for key, value := range hosts {
				if valueMap, ok := value.(map[string]interface{}); ok {
					apiKey, _ := valueMap["api_key"].(string)
					if apiKey == existingApiKey {
						name = key
						break
					}
				}
			}
		} else {
			for key, _ := range hosts {
				if name == "" || key < name {
					name = key
				}
			}
		}
		if name == "" {
			return "", "", nil, fmt.Errorf("Could not find any matching hosts in config \"%s\"", fileName)
		}
	}

	config, err := getMapFromMapAny(hosts, name, fileName)
	if err != nil {
		return "", "", nil, err
	}

	apiKey, err := getNonEmptyStringFromMapAny(config, "api_key", fileName)
	if err != nil {
		return "", "", nil, err
	}

	u, err := getNonEmptyStringFromMapAny(config, "url", fileName)
	if err != nil {
		return "", "", nil, err
	}

	return u, apiKey, config, nil
}

func getDefaultConfigPath() string {
	p, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(p, ".truenas_incus_ctl", "config.json")
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

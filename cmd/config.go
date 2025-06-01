package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"syscall"
	"time"
	"truenas/truenas_incus_ctl/core"
	"truenas/truenas_incus_ctl/truenas_api"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local configuration settings",
	Long: `Manage TrueNAS connection configurations.
	
Available Commands:
  login                             - Interactively add a new connection
  add <name> [parameters...]    - Non-interactively add a new connection
  set <name> [parameters...]    - Update parameters in config file
  list                              - Lists all saved connections
  show                              - Display the raw contents of the configuration file
  remove <name>                 - Remove a saved connection by name`,
	Example: `  # Add a new connection interactively
  truenas_incus_ctl config login

  # Add a new connection non-interactively
  truenas_incus_ctl config add prod-server --host 192.168.0.31 --api-key "api-key-goes-here"

  # List all saved connections
  truenas_incus_ctl config list

  # Remove a connection
  truenas_incus_ctl config remove truenas-production`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.HelpFunc()(cmd, args)
			return
		}
	},
}

var configLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Interactively add a new connection to the configuration",
	Args:  cobra.NoArgs,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the contents of the configuration file",
	Args:  cobra.NoArgs,
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all saved connection names",
	Args:    cobra.NoArgs,
	Aliases: []string{"ls"},
}

var configRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Short:   "Remove a saved connection by name",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"delete", "del", "rm"},
}

var configAddCmd = &cobra.Command{
	Use:     "add <name> [parameters...]",
	Short:   "Non-interactively add a new connection to the configuration",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"create"},
}

var configSetCmd = &cobra.Command{
	Use:     "set <name> [parameters...]",
	Short:   "Edit the configuration of the given connection",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"update"},
}

func init() {
	configLoginCmd.RunE = WrapCommandFuncWithoutApi(loginToHost)
	configShowCmd.RunE = WrapCommandFunc(showConfig)
	configListCmd.RunE = WrapCommandFunc(listConfigs)
	configRemoveCmd.RunE = WrapCommandFunc(removeConfig)
	configAddCmd.RunE = WrapCommandFuncWithoutApi(addHost)
	configSetCmd.RunE = WrapCommandFuncWithoutApi(setConfig)

	_configEditCommands := []*cobra.Command {configAddCmd, configSetCmd}
	for _, c := range _configEditCommands {
		c.Flags().Bool("no-verify", false, "Don't verify the new host and API key before updating the config")
	}

	configCmd.AddCommand(configLoginCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func showConfig(cmd *cobra.Command, api core.Session, args []string) error {
	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to read config file %s: %v", configPath, err)
	}

	// Pretty print the JSON
	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
	}

	// Pretty print the JSON with indentation
	prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to format config file: %v", err)
	}

	fmt.Printf("Configuration file: %s\n\n", configPath)
	fmt.Println(string(prettyJSON))
	return nil
}

// addHost implements the non-interactive version of adding a connection to the config
func addHost(cmd *cobra.Command, api core.Session, args []string) error {
	// Note: 'api' parameter will be nil for this command, which is expected
	options, _ := GetCobraFlags(cmd, true, nil)
	name := args[0]
	hostname := options.allFlags["host"]
	apiKey := options.allFlags["api_key"]
	strDebug, passedDebug := options.usedFlags["debug"]
	strInsecure, passedInsecure := options.usedFlags["allow_insecure"]
	sockPath, passedSockPath := options.usedFlags["daemon_socket"]

	if hostname == "" {
		return fmt.Errorf("Hostname cannot be empty")
	}
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if !core.IsStringTrue(options.allFlags, "no_verify") {
		if err := verifyHost(hostname, apiKey); err != nil {
			return err
		}
	}

	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	configs, err := loadConfig(configPath)

	// Add or update host entry with URL including API endpoint
	// Store the complete URL with /api/current path under the name
	hostConfig := map[string]interface{}{
		"url":     hostname,
		"api_key": apiKey,
	}
	if passedDebug {
		hostConfig["debug"] = strDebug == "true"
	}
	if passedInsecure {
		hostConfig["allow_insecure"] = strInsecure == "true"
	}
	if passedSockPath {
		hostConfig["daemon_socket"] = sockPath
	}

	hosts, _ := configs["hosts"].(map[string]interface{})
	hosts[name] = hostConfig
	configs["hosts"] = hosts

	if err = saveConfig(configPath, configs); err != nil {
		return err
	}

	fmt.Printf("Configuration for '%s' (connecting to %s) saved to %s\n", name, hostname, configPath)
	return nil
}

func setConfig(cmd *cobra.Command, api core.Session, args []string) error {
	// Note: 'api' parameter will be nil for this command, which is expected
	options, _ := GetCobraFlags(cmd, true, nil)
	name := args[0]
	hostname := options.allFlags["host"]
	apiKey := options.allFlags["api_key"]
	strDebug, passedDebug := options.usedFlags["debug"]
	strInsecure, passedInsecure := options.usedFlags["allow_insecure"]
	sockPath, passedSockPath := options.usedFlags["daemon_socket"]

	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	configs, err := loadConfig(configPath)

	hosts, _ := configs["hosts"].(map[string]interface{})
	if len(hosts) == 0 {
		return fmt.Errorf("Could not find hosts in config file")
	}
	profile, _ := hosts[name].(map[string]interface{})
	if len(profile) == 0 {
		return fmt.Errorf("Could not find host \"%s\" in config file", name)
	}

	if hostname == "" {
		hostname, _ = profile["url"].(string)
		if hostname == "" {
			return fmt.Errorf("Hostname cannot be empty")
		}
	} else {
		profile["url"] = hostname
	}

	if apiKey == "" {
		apiKey, _ = profile["api_key"].(string)
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
	} else {
		profile["api_key"] = apiKey
	}

	if !core.IsStringTrue(options.allFlags, "no_verify") {
		if err := verifyHost(hostname, apiKey); err != nil {
			return err
		}
	}

	if passedDebug {
		profile["debug"] = strDebug == "true"
	}
	if passedInsecure {
		profile["allow_insecure"] = strInsecure == "true"
	}
	if passedSockPath {
		profile["daemon_socket"] = sockPath
	}

	hosts[name] = profile
	configs["hosts"] = hosts

	if err = saveConfig(configPath, configs); err != nil {
		return err
	}

	fmt.Printf("Configuration for '%s' (connecting to %s) saved to %s\n", name, hostname, configPath)
	return nil
}

// listConfigs lists all connection names in the config
func listConfigs(cmd *cobra.Command, api core.Session, args []string) error {
	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No configurations found (config file does not exist)")
			return nil
		}
		return fmt.Errorf("Failed to read config file %s: %v", configPath, err)
	}

	// Parse the JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
	}

	// Extract and print the names
	hosts, ok := config["hosts"].(map[string]interface{})
	if !ok || len(hosts) == 0 {
		fmt.Println("No connections configured")
		return nil
	}

	fmt.Println("Available connection names:")
	for name := range hosts {
		fmt.Printf("  %s\n", name)
	}
	
	return nil
}

// removeConfig removes a connection from the config by name
func removeConfig(cmd *cobra.Command, api core.Session, args []string) error {
	name := args[0]

	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Config file does not exist: %s", configPath)
		}
		return fmt.Errorf("Failed to read config file %s: %v", configPath, err)
	}

	// Parse the JSON
	var configs map[string]interface{}
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
	}

	// Check if hosts section exists
	hosts, ok := configs["hosts"].(map[string]interface{})
	if !ok || len(hosts) == 0 {
		return fmt.Errorf("No connections configured in %s", configPath)
	}

	// Check if the name exists
	if _, exists := hosts[name]; !exists {
		return fmt.Errorf("Connection with name '%s' not found", name)
	}

	// Remove the connection
	delete(hosts, name)
	configs["hosts"] = hosts

	if err = saveConfig(configPath, configs); err != nil {
		return err
	}

	fmt.Printf("Connection '%s' has been removed from the configuration\n", name)
	return nil
}

func saveConfig(configPath string, configs map[string]interface{}) error {
	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize config: %v", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write config to %s: %v", configPath, err)
	}

	return nil
}

func loadConfig(configPath string) (map[string]interface{}, error) {
	// Ensure the config directory exists
	configDir := path.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create config directory %s: %v", configDir, err)
	}

	// Read existing config or create new config
	var configs map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If the file doesn't exist, create a new config
		configs = make(map[string]interface{})
	} else {
		// Parse existing config
		if err := json.Unmarshal(data, &configs); err != nil {
			return nil, fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
		}
	}

	// Ensure hosts section exists and is a map
	if _, ok := configs["hosts"].(map[string]interface{}); !ok {
		configs["hosts"] = make(map[string]interface{})
	}

	return configs, nil
}

func verifyHost(hostname, apiKey string) error {
	// Construct the WebSocket URL with API endpoint
	url := core.GetApiUrlFromHostName(hostname)
	fmt.Printf("Testing connection to %s...\n", url)

	// Test the connection by creating a temporary client
	// Pass false to disable SSL verification and allow self-signed certificates
	client, err := truenas_api.NewClient(url, false)
	if err != nil {
		return fmt.Errorf("Failed to create connection to %s: %v", url, err)
	}
	defer client.Close()

	// Attempt to login to verify API key
	err = client.Login("", "", apiKey)
	if err != nil {
		return fmt.Errorf("Failed to login to %s: %v", url, err)
	}

	// Test basic connectivity with a ping
	result, err := client.Ping()
	if err != nil {
		return fmt.Errorf("Failed to ping %s: %v", url, err)
	}

	if result != "pong" {
		return fmt.Errorf("Unexpected ping response from %s: %s", url, result)
	}

	fmt.Printf("Successfully connected to %s\n", url)
	return nil
}

// loginToHost implements the login subcommand functionality
func loginToHost(cmd *cobra.Command, api core.Session, args []string) error {
	// Note: 'api' parameter will be nil for this command, which is expected

	// Get the config file path to check for existing names
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Read existing config if it exists
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err == nil {
		// Parse existing config
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("Error reading config file %s: %v", configPath, err)
	} else {
		// If file doesn't exist, create a new config
		config = make(map[string]interface{})
	}

	// Ensure hosts section exists
	var hosts map[string]interface{}
	if hostsObj, ok := config["hosts"]; ok {
		hosts, _ = hostsObj.(map[string]interface{})
	}
	if hosts == nil {
		hosts = make(map[string]interface{})
		config["hosts"] = hosts
	}

	// Prompt for name and validate it doesn't exist
	var name string
	for {
		fmt.Print("Enter a name for this connection: ")
		fmt.Scanln(&name)
		if name == "" {
			fmt.Println("name cannot be empty. Please try again.")
			continue
		}
		if _, exists := hosts[name]; exists {
			fmt.Printf("Error: A connection with name '%s' already exists. Please choose a different name.\n", name)
			continue
		}
		break
	}

	// Prompt for hostname
	var hostname string
	for {
		fmt.Print("Enter the TrueNAS hostname or IP address: ")
		fmt.Scanln(&hostname)
		if hostname == "" {
			fmt.Println("Hostname cannot be empty. Please try again.")
			continue
		}
		break
	}
	fmt.Printf("Setting up connection to TrueNAS host: %s\n", hostname)

	// Prompt for authentication method
	var authMethod string
	for {
		fmt.Print("Choose authentication method (1 for API Key, 2 for Username/Password): ")
		fmt.Scanln(&authMethod)
		if authMethod != "1" && authMethod != "2" {
			fmt.Println("Please enter either 1 or 2 to select your authentication method.")
			continue
		}
		break
	}

	// Construct the WebSocket URL with API endpoint
	url := core.GetApiUrlFromHostName(hostname)
	fmt.Printf("Testing connection to %s...\n", url)

	// Test the connection by creating a temporary client
	// Pass false to disable SSL verification and allow self-signed certificates
	client, err := truenas_api.NewClient(url, false)
	if err != nil {
		return fmt.Errorf("Failed to create connection to %s: %v", url, err)
	}

	var apiKey string
	if authMethod == "1" {
		// Prompt for API key
		for {
			fmt.Print("Enter your TrueNAS API key: ")
			fmt.Scanln(&apiKey)
			if apiKey == "" {
				fmt.Println("API key cannot be empty. Please try again.")
				continue
			}
			break
		}

		// Attempt to login to verify API key
		err = client.Login("", "", apiKey)
		if err != nil {
			client.Close()
			return fmt.Errorf("Failed to login to %s: %v", url, err)
		}
	} else {
		// Prompt for username
		var username string
		for {
			fmt.Print("Enter your TrueNAS username: ")
			fmt.Scanln(&username)
			if username == "" {
				fmt.Println("Username cannot be empty. Please try again.")
				continue
			}
			break
		}

		// Prompt for password with masking
		var password string
		for {
			fmt.Print("Enter your TrueNAS password: ")
			// ReadPassword will disable echo and read password from terminal
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // Add a newline after password input

			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				continue
			}

			password = string(bytePassword)
			if password == "" {
				fmt.Println("Password cannot be empty. Please try again.")
				continue
			}
			break
		}

		// Attempt to login with username and password
		err = client.Login(username, password, "")
		if err != nil {
			client.Close()
			return fmt.Errorf("Failed to login to %s with username/password: %v", url, err)
		}

		// Generate an API key using api_key.create
		fmt.Println("Generating API key...")
		currentTime := time.Now().Format("2006-01-02")
		keyName := fmt.Sprintf("Auto-Generated by truenas_incus_ctl %s", currentTime)

		// Create API key
		params := []interface{}{
			map[string]interface{}{
				"name":     keyName,
				"username": username,
			},
		}

		res, err := client.Call("api_key.create", 30, params)
		if err != nil {
			client.Close()
			return fmt.Errorf("Failed to create API key: %v", err)
		}

		// Parse the response to get the API key
		var response map[string]interface{}
		if err := json.Unmarshal(res, &response); err != nil {
			client.Close()
			return fmt.Errorf("Failed to parse API key creation response: %v", err)
		}

		// Check for errors in the response
		if errorData, exists := response["error"]; exists {
			client.Close()
			return fmt.Errorf("API key creation error: %v", errorData)
		}

		// Extract the API key from the result
		result, ok := response["result"].(map[string]interface{})
		if !ok {
			client.Close()
			return fmt.Errorf("Unexpected response format for API key creation")
		}

		apiKey, ok = result["key"].(string)
		if !ok {
			client.Close()
			return fmt.Errorf("Could not extract API key from response")
		}

		fmt.Println("API key successfully generated")
	}

	// Test basic connectivity with a ping
	result, err := client.Ping()
	if err != nil {
		client.Close()
		return fmt.Errorf("Failed to ping %s: %v", url, err)
	}

	if result != "pong" {
		client.Close()
		return fmt.Errorf("Unexpected ping response from %s: %s", url, result)
	}

	fmt.Printf("Successfully connected to %s\n", url)
	client.Close()

	// Ensure the config directory exists
	configDir := path.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("Failed to create config directory %s: %v", configDir, err)
	}

	// Add or update host entry with URL including API endpoint
	// Store the complete URL with /api/current path under the name
	hostConfig := map[string]interface{}{
		"url":     url, // Using the same URL with /api/current path
		"api_key": apiKey,
	}
	hosts[name] = hostConfig

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize config: %v", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write config to %s: %v", configPath, err)
	}

	fmt.Printf("Configuration for '%s' (connecting to %s) saved to %s\n", name, hostname, configPath)
	return nil
}

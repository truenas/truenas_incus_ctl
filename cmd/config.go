package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"truenas/truenas_incus_ctl/core"
	"truenas/truenas_incus_ctl/truenas_api"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local configuration settings",
	Long: `Manage TrueNAS connection configurations.
	
Available Commands:
  login <hostname>  - Interactively add a new host to the configuration file
  list              - Lists all saved connections
  show              - Display the raw contents of the configuration file
  remove <nickname> - Remove a saved connection by nickname`,
	Example: `  # Add a new connection
  truenas_incus_ctl config login 192.168.0.31
  
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
	Use:   "login <hostname>...",
	Short: "Interactively logs into and creates an API key for the specified hostname",
	Args:  cobra.MinimumNArgs(1),
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the contents of the configuration file",
	Args:  cobra.NoArgs,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved connection nicknames",
	Args:  cobra.NoArgs,
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <nickname>",
	Short: "Remove a saved connection by nickname",
	Args:  cobra.ExactArgs(1),
}

func init() {
	configLoginCmd.RunE = WrapCommandFunc(loginToHost)
	configShowCmd.RunE = WrapCommandFunc(showConfig)
	configListCmd.RunE = WrapCommandFunc(listConfigs)
	configRemoveCmd.RunE = WrapCommandFunc(removeConfig)

	configCmd.AddCommand(configLoginCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRemoveCmd)
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

// listConfigs lists all connection nicknames in the config
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

	// Extract and print the nicknames
	hosts, ok := config["hosts"].(map[string]interface{})
	if !ok || len(hosts) == 0 {
		fmt.Println("No connections configured")
		return nil
	}

	fmt.Println("Available connection nicknames:")
	for nickname := range hosts {
		fmt.Printf("  %s\n", nickname)
	}
	
	return nil
}

// removeConfig removes a connection from the config by nickname
func removeConfig(cmd *cobra.Command, api core.Session, args []string) error {
	nickname := args[0]
	
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
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
	}

	// Check if hosts section exists
	hosts, ok := config["hosts"].(map[string]interface{})
	if !ok || len(hosts) == 0 {
		return fmt.Errorf("No connections configured in %s", configPath)
	}

	// Check if the nickname exists
	if _, exists := hosts[nickname]; !exists {
		return fmt.Errorf("Connection with nickname '%s' not found", nickname)
	}

	// Remove the connection
	delete(hosts, nickname)

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize config: %v", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write config to %s: %v", configPath, err)
	}

	fmt.Printf("Connection '%s' has been removed from the configuration\n", nickname)
	return nil
}

// loginToHost implements the login subcommand functionality
func loginToHost(cmd *cobra.Command, api core.Session, args []string) error {
	// Note: 'api' parameter will be nil for this command, which is expected
	hostname := args[0]
	fmt.Printf("Setting up connection to TrueNAS host: %s\n", hostname)

	// Prompt for nickname
	fmt.Print("Enter a nickname for this connection (press Enter to use hostname): ")
	var nickname string
	fmt.Scanln(&nickname)
	if nickname == "" {
		nickname = hostname
		fmt.Printf("Using '%s' as the nickname\n", nickname)
	}

	// Prompt for API key
	fmt.Print("Enter your TrueNAS API key: ")
	var apiKey string
	fmt.Scanln(&apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Construct the WebSocket URL with API endpoint
	url := "wss://" + hostname + "/api/current"
	fmt.Printf("Testing connection to %s...\n", url)

	// Test the connection by creating a temporary client
	client, err := truenas_api.NewClient(url, false)
	if err != nil {
		return fmt.Errorf("Failed to create connection to %s: %v", url, err)
	}

	// Attempt to login to verify API key
	err = client.Login("", "", apiKey)
	if err != nil {
		client.Close()
		return fmt.Errorf("Failed to login to %s: %v", url, err)
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

	// Get the config file path
	configPath := g_configFileName
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Ensure the config directory exists
	configDir := path.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("Failed to create config directory %s: %v", configDir, err)
	}

	// Read existing config or create new config
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If the file doesn't exist, create a new config
		config = make(map[string]interface{})
	} else {
		// Parse existing config
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("Failed to parse config file %s: %v", configPath, err)
		}
	}

	// Ensure hosts section exists
	hosts, ok := config["hosts"].(map[string]interface{})
	if !ok {
		hosts = make(map[string]interface{})
		config["hosts"] = hosts
	}

	// Add or update host entry with URL including API endpoint
	// Store the complete URL with /api/current path under the nickname
	hostConfig := map[string]interface{}{
		"url":     url, // Using the same URL with /api/current path
		"api_key": apiKey,
	}
	hosts[nickname] = hostConfig

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize config: %v", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write config to %s: %v", configPath, err)
	}

	fmt.Printf("Configuration for '%s' (nickname for %s) saved to %s\n", nickname, hostname, configPath)
	return nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"truenas/truenas_incus_ctl/core"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit local configuration settings",
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

func init() {
	configLoginCmd.RunE = WrapCommandFunc(loginToHost)
	configShowCmd.RunE = WrapCommandFunc(showConfig)

	configCmd.AddCommand(configLoginCmd)
	configCmd.AddCommand(configShowCmd)
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

// loginToHost implements the login subcommand functionality
func loginToHost(cmd *cobra.Command, api core.Session, args []string) error {
	// Placeholder for login functionality
	// This will be implemented in a future enhancement
	return fmt.Errorf("Login functionality not yet implemented")
}

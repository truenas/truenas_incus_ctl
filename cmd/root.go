package cmd

import (
	"os"
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
	delete(flags, "key-file")
}

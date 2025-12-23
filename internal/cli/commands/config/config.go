package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigCmd represents the config command group
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long: `View and modify CLI configuration settings.

Configuration is stored in ~/.adsops-utils/config.yaml

Examples:
  # Initialize configuration
  changes config init

  # View current configuration
  changes config view

  # Set a configuration value
  changes config set api_url https://api.example.com

  # Get a configuration value
  changes config get api_url`,
}

func init() {
	ConfigCmd.AddCommand(initCmd)
	ConfigCmd.AddCommand(viewCmd)
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(getCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CLI configuration",
	Long: `Initialize the CLI configuration file.

This will create a default configuration file at
~/.adsops-utils/config.yaml`,
	Run: runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := home + "/.adsops-utils"
	configFile := configDir + "/config.yaml"

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Configuration already exists at %s\n", configFile)
		fmt.Print("Overwrite? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Set default values
	viper.Set("api_url", "https://api.changes.afterdarksys.com")
	viper.Set("output", "table")
	viper.Set("verbose", false)

	// Write config file
	if err := viper.WriteConfigAs(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration initialized at %s\n", configFile)
}

var viewCmd = &cobra.Command{
	Use:     "view",
	Aliases: []string{"show"},
	Short:   "View current configuration",
	Run:     runView,
}

func runView(cmd *cobra.Command, args []string) {
	fmt.Println("Current Configuration")
	fmt.Println("====================")
	fmt.Println()

	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
}

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Examples:
  changes config set api_url https://api.example.com
  changes config set output json
  changes config set verbose true`,
	Args: cobra.ExactArgs(2),
	Run:  runSet,
}

func runSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	viper.Set(key, value)

	if err := viper.WriteConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s\n", key, value)
}

var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run:   runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := viper.Get(key)

	if value == nil {
		fmt.Printf("%s is not set\n", key)
		return
	}

	fmt.Printf("%s = %v\n", key, value)
}

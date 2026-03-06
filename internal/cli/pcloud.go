package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	// BaseURL to pCloud API
	BaseURL string

	// AccessToken for user
	AccessToken string
	cfgFile     string
	verbose     bool
)

// RootCmd declares the main command
var RootCmd = &cobra.Command{
	Use:   "pcloud-cli",
	Short: "pcloud-cli is a command line interface to the pCloud API.",
	Long: `A command line interface to interact with pCloud storage.
More info can be found on github, http://github.com/storvik/pcloud-cli`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

// Execute adds all child commands to the root command
func Execute() {
	BaseURL = strings.TrimSpace(os.Getenv("PCLOUD_BASE_URL"))

	if BaseURL != "" {
		pcloud.SetBaseURL(BaseURL)
	}

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pcloud.json)")
	RootCmd.PersistentFlags().StringVar(&AccessToken, "token", "", "bearer token to access API, can be used when not using config file")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for debugging")

	viper.SetDefault("author", "Petter S. Storvik <petterstorvik@gmail.com>")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	viper.SetConfigName(".pcloud")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if cfgFile != "" { // Use custom config file if --config flag set
		viper.SetConfigFile(cfgFile)
	}

	err := viper.ReadInConfig()
	if err != nil { // No config file found, authorize of token not set
		if AccessToken == "" {
			fmt.Println("Config file not found, run onboard or pass token with --token")
		}
	} else {
		if verbose {
			fmt.Println("Configuration file, " + viper.ConfigFileUsed() + " found")
		}
	}

	if strings.TrimSpace(AccessToken) == "" {
		AccessToken = strings.TrimSpace(viper.GetString("access_token"))
	}

	if strings.TrimSpace(BaseURL) == "" {
		BaseURL = strings.TrimSpace(viper.GetString("base_url"))
	}

	if strings.TrimSpace(BaseURL) != "" {
		pcloud.SetBaseURL(BaseURL)
	}

	pcloud.SetVerbose(verbose)
}

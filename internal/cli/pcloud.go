package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	API *pcloud.API

	accessToken string
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

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pcloud.json)")
	RootCmd.PersistentFlags().StringVar(&accessToken, "token", "", "bearer token to access API, can be used when not using config file")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for debugging")

	viper.SetDefault("author", "Petter S. Storvik <petterstorvik@gmail.com>")
}

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
		if accessToken == "" {
			fmt.Println("Config file not found, run onboard or pass token with --token")
		}
	} else {
		if verbose {
			fmt.Println("Configuration file, " + viper.ConfigFileUsed() + " found")
		}
	}

	if strings.TrimSpace(accessToken) == "" {
		accessToken = strings.TrimSpace(viper.GetString("access_token"))
	}

	baseURL := strings.TrimSpace(viper.GetString("base_url"))

	API = pcloud.NewAPI()
	API.AccessToken = accessToken
	API.BaseURL = baseURL
}

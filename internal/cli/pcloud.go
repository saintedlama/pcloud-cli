package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	// BaseURL to pCloud API
	BaseURL = "https://eapi.pcloud.com"
	// ClientID is pCloud ID of pcloud-cli
	ClientID = "wMJTDKXtja"
	// ClientSecret is secret key needed to identify app
	ClientSecret = "bCS3k9W89t0zL51qpcL2Ck3bjnF7"

	// AccessToken for user
	AccessToken string
	cfgFile     string
	verbose     bool
)

// RootCmd declares the main command
var RootCmd = &cobra.Command{
	Use:   "pCloud-cli",
	Short: "pCloud-cli is a command line interface to the pCloud API.",
	Long: `A command line interface to interact with pCloud storage.
More info can be found on github, http://github.com/storvik/pcloud-cli`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

// Execute adds all child commands to the root command
func Execute(baseurl, clientid, clientsecret string) {
	BaseURL = baseurl
	ClientID = clientid
	ClientSecret = clientsecret
	pcloud.SetBaseURL(BaseURL)

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
			fmt.Println("Config file not found, authorize or pass token with --token")
		}
	} else {
		if verbose {
			fmt.Println("Configuration file, " + viper.ConfigFileUsed() + " found")
		}
		AccessToken = viper.GetString("access_token")
	}

	pcloud.SetVerbose(verbose)
}

package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	API     *pcloud.API
	cfgFile string
	verbose bool
)

// RootCmd declares the main command
var RootCmd = &cobra.Command{
	Use:   "pcloud-cli",
	Short: "pcloud-cli is a command line interface to the pCloud API.",
	Long: `A command line interface to interact with pCloud storage.
More info can be found on github, http://github.com/saintedlama/pcloud-cli`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetString("auth_token") == "" {
			fmt.Println("No configuration found. Starting onboarding...")
			fmt.Println()
			onboard(cmd, args)
			return
		}

		info, err := API.GetUserInfo()
		if err != nil {
			fmt.Println("Could not fetch account info:", err)
			cmd.Help()
			return
		}

		planLabel := "Free"
		switch {
		case info.PremiumLifetime:
			planLabel = "Premium Lifetime"
		case info.Premium:
			planLabel = "Premium"
		}

		usedPct := float64(info.UsedQuota) / float64(info.Quota) * 100

		label := color.New(color.FgHiBlack)
		value := color.New(color.FgCyan, color.Bold)

		var storageColor *color.Color
		switch {
		case usedPct >= 90:
			storageColor = color.New(color.FgRed, color.Bold)
		case usedPct >= 70:
			storageColor = color.New(color.FgYellow, color.Bold)
		default:
			storageColor = color.New(color.FgGreen, color.Bold)
		}

		label.Print("User Information\n")
		label.Print("Account:  ")
		value.Println(info.Email)
		label.Print("Plan:     ")
		value.Println(planLabel)
		label.Print("Storage:  ")
		storageColor.Printf("%s / %s", formatBytes(info.UsedQuota), formatBytes(info.Quota))
		color.New(color.FgHiBlack).Printf(" (%.1f%% used)\n", usedPct)
		fmt.Println()
		cmd.Help()
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pcloud.json)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for debugging")

	viper.SetDefault("author", "Petter S. Storvik <petterstorvik@gmail.com>")
}

func initConfig() {

	viper.SetConfigName(".pcloud")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	err := viper.ReadInConfig()
	if err == nil && verbose {
		fmt.Println("Configuration file, " + viper.ConfigFileUsed() + " found")
	}

	API = pcloud.NewAPI()
	API.AuthToken = viper.GetString("auth_token")
	API.BaseURL = viper.GetString("base_url")
}

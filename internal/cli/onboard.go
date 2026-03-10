package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(onboardCmd)
}

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Set up pcloud-cli for the first time.",
	Long: `Onboard guides you through the initial setup of pcloud-cli.
It will ask for your preferred server region, email, and password,
then save the resulting session credentials to your config file.`,
	Run: onboard,
}

func onboard(cmd *cobra.Command, args []string) {
	fmt.Println("Your username and password are used once to obtain a session token.")
	fmt.Println("Only the token is saved to ~/.pcloud.json — your credentials are never stored.")
	fmt.Println()

	// Check for existing config
	if viper.ConfigFileUsed() != "" {
		fmt.Println("An existing configuration file was found:", viper.ConfigFileUsed())
		confirmPrompt := promptui.Select{
			Label: "Continue and overwrite the existing configuration?",
			Items: []string{"No", "Yes"},
		}
		_, confirm, err := confirmPrompt.Run()
		if err != nil || confirm == "No" {
			fmt.Println("Onboarding cancelled.")
			return
		}
	}

	// Prompt for server region
	regionPrompt := promptui.Select{
		Label: "Server region",
		Items: []string{"Global (US) — api.pcloud.com", "Europe (EU) — eapi.pcloud.com"},
	}
	regionIdx, _, err := regionPrompt.Run()
	if err != nil {
		fmt.Println("Region selection cancelled:", err)
		os.Exit(1)
	}
	baseURLs := []string{"https://api.pcloud.com", "https://eapi.pcloud.com"}
	API.BaseURL = baseURLs[regionIdx]

	// Prompt for credentials
	usernamePrompt := promptui.Prompt{
		Label: "Email",
	}
	username, err := usernamePrompt.Run()
	if err != nil {
		fmt.Println("Input cancelled:", err)
		os.Exit(1)
	}

	passwordPrompt := promptui.Prompt{
		Label: "Password",
		Mask:  '*',
	}
	password, err := passwordPrompt.Run()
	if err != nil {
		fmt.Println("Input cancelled:", err)
		os.Exit(1)
	}

	// Authenticate
	response, _, err := API.LoginWithPassword(username, password)
	if err != nil {
		fmt.Println("Login failed:", err)
		os.Exit(1)
	}
	if response.Auth == "" {
		fmt.Println("Login failed: no auth token returned.")
		os.Exit(1)
	}

	// Write config
	viper.Set("auth_token", response.Auth)
	viper.Set("userid", response.UserID)
	viper.Set("base_url", API.BaseURL)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Could not determine home directory:", err)
		os.Exit(1)
	}
	configPath := filepath.Join(homeDir, ".pcloud.json")
	if err := viper.WriteConfigAs(configPath); err != nil {
		fmt.Println("Failed to write config:", err)
		os.Exit(1)
	}
	fmt.Println("Configuration written to", configPath)
	fmt.Println("Onboarding complete! You can now use pcloud-cli.")
}

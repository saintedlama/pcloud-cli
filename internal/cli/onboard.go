package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/storvik/pcloud-cli/internal/helpers"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

func init() {
	RootCmd.AddCommand(onboardCmd)
}

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Set up pcloud-cli for the first time.",
	Long: `Onboard guides you through the initial setup of pcloud-cli.
It will ask for your preferred server region, open the pCloud authorization
page so you can grant access, and save the resulting credentials to your
config file.`,

	Run: onboard,
}

func onboard(cmd *cobra.Command, args []string) {
	// Check for existing config
	if viper.ConfigFileUsed() != "" {
		fmt.Println("An existing configuration file was found:", viper.ConfigFileUsed())
		if !helpers.AskConfirmation("Continue and overwrite the existing configuration?") {
			fmt.Println("Onboarding cancelled.")
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// Prompt for server region
	var apiBaseURL, tokenEndpoint string
	for {
		fmt.Println()
		fmt.Println("Select server region:")
		fmt.Println("  1) US  (api.pcloud.com)")
		fmt.Println("  2) EU  (eapi.pcloud.com)")
		fmt.Print("Region [1/2]: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			os.Exit(1)
		}
		switch strings.TrimSpace(choice) {
		case "1", "us", "US":
			apiBaseURL = "https://api.pcloud.com"
			tokenEndpoint = "https://api.pcloud.com/oauth2_token"
		case "2", "eu", "EU":
			apiBaseURL = "https://eapi.pcloud.com"
			tokenEndpoint = "https://eapi.pcloud.com/oauth2_token"
		default:
			fmt.Println("Invalid choice, please enter 1 or 2.")
			continue
		}
		break
	}

	// OAuth flow
	authURL := pcloud.OAuthURL()
	fmt.Println()
	fmt.Println("Open the URL below in your browser and authorize pcloud-cli.")
	fmt.Println("After authorization you will be shown a code — paste it here.")
	fmt.Println(authURL)
	helpers.Clipboard.Add(authURL)

	fmt.Print("Code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading code:", err)
		os.Exit(1)
	}

	// Exchange code for access token
	api := pcloud.NewAPI()
	auth, err := api.Authorize(tokenEndpoint, code)
	if err != nil {
		fmt.Println("Authorization failed:", err)
		os.Exit(1)
	}

	// Write config via viper
	viper.Set("access_token", auth.AccessToken)
	viper.Set("userid", auth.UserID)
	viper.Set("base_url", apiBaseURL)

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

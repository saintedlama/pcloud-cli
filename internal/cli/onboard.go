package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
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

type regionOption struct {
	Name     string
	BaseURL  string
	TokenURL string
}

func onboard(cmd *cobra.Command, args []string) {
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
	regions := []regionOption{
		{Name: "US (api.pcloud.com)", BaseURL: "https://api.pcloud.com", TokenURL: "https://api.pcloud.com/oauth2_token"},
		{Name: "EU (eapi.pcloud.com)", BaseURL: "https://eapi.pcloud.com", TokenURL: "https://eapi.pcloud.com/oauth2_token"},
	}
	regionNames := make([]string, len(regions))
	for i, r := range regions {
		regionNames[i] = r.Name
	}
	regionPrompt := promptui.Select{
		Label: "Select server region",
		Items: regionNames,
	}
	idx, _, err := regionPrompt.Run()
	if err != nil {
		fmt.Println("Region selection cancelled:", err)
		os.Exit(1)
	}
	apiBaseURL := regions[idx].BaseURL
	tokenEndpoint := regions[idx].TokenURL

	// OAuth flow
	authURL := pcloud.OAuthURL()
	fmt.Println()
	fmt.Println("Open the URL below in your browser and authorize pcloud-cli.")
	fmt.Println("After authorization you will be shown a code — paste it here.")
	fmt.Println(authURL)
	helpers.Clipboard.Add(authURL)

	codePrompt := promptui.Prompt{
		Label: "Code",
		Validate: func(input string) error {
			if len(input) == 0 {
				return fmt.Errorf("code cannot be empty")
			}
			return nil
		},
	}
	code, err := codePrompt.Run()
	if err != nil {
		fmt.Println("Input cancelled:", err)
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

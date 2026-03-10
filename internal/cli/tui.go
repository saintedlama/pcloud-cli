package cli

import (
	"fmt"
	"os"

	"github.com/saintedlama/pcloud-cli/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open an interactive file browser.",
	Long:  `Open an interactive terminal UI to browse and navigate your pCloud storage.`,
	Run: func(cmd *cobra.Command, args []string) {
		if !API.IsConfigured() {
			fmt.Println("Not configured. Run onboard first.")
			os.Exit(1)
		}
		if err := tui.Run(API); err != nil {
			fmt.Println("TUI error:", err)
			os.Exit(1)
		}
	},
}

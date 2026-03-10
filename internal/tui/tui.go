package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
)

// Run starts the TUI program and blocks until the user quits.
func Run(api *pcloud.API) error {
	p := tea.NewProgram(
		newModel(api),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}

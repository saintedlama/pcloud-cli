package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
)

// Run starts the TUI program and blocks until the user quits.
func Run(api *pcloud.API) error {
	p := tea.NewProgram(newModel(api))
	_, err := p.Run()
	return err
}

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/filebrowser"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// model is the top-level application model.
type model struct {
	browser filebrowser.Model
	width   int
	height  int
}

func newModel(api *pcloud.API) model {
	return model{
		browser: filebrowser.New(api, 80, 24),
	}
}

func (m model) Init() tea.Cmd {
	return m.browser.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.SetSize(msg.Width, msg.Height)
		return m, nil

	case msgs.ShowDialogMsg:
		dm := NewDialogModel(msg.Content, m, m.width, m.height)
		return dm, dm.Init()

	case msgs.CloseDialogMsg:
		// A dialog closed with a result — refresh the current folder.
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.browser.View()
}

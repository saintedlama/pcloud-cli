package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/filebrowser"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/systemd"
)

const (
	viewFiles   = 0
	viewDaemons = 1
)

// model is the top-level application model.
type model struct {
	browser    filebrowser.Model
	sysDaemons systemd.Model
	activeView int
	width      int
	height     int
}

func newModel(api *pcloud.API) model {
	return model{
		browser:    filebrowser.New(api, 80, 24),
		sysDaemons: systemd.New(80, 24),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.browser.Init(), m.sysDaemons.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.SetSize(msg.Width, msg.Height)
		m.sysDaemons = m.sysDaemons.SetSize(msg.Width, msg.Height)
		return m, nil

	case msgs.ShowDialogMsg:
		dm := NewDialogModel(msg.Content, m, m.width, m.height)
		return dm, dm.Init()

	case msgs.CloseDialogMsg:
		if m.activeView == viewDaemons {
			var cmd tea.Cmd
			m.sysDaemons, cmd = m.sysDaemons.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "tab" {
			if m.activeView == viewFiles {
				m.activeView = viewDaemons
			} else {
				m.activeView = viewFiles
			}
			return m, nil
		}
	}

	// Keyboard events go only to the active view.
	if _, isKey := msg.(tea.KeyPressMsg); isKey {
		if m.activeView == viewDaemons {
			var cmd tea.Cmd
			m.sysDaemons, cmd = m.sysDaemons.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd
	}

	// All other messages (background results, spinner ticks, …) go to both
	// models so that background work (e.g. unit list load) completes even
	// while the other tab is visible.
	var cmds []tea.Cmd
	var bCmd, dCmd tea.Cmd
	m.browser, bCmd = m.browser.Update(msg)
	m.sysDaemons, dCmd = m.sysDaemons.Update(msg)
	cmds = append(cmds, bCmd, dCmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	var content string
	if m.activeView == viewDaemons {
		content = m.sysDaemons.View()
	} else {
		content = m.browser.View()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

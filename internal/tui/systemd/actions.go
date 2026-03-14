package systemd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type sysdAction struct {
	label string
	key   string
}

var unitActions = []sysdAction{
	{label: "Start", key: "start"},
	{label: "Stop", key: "stop"},
	{label: "Enable", key: "enable"},
	{label: "Disable", key: "disable"},
	{label: "Change mode", key: "change-mode"},
	{label: "Logs", key: "logs"},
	{label: "Remove", key: "remove"},
}

var (
	actionNormalStyle   = lipgloss.NewStyle().PaddingLeft(2)
	actionSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("62"))
	dangerSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("9"))
)

// ActionsDialog presents an action menu for a selected systemd unit.
type ActionsDialog struct {
	unit   Unit
	cursor int
	width  int
	height int
}

// NewActionsDialog creates an action picker for the given unit.
func NewActionsDialog(unit Unit, width, height int) ActionsDialog {
	return ActionsDialog{unit: unit, width: width, height: height}
}

func (m ActionsDialog) Init() tea.Cmd { return nil }

func (m ActionsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case unitOpDoneMsg:
		result := msg
		return m, func() tea.Msg {
			return msgs.CloseDialogMsg{Result: result}
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(unitActions)-1 {
				m.cursor++
			}
		case "esc":
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		case "enter":
			return m.runSelected()
		}
	}
	return m, nil
}

func (m ActionsDialog) runSelected() (tea.Model, tea.Cmd) {
	a := unitActions[m.cursor]
	switch a.key {
	case "start", "stop", "enable", "disable":
		name, op := m.unit.Name, a.key
		return m, func() tea.Msg {
			return runSystemctlOp(op, name)
		}
	case "change-mode":
		dlg := NewChangeModeDialog(m.unit)
		return m, func() tea.Msg {
			return msgs.ShowDialogMsg{Content: dlg}
		}
	case "logs":
		dlg := NewLogsDialog(m.unit.Name, m.width, m.height)
		return m, func() tea.Msg {
			return msgs.ShowDialogMsg{Content: dlg}
		}
	case "remove":
		dlg := NewRemoveDialog(m.unit, m.width, m.height)
		return m, func() tea.Msg {
			return msgs.ShowDialogMsg{Content: dlg}
		}
	}
	return m, nil
}

func (m ActionsDialog) View() tea.View {
	s := titleStyle.Render("pCloud") + "  "
	s += sectionStyle.Render("Sync Daemon Actions")
	s += "\n\n"
	s += "  Unit: " + dimStyle.Render(m.unit.ShortName()) + "\n\n"

	for i, a := range unitActions {
		label := fmt.Sprintf(" %-10s", a.label)
		if i == m.cursor {
			if a.key == "remove" {
				s += dangerSelectedStyle.Render(label)
			} else {
				s += actionSelectedStyle.Render(label)
			}
		} else {
			s += actionNormalStyle.Render(label)
		}
		s += "\n"
	}

	s += "\n"
	s += helpStyle.Render("  ↑/↓ select  |  enter confirm  |  esc cancel")
	return tea.NewView(s)
}

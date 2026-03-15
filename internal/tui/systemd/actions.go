package systemd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/selector"
)

type sysdAction struct {
	label string
	key   string
	style selector.ItemStyle
}

var unitActions = []sysdAction{
	{label: "Start", key: "start"},
	{label: "Stop", key: "stop"},
	{label: "Enable", key: "enable"},
	{label: "Disable", key: "disable"},
	{label: "Change mode", key: "change-mode"},
	{label: "Logs", key: "logs"},
	{label: "Remove", key: "remove", style: selector.StyleDanger},
}

// ActionsDialog presents an action menu for a selected systemd unit.
type ActionsDialog struct {
	unit   Unit
	list   selector.Selector
	width  int
	height int
}

// NewActionsDialog creates an action picker for the given unit.
// Actions that are not applicable to the unit's current state are disabled.
func NewActionsDialog(unit Unit, width, height int) ActionsDialog {
	isActive := unit.ActiveState == "active"
	isEnabled := unit.EnabledState == "enabled"

	disabled := map[string]bool{
		"start":   isActive,
		"stop":    !isActive,
		"enable":  isEnabled,
		"disable": !isEnabled,
	}

	items := make([]selector.Item, len(unitActions))
	for i, a := range unitActions {
		items[i] = selector.Item{Label: a.label, Key: a.key, Style: a.style, Disabled: disabled[a.key]}
	}
	cursor := 0
	for i, item := range items {
		if !item.Disabled {
			cursor = i
			break
		}
	}
	return ActionsDialog{unit: unit, list: selector.New(items, cursor), width: width, height: height}
}

func (m ActionsDialog) Init() tea.Cmd { return nil }

func (m ActionsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.list, _ = m.list.Update(msg)
	switch msg := msg.(type) {
	case unitOpDoneMsg:
		result := msg
		return m, func() tea.Msg {
			return msgs.CloseDialogMsg{Result: result}
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		case "enter":
			return m.runSelected()
		}
	}
	return m, nil
}

func (m ActionsDialog) runSelected() (tea.Model, tea.Cmd) {
	sel, _ := m.list.Selected()
	switch sel.Key {
	case "start", "stop", "enable", "disable":
		name, op := m.unit.Name, sel.Key
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
	s += "  Unit:      " + dimStyle.Render(m.unit.ShortName()) + "\n"
	s += "  Cloud:     " + dimStyle.Render(m.unit.CloudPath) + "\n"
	s += "  Local:     " + dimStyle.Render(m.unit.LocalPath) + "\n"
	s += "  Interval:  " + dimStyle.Render(m.unit.Interval) + "   Size: " + dimStyle.Render(formatBytes(m.unit.LocalSize)) + "\n\n"

	s += m.list.View()
	s += "\n"
	s += helpStyle.Render("  ↑/↓ select  |  enter confirm  |  esc cancel")
	return tea.NewView(s)
}

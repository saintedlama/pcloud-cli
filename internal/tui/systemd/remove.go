package systemd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type removeState int

const (
	removeConfirm removeState = iota
	removeRunning
	removeDone
)

type removeUnitMsg struct{}

// RemoveDialog asks the user to confirm, then stops, disables, and deletes
// the systemd unit file for the selected sync daemon.
type RemoveDialog struct {
	unit   Unit
	input  textinput.Model
	state  removeState
	err    error
	width  int
	height int
}

// NewRemoveDialog creates a remove confirmation dialog for the given unit.
func NewRemoveDialog(unit Unit, width, height int) *RemoveDialog {
	ti := textinput.New()
	ti.CharLimit = 1
	ti.SetWidth(5)
	ti.Placeholder = "N"
	return &RemoveDialog{
		unit:   unit,
		input:  ti,
		state:  removeConfirm,
		width:  width,
		height: height,
	}
}

func (m *RemoveDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *RemoveDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == removeDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				return msgs.CloseDialogMsg{Result: "removed"}
			}
		}
		return m, nil
	}

	if m.state == removeRunning {
		switch msg := msg.(type) {
		case removeUnitMsg:
			m.state = removeDone
			return m, nil
		case msgs.ErrMsg:
			m.err = msg.Err
			return m, nil
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch kMsg.String() {
		case "enter":
			if strings.EqualFold(strings.TrimSpace(m.input.Value()), "Y") {
				m.input.Blur()
				m.state = removeRunning
				return m, removeUnit(m.unit)
			}
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		case "esc":
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *RemoveDialog) View() tea.View {
	s := titleStyle.Render("pCloud") + "  "
	s += errorStyle.Render("Remove Sync Daemon")
	s += "\n\n"
	s += "  Unit: " + dimStyle.Render(m.unit.ShortName()) + "\n\n"

	if m.state == removeDone {
		s += successStyle.Render("  Removed successfully")
		s += "\n\n"
		s += helpStyle.Render("  Press any key to continue")
		return tea.NewView(s)
	}

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("  Error: %v", m.err))
		s += "\n\n"
		s += helpStyle.Render("  Press any key to continue")
		return tea.NewView(s)
	}

	if m.state == removeRunning {
		s += "  Removing..."
		return tea.NewView(s)
	}

	s += errorStyle.Render("  This will stop, disable, and delete the unit file.") + "\n"
	s += errorStyle.Render("  This action cannot be undone!") + "\n\n"
	s += "  Type Y to confirm: " + m.input.View()
	s += "\n\n"
	s += helpStyle.Render("  Enter to confirm  |  Esc to cancel")
	return tea.NewView(s)
}

func removeUnit(unit Unit) tea.Cmd {
	return func() tea.Msg {
		if out, err := exec.Command( //nolint:gosec
			"systemctl", "--user", "stop", unit.Name,
		).CombinedOutput(); err != nil {
			return msgs.ErrMsg{Err: fmt.Errorf("stop %s: %w\n%s", unit.Name, err, bytes.TrimSpace(out))}
		}
		if out, err := exec.Command( //nolint:gosec
			"systemctl", "--user", "disable", unit.Name,
		).CombinedOutput(); err != nil {
			return msgs.ErrMsg{Err: fmt.Errorf("disable %s: %w\n%s", unit.Name, err, bytes.TrimSpace(out))}
		}
		unitPath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", unit.Name)
		if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
			return msgs.ErrMsg{Err: fmt.Errorf("remove unit file: %w", err)}
		}
		exec.Command("systemctl", "--user", "daemon-reload").Run() //nolint:gosec,errcheck
		return removeUnitMsg{}
	}
}

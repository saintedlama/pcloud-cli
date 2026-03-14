package systemd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type changeModeState int

const (
	changeModeSelect changeModeState = iota
	changeModeApplying
	changeModeDone
)

// changeModeAppliedMsg carries the result of rewriting the unit file.
type changeModeAppliedMsg struct{ err error }

// ChangeModeDialog lets the user pick a new sync mode for an existing unit.
type ChangeModeDialog struct {
	unit    Unit
	cursor  int
	spinner spinner.Model
	state   changeModeState
	err     error
}

// NewChangeModeDialog creates the dialog pre-selected on the unit's current mode.
func NewChangeModeDialog(unit Unit) *ChangeModeDialog {
	cursor := 0
	for i, m := range addSyncModes {
		if m.flag == unit.Mode {
			cursor = i
			break
		}
	}
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	return &ChangeModeDialog{
		unit:    unit,
		cursor:  cursor,
		spinner: s,
		state:   changeModeSelect,
	}
}

func (m *ChangeModeDialog) Init() tea.Cmd { return nil }

func (m *ChangeModeDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == changeModeDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				return msgs.CloseDialogMsg{Result: unitOpDoneMsg{msg: fmt.Sprintf("mode changed to %q", addSyncModes[m.cursor].flag)}}
			}
		}
		return m, nil
	}

	if m.state == changeModeApplying {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		case changeModeAppliedMsg:
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.state = changeModeDone
			}
			return m, nil
		}
		return m, nil
	}

	kMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch kMsg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(addSyncModes)-1 {
			m.cursor++
		}
	case "enter":
		m.state = changeModeApplying
		return m, tea.Batch(
			m.spinner.Tick,
			applyModeChange(m.unit, addSyncModes[m.cursor].flag),
		)
	case "esc":
		return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
	}

	return m, nil
}

func (m *ChangeModeDialog) View() tea.View {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("pCloud") + "  ")
	sb.WriteString(sectionStyle.Render("Change Sync Mode"))
	sb.WriteString("\n\n")
	sb.WriteString("  Unit: ")
	sb.WriteString(dimStyle.Render(m.unit.ShortName()))
	sb.WriteString("\n\n")

	if m.state == changeModeDone {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  Mode changed to %q and service restarted.", addSyncModes[m.cursor].flag)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		return tea.NewView(sb.String())
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		return tea.NewView(sb.String())
	}

	if m.state == changeModeApplying {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString("  Applying new mode…")
		return tea.NewView(sb.String())
	}

	sb.WriteString("  New mode:\n\n")
	for i, sm := range addSyncModes {
		prefix := "    "
		label := sm.label
		if sm.flag == m.unit.Mode {
			label += "  (current)"
		}
		if i == m.cursor {
			sb.WriteString(selectedRowStyle.Render(prefix + "> " + label))
		} else {
			sb.WriteString(prefix + "  " + label)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  ↑/↓ select  |  Enter confirm  |  Esc cancel"))
	return tea.NewView(sb.String())
}

// applyModeChange rewrites the --mode flag in the ExecStart line of the unit
// file, then daemon-reloads and restarts the service.
func applyModeChange(unit Unit, newMode string) tea.Cmd {
	return func() tea.Msg {
		unitPath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", unit.Name)

		data, err := os.ReadFile(unitPath) //nolint:gosec // path built from HOME + known unit name
		if err != nil {
			return changeModeAppliedMsg{err: fmt.Errorf("read unit file: %w", err)}
		}

		var out bytes.Buffer
		sc := bufio.NewScanner(bytes.NewReader(data))
		for sc.Scan() {
			line := sc.Text()
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "ExecStart=") {
				line = rewriteMode(line, newMode)
			}
			out.WriteString(line)
			out.WriteByte('\n')
		}

		if err := os.WriteFile(unitPath, out.Bytes(), 0o644); err != nil { //nolint:gosec
			return changeModeAppliedMsg{err: fmt.Errorf("write unit file: %w", err)}
		}

		if outBytes, err := exec.Command( //nolint:gosec
			"systemctl", "--user", "daemon-reload",
		).CombinedOutput(); err != nil {
			return changeModeAppliedMsg{err: fmt.Errorf("daemon-reload: %w\n%s", err, bytes.TrimSpace(outBytes))}
		}

		if outBytes, err := exec.Command( //nolint:gosec
			"systemctl", "--user", "restart", unit.Name,
		).CombinedOutput(); err != nil {
			return changeModeAppliedMsg{err: fmt.Errorf("restart %s: %w\n%s", unit.Name, err, bytes.TrimSpace(outBytes))}
		}

		return changeModeAppliedMsg{}
	}
}

// rewriteMode replaces the value after --mode in an ExecStart line.
// If --mode is absent it appends it.
func rewriteMode(line, newMode string) string {
	fields := strings.Fields(line)
	for i, f := range fields {
		if f == "--mode" && i+1 < len(fields) {
			fields[i+1] = newMode
			return strings.Join(fields, " ")
		}
	}
	// --mode not found: append it
	return line + " --mode " + newMode
}

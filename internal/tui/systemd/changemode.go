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
	"github.com/saintedlama/pcloud-cli/internal/tui/selector"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
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
	list    selector.Selector
	spinner spinner.Model
	state   changeModeState
	err     error
}

// NewChangeModeDialog creates the dialog pre-selected on the unit's current mode.
func NewChangeModeDialog(unit Unit) *ChangeModeDialog {
	cursor := 0
	modeItems := make([]selector.Item, len(addSyncModes))
	for i, sm := range addSyncModes {
		label := sm.label
		if sm.flag == unit.Mode {
			label += "  (current)"
			cursor = i
		}
		modeItems[i] = selector.Item{Label: label, Key: sm.flag}
	}
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	return &ChangeModeDialog{
		unit:    unit,
		list:    selector.New(modeItems, cursor),
		spinner: s,
		state:   changeModeSelect,
	}
}

func (m *ChangeModeDialog) Init() tea.Cmd { return nil }

func (m *ChangeModeDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == changeModeDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				sel, _ := m.list.Selected()
				return msgs.CloseDialogMsg{Result: unitOpDoneMsg{msg: fmt.Sprintf("mode changed to %q", sel.Key)}}
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

	m.list, _ = m.list.Update(msg)
	switch kMsg.String() {
	case "enter":
		sel, _ := m.list.Selected()
		m.state = changeModeApplying
		return m, tea.Batch(
			m.spinner.Tick,
			applyModeChange(m.unit, sel.Key),
		)
	case "esc":
		return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
	}

	return m, nil
}

func (m *ChangeModeDialog) View() tea.View {
	var sb strings.Builder
	sb.WriteString(tuistyles.Title.Render("pCloud") + "  ")
	sb.WriteString(sectionStyle.Render("Change Sync Mode"))
	sb.WriteString("\n\n")
	sb.WriteString("  Unit: ")
	sb.WriteString(dimStyle.Render(m.unit.ShortName()))
	sb.WriteString("\n\n")

	if m.state == changeModeDone {
		sel, _ := m.list.Selected()
		sb.WriteString(tuistyles.Success.Render(fmt.Sprintf("  Mode changed to %q and service restarted.", sel.Key)))
		sb.WriteString("\n\n")
		sb.WriteString(tuistyles.Help.Render("  Press any key to continue"))
		return tea.NewView(sb.String())
	}

	if m.err != nil {
		sb.WriteString(tuistyles.Error.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
		sb.WriteString(tuistyles.Help.Render("  Press any key to continue"))
		return tea.NewView(sb.String())
	}

	if m.state == changeModeApplying {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString("  Applying new mode…")
		return tea.NewView(sb.String())
	}

	sb.WriteString("  New mode:\n\n")
	sb.WriteString(m.list.View())
	sb.WriteString("\n")
	sb.WriteString(tuistyles.Help.Render("  ↑/↓ select  |  Enter confirm  |  Esc cancel"))
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

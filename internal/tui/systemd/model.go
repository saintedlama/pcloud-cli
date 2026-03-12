// Package systemd provides a TUI component that lists, inspects, and manages
// systemd --user services whose names start with "pcloud-sync-".
package systemd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// Unit represents a single pcloud-sync systemd user service.
type Unit struct {
	Name         string // full unit name, e.g. "pcloud-sync-Music.service"
	ActiveState  string // "active", "inactive", "failed", …
	EnabledState string // "enabled", "disabled", "static", …
	Mode         string // sync direction: "down" or "up"
}

// ShortName strips the common prefix and suffix for compact display.
func (u Unit) ShortName() string {
	s := strings.TrimPrefix(u.Name, "pcloud-sync-")
	s = strings.TrimSuffix(s, ".service")
	return s
}

// unitsLoadedMsg is sent once the unit list has been fetched.
type unitsLoadedMsg struct {
	units []Unit
}

// unitOpDoneMsg carries the result of an enable/disable/start/stop operation.
type unitOpDoneMsg struct {
	err error
	msg string
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).
			Padding(0, 1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11"))

	colHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("244"))

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("62"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Padding(0, 1)
)

func activeStateStyle(state string) lipgloss.Style {
	switch state {
	case "active":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // grey
	}
}

func enabledStateStyle(state string) lipgloss.Style {
	switch state {
	case "enabled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	case "disabled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // grey
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	}
}

func modeStyle(mode string) lipgloss.Style {
	switch mode {
	case "up":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("213")) // magenta
	default: // "down"
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // blue
	}
}

// Model is the systemd daemon manager TUI component.
// It satisfies the same component interface as filebrowser.Model:
// Init/Update/(View string), not a full tea.Model.
type Model struct {
	units     []Unit
	cursor    int
	loading   bool
	spinner   spinner.Model
	statusMsg string
	width     int
	height    int
}

// New constructs a Model that will load units on Init.
func New(width, height int) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	return Model{
		spinner: s,
		loading: true,
		width:   width,
		height:  height,
	}
}

// Init kicks off the initial unit list load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadUnits)
}

// SetSize updates the component dimensions and returns the updated model.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

// Update handles all messages for the systemd component.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case unitsLoadedMsg:
		m.loading = false
		m.units = msg.units
		if m.cursor >= len(m.units) && len(m.units) > 0 {
			m.cursor = len(m.units) - 1
		}
		return m, nil

	case unitOpDoneMsg:
		if msg.err != nil {
			m.statusMsg = errorStyle.Render(msg.err.Error())
		} else {
			m.statusMsg = successStyle.Render(msg.msg)
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadUnits)

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		m.statusMsg = ""
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.units)-1 {
				m.cursor++
			}
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, loadUnits)
		case "e":
			if u := m.selected(); u != nil {
				name := u.Name
				return m, func() tea.Msg { return runSystemctlOp("enable", name) }
			}
		case "d":
			if u := m.selected(); u != nil {
				name := u.Name
				return m, func() tea.Msg { return runSystemctlOp("disable", name) }
			}
		case "s":
			if u := m.selected(); u != nil {
				name := u.Name
				return m, func() tea.Msg { return runSystemctlOp("start", name) }
			}
		case "x":
			if u := m.selected(); u != nil {
				name := u.Name
				return m, func() tea.Msg { return runSystemctlOp("stop", name) }
			}
		case "enter", "l":
			if u := m.selected(); u != nil {
				dlg := NewLogsDialog(u.Name, m.width, m.height)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dlg}
				}
			}
		}
	}
	return m, nil
}

func (m *Model) selected() *Unit {
	if len(m.units) == 0 || m.cursor < 0 || m.cursor >= len(m.units) {
		return nil
	}
	return &m.units[m.cursor]
}

// View renders the systemd manager component as a plain string.
func (m Model) View() string {
	header := titleStyle.Render("pCloud") + "  " + sectionStyle.Render("Sync Daemons")
	header += "\n\n"

	if m.loading {
		return header + "  " + m.spinner.View() + "  Loading..."
	}

	if len(m.units) == 0 {
		body := dimStyle.Render("No pcloud-sync services found.\nInstall one with: pcloud-cli sync systemd <path>")
		foot := helpStyle.Render("r reload  |  tab switch to files  |  q quit")
		return header + body + "\n\n" + foot
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(colHeaderStyle.Render(fmt.Sprintf("  %-36s  %-6s  %-10s  %-10s", "Service", "Mode", "Active", "Enabled")))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width))
	sb.WriteString("\n")

	for i, u := range m.units {
		name := fmt.Sprintf("%-36s", u.ShortName())
		modeStr := modeStyle(u.Mode).Render(fmt.Sprintf("%-6s", u.Mode))
		activeStr := activeStateStyle(u.ActiveState).Render(fmt.Sprintf("%-10s", u.ActiveState))
		enabledStr := enabledStateStyle(u.EnabledState).Render(fmt.Sprintf("%-10s", u.EnabledState))
		row := fmt.Sprintf("  %s  %s  %s  %s", name, modeStr, activeStr, enabledStr)
		if i == m.cursor {
			sb.WriteString(selectedRowStyle.Render(fmt.Sprintf("  %-36s  %-6s  %-10s  %-10s", u.ShortName(), u.Mode, u.ActiveState, u.EnabledState)))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	if m.statusMsg != "" {
		sb.WriteString(m.statusMsg + "\n\n")
	}
	sb.WriteString(helpStyle.Render("↑/↓ navigate  |  e enable  |  d disable  |  s start  |  x stop  |  enter/l logs  |  r reload  |  tab files  |  q quit"))
	return sb.String()
}

// loadUnits queries systemctl --user for all pcloud-sync services.
func loadUnits() tea.Msg {
	fileMap := make(map[string]string) // name -> enabled state
	{
		out, _ := exec.Command( //nolint:gosec
			"systemctl", "--user", "list-unit-files",
			"--type=service", "--plain", "--no-legend",
		).Output()
		sc := bufio.NewScanner(bytes.NewReader(out))
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) >= 2 && strings.HasPrefix(fields[0], "pcloud-sync-") {
				fileMap[fields[0]] = fields[1]
			}
		}
	}

	activeMap := make(map[string]string) // name -> active state
	{
		out, _ := exec.Command( //nolint:gosec
			"systemctl", "--user", "list-units",
			"--type=service", "--all", "--plain", "--no-legend",
		).Output()
		sc := bufio.NewScanner(bytes.NewReader(out))
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			// Fields: UNIT LOAD ACTIVE SUB [DESCRIPTION…]
			if len(fields) >= 3 && strings.HasPrefix(fields[0], "pcloud-sync-") {
				activeMap[fields[0]] = fields[2]
			}
		}
	}

	var units []Unit
	for name, enabled := range fileMap {
		active := activeMap[name]
		if active == "" {
			active = "inactive"
		}
		units = append(units, Unit{
			Name:         name,
			ActiveState:  active,
			EnabledState: enabled,
			Mode:         parseUnitMode(name),
		})
	}
	sort.Slice(units, func(i, j int) bool {
		return units[i].Name < units[j].Name
	})
	return unitsLoadedMsg{units: units}
}

// runSystemctlOp executes a single systemctl --user operation and returns a
// unitOpDoneMsg. Unit names are sourced from systemctl list output, not from
// arbitrary user input.
func runSystemctlOp(op, name string) tea.Msg {
	out, err := exec.Command( //nolint:gosec
		"systemctl", "--user", op, name,
	).CombinedOutput()
	if err != nil {
		return unitOpDoneMsg{
			err: fmt.Errorf("%s %s: %w\n%s", op, name, err, bytes.TrimSpace(out)),
		}
	}
	return unitOpDoneMsg{msg: fmt.Sprintf("%s %s: OK", op, name)}
}

// parseUnitMode reads the systemd unit file for name and returns the sync mode
// by looking for --mode in the ExecStart line. Defaults to "down".
func parseUnitMode(name string) string {
	unitPath := fmt.Sprintf("%s/.config/systemd/user/%s", os.Getenv("HOME"), name)
	data, err := os.ReadFile(unitPath) //nolint:gosec // path built from HOME + known unit name
	if err != nil {
		return "down"
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "ExecStart=") {
			continue
		}
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "--mode" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}
	return "down"
}

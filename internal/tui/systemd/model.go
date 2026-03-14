// Package systemd provides a TUI component that lists, inspects, and manages
// systemd --user services whose names start with "pcloud-sync-".
package systemd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// Unit represents a single pcloud-sync systemd user service.
type Unit struct {
	Name         string // full unit name, e.g. "pcloud-sync-Music.service"
	ActiveState  string // "active", "inactive", "failed", …
	EnabledState string // "enabled", "disabled", "static", …
	Mode         string // sync direction: "down", "up", or "two-way"
	CloudPath    string // remote pCloud directory
	LocalPath    string // absolute local directory
	Interval     string // polling interval, e.g. "1m0s"
	LocalSize    int64  // local directory size in bytes
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
	case "two-way":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	default: // "down"
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // blue
	}
}

// Model is the systemd daemon manager TUI component.
// It satisfies the same component interface as filebrowser.Model:
// Init/Update/(View string), not a full tea.Model.
type Model struct {
	api       pcloud.CloudAPI
	units     []Unit
	cursor    int
	loading   bool
	spinner   spinner.Model
	statusMsg string
	width     int
	height    int
}

// New constructs a Model that will load units on Init.
func New(api pcloud.CloudAPI, width, height int) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	return Model{
		api:     api,
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

	case msgs.CloseDialogMsg:
		if res, ok := msg.Result.(unitOpDoneMsg); ok {
			if res.err != nil {
				m.statusMsg = errorStyle.Render(res.err.Error())
			} else {
				m.statusMsg = successStyle.Render(res.msg)
			}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, loadUnits)
		}
		if msg.Result == "removed" || msg.Result == "added" {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, loadUnits)
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
		case "enter", "a":
			if u := m.selected(); u != nil {
				dlg := NewActionsDialog(*u, m.width, m.height)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dlg}
				}
			}
		case "n":
			dlg := NewAddDaemonDialog(m.api)
			return m, func() tea.Msg {
				return msgs.ShowDialogMsg{Content: dlg}
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
		body := dimStyle.Render("No pcloud-sync services found.\nPress n to add one.")
		foot := helpStyle.Render("n add  |  r reload  |  tab switch to files  |  q quit")
		return header + body + "\n\n" + foot
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(colHeaderStyle.Render(fmt.Sprintf("  %-28s  %-8s  %-10s  %-10s  %-8s  %-10s", "Service", "Mode", "Active", "Enabled", "Interval", "Local size")))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width))
	sb.WriteString("\n")

	for i, u := range m.units {
		name := fmt.Sprintf("%-28s", u.ShortName())
		modeStr := modeStyle(u.Mode).Render(fmt.Sprintf("%-8s", u.Mode))
		activeStr := activeStateStyle(u.ActiveState).Render(fmt.Sprintf("%-10s", u.ActiveState))
		enabledStr := enabledStateStyle(u.EnabledState).Render(fmt.Sprintf("%-10s", u.EnabledState))
		intervalStr := fmt.Sprintf("%-8s", u.Interval)
		sizeStr := fmt.Sprintf("%-10s", formatBytes(u.LocalSize))
		row := fmt.Sprintf("  %s  %s  %s  %s  %s  %s", name, modeStr, activeStr, enabledStr, intervalStr, sizeStr)
		if i == m.cursor {
			sb.WriteString(selectedRowStyle.Render(fmt.Sprintf("  %-28s  %-8s  %-10s  %-10s  %-8s  %-10s", u.ShortName(), u.Mode, u.ActiveState, u.EnabledState, u.Interval, formatBytes(u.LocalSize))))
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	if m.statusMsg != "" {
		sb.WriteString(m.statusMsg + "\n\n")
	}
	sb.WriteString(helpStyle.Render("↑/↓ navigate  |  enter/a actions  |  n add  |  r reload  |  tab files  |  q quit"))
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
		info := parseUnitFile(name)
		units = append(units, Unit{
			Name:         name,
			ActiveState:  active,
			EnabledState: enabled,
			Mode:         info.mode,
			CloudPath:    info.cloudPath,
			LocalPath:    info.localPath,
			Interval:     info.interval,
			LocalSize:    localDirSize(info.localPath),
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

// unitFileInfo holds the values parsed from an ExecStart line in a unit file.
type unitFileInfo struct {
	mode      string
	cloudPath string
	localPath string
	interval  string
}

// parseUnitFile reads the systemd unit file for name and extracts the sync
// parameters from the ExecStart line. It handles double-quoted arguments.
func parseUnitFile(name string) unitFileInfo {
	info := unitFileInfo{mode: "down"}
	unitPath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", name)
	data, err := os.ReadFile(unitPath) //nolint:gosec // path built from HOME + known unit name
	if err != nil {
		return info
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}
		tokens := tokenizeExecStart(strings.TrimPrefix(line, "ExecStart="))
		// tokens: <exec> sync daemon <cloudPath> <localPath> [--interval N] [--mode M]
		for i, t := range tokens {
			if t == "daemon" && i+2 < len(tokens) {
				info.cloudPath = tokens[i+1]
				info.localPath = tokens[i+2]
			}
			if t == "--mode" && i+1 < len(tokens) {
				info.mode = tokens[i+1]
			}
			if t == "--interval" && i+1 < len(tokens) {
				info.interval = tokens[i+1]
			}
		}
		break
	}
	return info
}

// tokenizeExecStart splits an ExecStart value into tokens, respecting
// double-quoted arguments (as produced by the unit template's q function).
func tokenizeExecStart(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && !inQuote:
			inQuote = true
		case c == '"' && inQuote:
			inQuote = false
		case c == ' ' && !inQuote:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// localDirSize returns the total size in bytes of all files under path.
func localDirSize(path string) int64 {
	if path == "" {
		return 0
	}
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// formatBytes formats a byte count as a human-readable string (e.g. "1.2 GB").
func formatBytes(b int64) string {
	if b <= 0 {
		return "—"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

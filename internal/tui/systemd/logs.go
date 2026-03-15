package systemd

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

// logsReadyMsg carries the journal output for a unit.
type logsReadyMsg struct {
	unit    string
	content string
}

// LogsDialog is a tea.Model that renders journal output for a systemd unit
// in a scrollable viewport. It is shown via msgs.ShowDialogMsg and closes with
// msgs.CloseDialogMsg when the user presses esc.
type LogsDialog struct {
	unit    string
	vp      viewport.Model
	spinner spinner.Model
	loading bool
	width   int
	height  int
}

// NewLogsDialog builds a LogsDialog for the given unit name.
// width and height are the current terminal dimensions.
func NewLogsDialog(unit string, width, height int) LogsDialog {
	// Reserve space: border (2×2) + title row (2) + help row (1) + padding (2).
	vpW := width - 8
	vpH := height - 9
	if vpW < 20 {
		vpW = 20
	}
	if vpH < 5 {
		vpH = 5
	}
	vp := viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
	vp.SoftWrap = true

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return LogsDialog{
		unit:    unit,
		vp:      vp,
		spinner: s,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m LogsDialog) Init() tea.Cmd {
	unit := m.unit
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		return fetchLogs(unit)
	})
}

func (m LogsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			unit := m.unit
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return fetchLogs(unit)
			})
		}

	case logsReadyMsg:
		m.loading = false
		m.vp.SetContent(msg.content)
		m.vp.GotoBottom()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m LogsDialog) View() tea.View {
	unitShort := strings.TrimPrefix(m.unit, "pcloud-sync-")
	unitShort = strings.TrimSuffix(unitShort, ".service")
	title := tuistyles.Title.Render("Logs") + "  " + tuistyles.Title.Render(unitShort)

	var body string
	if m.loading {
		body = "  " + m.spinner.View() + "  Loading logs…"
	} else {
		body = m.vp.View()
	}

	scrollInfo := ""
	if !m.loading {
		pct := int(m.vp.ScrollPercent() * 100)
		scrollInfo = fmt.Sprintf("  %d%%", pct)
	}
	help := tuistyles.Help.Render("↑/↓ scroll  |  r refresh  |  esc close") + scrollInfo

	inner := title + "\n\n" + body + "\n\n" + help

	innerW := m.width - 8
	innerH := m.height - 5
	if innerW < 20 {
		innerW = 20
	}
	if innerH < 8 {
		innerH = 8
	}

	lines := strings.Split(inner, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	inner = strings.Join(lines[:innerH], "\n")

	bordered := tuistyles.DialogBorder.Width(innerW).Height(innerH).Render(inner)
	return tea.NewView(bordered)
}

// fetchLogs runs journalctl --user for the given unit and returns the output.
// The unit name originates from systemctl list output, not from user input.
func fetchLogs(unit string) tea.Msg {
	out, err := exec.Command( //nolint:gosec
		"journalctl", "--user", "-u", unit, "-n", "300", "--no-pager",
	).Output()
	if err != nil || len(out) == 0 {
		return logsReadyMsg{
			unit:    unit,
			content: fmt.Sprintf("No logs found for %s.\n(The unit may not have run yet.)", unit),
		}
	}
	return logsReadyMsg{unit: unit, content: string(out)}
}

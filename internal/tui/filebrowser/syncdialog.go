package filebrowser

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type syncDialogState int

const (
	syncInput syncDialogState = iota
	syncRunning
	syncDone
)

// syncDoneMsg carries the result of the systemd unit installation.
type syncDoneMsg struct {
	unitName string
	err      error
}

// SyncDialog prompts for a local directory and installs a systemd user service
// that continuously syncs the selected pCloud folder to that directory.
type SyncDialog struct {
	cloudPath string
	input     textinput.Model
	spinner   spinner.Model
	state     syncDialogState
	unitName  string
	err       error
}

// NewSyncDialog creates a sync setup dialog for the given folder path.
func NewSyncDialog(cloudPath string) SyncDialog {
	base := filepath.Base(strings.TrimRight(filepath.ToSlash(cloudPath), "/"))
	if base == "" || base == "." || base == "/" {
		base = "pcloud-sync"
	}

	ti := textinput.New()
	ti.CharLimit = 255
	ti.SetWidth(40)
	ti.Placeholder = "local directory"
	ti.SetValue(base)

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return SyncDialog{
		cloudPath: cloudPath,
		input:     ti,
		spinner:   s,
		state:     syncInput,
	}
}

func (m SyncDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m SyncDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == syncDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		}
		return m, nil
	}

	if m.state == syncRunning {
		switch msg := msg.(type) {
		case syncDoneMsg:
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.state = syncDone
				m.unitName = msg.unitName
			}
			return m, nil
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		if kMsg.String() == "enter" {
			localDir := strings.TrimSpace(m.input.Value())
			if localDir == "" {
				return m, nil
			}
			m.input.Blur()
			m.state = syncRunning
			cloudPath := m.cloudPath
			return m, tea.Batch(m.spinner.Tick, installSyncUnit(cloudPath, localDir))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SyncDialog) View() tea.View {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("pCloud") + "  ")
	sb.WriteString(dialogTitleStyle.Render("Create Sync Daemon"))
	sb.WriteString("\n\n")
	sb.WriteString("  Cloud path:  ")
	sb.WriteString(pathStyle.Render(m.cloudPath))
	sb.WriteString("\n\n")

	if m.state == syncDone {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  Service %s enabled and started.", m.unitName)))
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

	if m.state == syncRunning {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString("  Installing systemd unit…")
		return tea.NewView(sb.String())
	}

	sb.WriteString("  Local dir:   ")
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("  Enter to confirm  |  Esc to cancel"))
	return tea.NewView(sb.String())
}

const syncUnitTemplate = `[Unit]
Description=pCloud sync {{q .CloudPath}} to {{q .LocalDir}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{q .ExecPath}} sync daemon {{q .CloudPath}} {{q .LocalDir}} --interval {{.Interval}}
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`

type syncUnitVars struct {
	CloudPath string
	LocalDir  string
	ExecPath  string
	Interval  string
}

// installSyncUnit creates the systemd user unit file and enables+starts the service.
func installSyncUnit(cloudPath, localDir string) tea.Cmd {
	return func() tea.Msg {
		execPath, err := os.Executable()
		if err != nil {
			return syncDoneMsg{err: fmt.Errorf("could not determine executable path: %w", err)}
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return syncDoneMsg{err: fmt.Errorf("could not resolve executable symlinks: %w", err)}
		}

		absLocal, err := filepath.Abs(localDir)
		if err != nil {
			return syncDoneMsg{err: fmt.Errorf("could not resolve local dir: %w", err)}
		}

		sanitized := strings.NewReplacer("/", "-", " ", "_").Replace(strings.Trim(cloudPath, "/"))
		unitName := "pcloud-sync-" + sanitized + ".service"

		unitDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
		unitPath := filepath.Join(unitDir, unitName)

		quoteFn := func(s string) string {
			if strings.ContainsAny(s, " \t") {
				return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
			}
			return s
		}
		tmpl := template.Must(template.New("unit").Funcs(template.FuncMap{"q": quoteFn}).Parse(syncUnitTemplate))

		vars := syncUnitVars{
			CloudPath: cloudPath,
			LocalDir:  absLocal,
			ExecPath:  execPath,
			Interval:  (60 * time.Second).String(),
		}

		if err := os.MkdirAll(unitDir, 0o755); err != nil {
			return syncDoneMsg{err: fmt.Errorf("could not create systemd user directory: %w", err)}
		}

		f, err := os.Create(unitPath)
		if err != nil {
			return syncDoneMsg{err: fmt.Errorf("could not write unit file: %w", err)}
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			f.Close()
			return syncDoneMsg{err: fmt.Errorf("could not render unit template: %w", err)}
		}
		if _, err := f.Write(buf.Bytes()); err != nil {
			f.Close()
			return syncDoneMsg{err: fmt.Errorf("could not write unit file: %w", err)}
		}
		f.Close()

		for _, scArgs := range [][]string{
			{"--user", "daemon-reload"},
			{"--user", "enable", "--now", unitName},
		} {
			out, err := exec.Command("systemctl", scArgs...).CombinedOutput() //nolint:gosec
			if err != nil {
				return syncDoneMsg{err: fmt.Errorf("systemctl %s: %w\n%s",
					strings.Join(scArgs, " "), err, bytes.TrimSpace(out))}
			}
		}

		return syncDoneMsg{unitName: unitName}
	}
}

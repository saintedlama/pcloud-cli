package systemd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/selector"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

type addState int

const (
	addModeSelect addState = iota
	addCloudInput
	addLocalInput
	addValidating
	addInstalling
	addDone
)

type addSyncMode struct {
	label string
	flag  string
}

var addSyncModes = []addSyncMode{
	{label: "down     (pCloud → local, polling)", flag: "down"},
	{label: "up       (local → pCloud, polling)", flag: "up"},
	{label: "two-way  (pCloud ↔ local, fs-events)", flag: "two-way"},
}

// addDoneMsg carries the result of the install step.
type addDoneMsg struct {
	unitName string
	err      error
}

// addValidatedMsg is sent after path validation completes successfully.
type addValidatedMsg struct{}

// addValidationErrMsg is sent when validation fails.
type addValidationErrMsg struct{ err error }

// AddDaemonDialog is a multi-step wizard for creating a new pcloud-sync systemd
// service. Steps: mode → pCloud path → local path → validate → install.
type AddDaemonDialog struct {
	api        pcloud.CloudAPI
	modeList   selector.Selector
	cloudInput textinput.Model
	localInput textinput.Model
	spinner    spinner.Model
	state      addState
	unitName   string
	err        error
}

// NewAddDaemonDialog creates the wizard dialog. api is used for pCloud path
// validation; it may be nil (validation is skipped).
func NewAddDaemonDialog(api pcloud.CloudAPI) *AddDaemonDialog {
	ci := textinput.New()
	ci.CharLimit = 512
	ci.SetWidth(40)
	ci.Placeholder = "/path/to/cloud/folder"

	li := textinput.New()
	li.CharLimit = 512
	li.SetWidth(40)
	li.Placeholder = "dir or /path/to/my/dir"

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	modeItems := make([]selector.Item, len(addSyncModes))
	for i, sm := range addSyncModes {
		modeItems[i] = selector.Item{Label: sm.label, Key: sm.flag}
	}

	return &AddDaemonDialog{
		api:        api,
		modeList:   selector.New(modeItems, 0),
		cloudInput: ci,
		localInput: li,
		spinner:    s,
		state:      addModeSelect,
	}
}

func (m *AddDaemonDialog) Init() tea.Cmd { return nil }

func (m *AddDaemonDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Done or error: any key closes.
	if m.state == addDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg { return msgs.CloseDialogMsg{Result: "added"} }
		}
		return m, nil
	}

	// Spinner ticks during async steps.
	if m.state == addValidating || m.state == addInstalling {
		switch msg := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		case addValidatedMsg:
			// Validation passed → start installing.
			sel, _ := m.modeList.Selected()
			m.state = addInstalling
			return m, tea.Batch(
				m.spinner.Tick,
				installUnit(m.cloudInput.Value(), m.localInput.Value(), sel.Key),
			)
		case addValidationErrMsg:
			m.err = msg.err
			return m, nil
		case addDoneMsg:
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.state = addDone
				m.unitName = msg.unitName
			}
			return m, nil
		}
		return m, nil
	}

	kMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		// Route input events to the active text field.
		var cmd tea.Cmd
		switch m.state {
		case addCloudInput:
			m.cloudInput, cmd = m.cloudInput.Update(msg)
		case addLocalInput:
			m.localInput, cmd = m.localInput.Update(msg)
		}
		return m, cmd
	}

	switch m.state {
	case addModeSelect:
		m.modeList, _ = m.modeList.Update(msg)
		switch kMsg.String() {
		case "enter":
			m.state = addCloudInput
			return m, m.cloudInput.Focus()
		}

	case addCloudInput:
		switch kMsg.String() {
		case "enter":
			if strings.TrimSpace(m.cloudInput.Value()) == "" {
				return m, nil
			}
			m.cloudInput.Blur()
			// Pre-fill the local path with the basename of the cloud path when
			// the user hasn't typed anything yet.
			if strings.TrimSpace(m.localInput.Value()) == "" {
				base := path.Base(strings.TrimRight(m.cloudInput.Value(), "/"))
				if base != "" && base != "." && base != "/" {
					m.localInput.SetValue(base)
				}
			}
			m.state = addLocalInput
			return m, m.localInput.Focus()
		default:
			var cmd tea.Cmd
			m.cloudInput, cmd = m.cloudInput.Update(msg)
			return m, cmd
		}

	case addLocalInput:
		switch kMsg.String() {
		case "enter":
			if strings.TrimSpace(m.localInput.Value()) == "" {
				return m, nil
			}
			m.localInput.Blur()
			m.state = addValidating
			return m, tea.Batch(
				m.spinner.Tick,
				m.validatePaths(),
			)
		default:
			var cmd tea.Cmd
			m.localInput, cmd = m.localInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// validatePaths returns a Cmd that checks:
//   - "down":     pCloud path is accessible via ListFolder
//   - "up":       local path exists on disk
//   - "two-way":  at least one path is accessible (either suffices)
func (m *AddDaemonDialog) validatePaths() tea.Cmd {
	api := m.api
	cloudPath := strings.TrimSpace(m.cloudInput.Value())
	localPath := strings.TrimSpace(m.localInput.Value())
	sel, _ := m.modeList.Selected()
	modeFlag := sel.Key

	return func() tea.Msg {
		localExists := func() bool {
			exp, err := filepath.Abs(localPath)
			if err != nil {
				return false
			}
			_, err = os.Stat(exp)
			return err == nil
		}
		cloudExists := func() bool {
			if api == nil {
				return true // no API, skip check
			}
			_, err := api.ListFolder(cloudPath, pcloud.ListFolderOptions{})
			return err == nil
		}

		switch modeFlag {
		case "down":
			if !cloudExists() {
				return addValidationErrMsg{err: fmt.Errorf("pCloud path %q does not exist or is not accessible", cloudPath)}
			}
		case "up":
			if !localExists() {
				return addValidationErrMsg{err: fmt.Errorf("local path %q does not exist", localPath)}
			}
		case "two-way":
			if !cloudExists() && !localExists() {
				return addValidationErrMsg{err: fmt.Errorf("neither pCloud path %q nor local path %q exists", cloudPath, localPath)}
			}
		}
		return addValidatedMsg{}
	}
}

func (m *AddDaemonDialog) View() tea.View {
	var sb strings.Builder
	sb.WriteString(tuistyles.Title.Render("pCloud") + "  ")
	sb.WriteString(sectionStyle.Render("Add Sync Daemon"))
	sb.WriteString("\n\n")

	if m.state == addDone {
		sb.WriteString(tuistyles.Success.Render(fmt.Sprintf("  Service %s enabled and started.", m.unitName)))
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

	if m.state == addValidating {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString("  Validating paths…")
		return tea.NewView(sb.String())
	}

	if m.state == addInstalling {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString("  Installing systemd unit…")
		return tea.NewView(sb.String())
	}

	if m.state == addModeSelect {
		sb.WriteString("  Sync mode:\n\n")
		sb.WriteString(m.modeList.View())
		sb.WriteString("\n")
		sb.WriteString(tuistyles.Help.Render("  ↑/↓ select  |  Enter confirm  |  Esc cancel"))
		return tea.NewView(sb.String())
	}

	// Show confirmed mode on path-input screens.
	confirmed, _ := m.modeList.Selected()
	sb.WriteString("  Mode:        ")
	sb.WriteString(dimStyle.Render(confirmed.Label))
	sb.WriteString("\n\n")

	if m.state == addCloudInput {
		sb.WriteString("  pCloud path: ")
		sb.WriteString(m.cloudInput.View())
		sb.WriteString("\n\n")
		sb.WriteString(tuistyles.Help.Render("  Enter confirm  |  Esc cancel"))
		return tea.NewView(sb.String())
	}

	// addLocalInput
	sb.WriteString("  pCloud path: ")
	sb.WriteString(dimStyle.Render(strings.TrimSpace(m.cloudInput.Value())))
	sb.WriteString("\n")
	sb.WriteString("  Local path:  ")
	sb.WriteString(m.localInput.View())
	sb.WriteString("\n\n")
	cwd, err := os.Getwd()
	if err == nil {
		sb.WriteString(tuistyles.Help.Render("  Relative paths are resolved from the current directory: " + cwd))
		sb.WriteString("\n")
	}
	sb.WriteString(tuistyles.Help.Render("  Enter confirm  |  Esc cancel"))
	return tea.NewView(sb.String())
}

const addUnitTemplate = `[Unit]
Description=pCloud sync {{q .CloudPath}} to {{q .LocalDir}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{q .ExecPath}} sync daemon {{q .CloudPath}} {{q .LocalDir}} --interval {{.Interval}} --mode {{.Mode}}
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`

type addUnitVars struct {
	CloudPath string
	LocalDir  string
	ExecPath  string
	Interval  string
	Mode      string
}

// installUnit writes the systemd unit file and enables+starts the service.
func installUnit(cloudPath, localDir, modeFlag string) tea.Cmd {
	return func() tea.Msg {
		execPath, err := os.Executable()
		if err != nil {
			return addDoneMsg{err: fmt.Errorf("could not determine executable path: %w", err)}
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return addDoneMsg{err: fmt.Errorf("could not resolve executable symlinks: %w", err)}
		}

		absLocal, err := filepath.Abs(strings.TrimSpace(localDir))
		if err != nil {
			return addDoneMsg{err: fmt.Errorf("could not resolve local dir: %w", err)}
		}

		cloudPath = strings.TrimSpace(cloudPath)
		sanitized := strings.NewReplacer("/", "-", " ", "_").Replace(strings.Trim(cloudPath, "/"))
		if sanitized == "" {
			sanitized = "default"
		}
		unitName := "pcloud-sync-" + sanitized + ".service"

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return addDoneMsg{err: fmt.Errorf("could not determine home directory: %w", err)}
		}
		unitDir := filepath.Join(homeDir, ".config", "systemd", "user")
		unitPath := filepath.Join(unitDir, unitName)

		quoteFn := func(s string) string {
			if strings.ContainsAny(s, " \t") {
				return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
			}
			return s
		}
		tmpl := template.Must(template.New("unit").Funcs(template.FuncMap{"q": quoteFn}).Parse(addUnitTemplate))

		vars := addUnitVars{
			CloudPath: cloudPath,
			LocalDir:  absLocal,
			ExecPath:  execPath,
			Interval:  (60 * time.Second).String(),
			Mode:      modeFlag,
		}

		if err := os.MkdirAll(unitDir, 0o755); err != nil {
			return addDoneMsg{err: fmt.Errorf("could not create systemd user directory: %w", err)}
		}

		f, err := os.Create(unitPath)
		if err != nil {
			return addDoneMsg{err: fmt.Errorf("could not write unit file: %w", err)}
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			f.Close()
			return addDoneMsg{err: fmt.Errorf("could not render unit template: %w", err)}
		}
		if _, err := f.Write(buf.Bytes()); err != nil {
			f.Close()
			return addDoneMsg{err: fmt.Errorf("could not write unit file: %w", err)}
		}
		f.Close()

		for _, scArgs := range [][]string{
			{"--user", "daemon-reload"},
			{"--user", "enable", "--now", unitName},
		} {
			out, err := exec.Command("systemctl", scArgs...).CombinedOutput() //nolint:gosec
			if err != nil {
				return addDoneMsg{err: fmt.Errorf("systemctl %s: %w\n%s",
					strings.Join(scArgs, " "), err, bytes.TrimSpace(out))}
			}
		}

		return addDoneMsg{unitName: unitName}
	}
}

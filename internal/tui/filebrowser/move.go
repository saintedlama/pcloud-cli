package filebrowser

import (
	"fmt"
	"path"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

type moveState int

const (
	moveInput moveState = iota
	moveRunning
	moveDone
)

// MoveDialog lets the user move a file to a different folder.
type MoveDialog struct {
	input textinput.Model
	api   pcloud.CloudAPI
	entry msgs.Entry
	state moveState
	err   error
}

// NewMoveDialog creates a move dialog for the given file entry.
func NewMoveDialog(api pcloud.CloudAPI, entry msgs.Entry) *MoveDialog {
	ti := textinput.New()
	ti.CharLimit = 500
	ti.SetWidth(40)
	ti.Placeholder = "/destination/folder"
	ti.SetValue(path.Dir(entry.Path))

	return &MoveDialog{
		input: ti,
		api:   api,
		entry: entry,
		state: moveInput,
	}
}

type moveFileMsg struct{}

func (m *MoveDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *MoveDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == moveDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				return msgs.CloseDialogMsg{Result: "moved"}
			}
		}
		return m, nil
	}

	if m.state == moveRunning {
		switch msg := msg.(type) {
		case moveFileMsg:
			m.state = moveDone
			return m, nil
		case msgs.ErrMsg:
			m.err = msg.Err
			return m, nil
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		if kMsg.String() == "enter" {
			destFolder := m.input.Value()
			if destFolder == "" {
				return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
			}
			toPath := path.Join(destFolder, m.entry.Name)
			if toPath == m.entry.Path {
				return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
			}
			m.input.Blur()
			m.state = moveRunning
			return m, moveFile(m.api, m.entry, toPath)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *MoveDialog) View() tea.View {
	s := tuistyles.Title.Render("pCloud") + "  "
	s += tuistyles.DialogTitle.Render("Move File")
	s += "\n\n"
	kind := "File"
	if m.entry.IsFolder {
		kind = "Folder"
	}
	s += "  " + kind + ": " + tuistyles.Path.Render(m.entry.Path) + "\n\n"

	if m.state == moveDone {
		s += tuistyles.Success.Render("  Moved successfully")
		s += "\n\n"
		s += tuistyles.Help.Render("  Press any key to continue")
		return tea.NewView(s)
	}

	if m.err != nil {
		s += tuistyles.Error.Render(fmt.Sprintf("  Error: %v", m.err))
		s += "\n\n"
		s += tuistyles.Help.Render("  Press any key to continue")
		return tea.NewView(s)
	}

	if m.state == moveRunning {
		s += "  Moving..."
		return tea.NewView(s)
	}

	s += "  Move to: " + m.input.View()
	s += "\n\n"
	s += tuistyles.Help.Render("  Enter to confirm  |  Esc to cancel")
	return tea.NewView(s)
}

func moveFile(api pcloud.CloudAPI, entry msgs.Entry, toPath string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if entry.IsFolder {
			_, err = api.RenameFolder(entry.Path, toPath)
		} else {
			_, err = api.RenameFile(entry.Path, toPath)
		}
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		return moveFileMsg{}
	}
}

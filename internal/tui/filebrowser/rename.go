package filebrowser

import (
	"fmt"
	"path"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type renameState int

const (
	renameInput renameState = iota
	renameRunning
	renameDone
)

// RenameDialog lets the user rename a file.
type RenameDialog struct {
	input textinput.Model
	api   *pcloud.API
	entry msgs.Entry
	state renameState
	err   error
}

// NewRenameDialog creates a rename dialog for the given file entry.
func NewRenameDialog(api *pcloud.API, entry msgs.Entry) RenameDialog {
	ti := textinput.New()
	ti.CharLimit = 255
	ti.SetWidth(40)
	ti.Placeholder = "new name"
	ti.SetValue(entry.Name)

	return RenameDialog{
		input: ti,
		api:   api,
		entry: entry,
		state: renameInput,
	}
}

type renameFileMsg struct{}

func (m RenameDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m RenameDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == renameDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				return msgs.CloseDialogMsg{Result: "renamed"}
			}
		}
		return m, nil
	}

	if m.state == renameRunning {
		switch msg := msg.(type) {
		case renameFileMsg:
			m.state = renameDone
			return m, nil
		case msgs.ErrMsg:
			m.err = msg.Err
			return m, nil
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		if kMsg.String() == "enter" {
			newName := m.input.Value()
			if newName == "" || newName == m.entry.Name {
				return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
			}
			m.input.Blur()
			m.state = renameRunning
			dir := path.Dir(m.entry.Path)
			toPath := path.Join(dir, newName)
			return m, renameFile(m.api, m.entry, toPath)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m RenameDialog) View() tea.View {
	s := titleStyle.Render("pCloud") + "  "
	s += dialogTitleStyle.Render("Rename File")
	s += "\n\n"
	kind := "File"
	if m.entry.IsFolder {
		kind = "Folder"
	}
	s += "  " + kind + ": " + pathStyle.Render(m.entry.Path) + "\n\n"

	if m.state == renameDone {
		s += successStyle.Render("  Renamed successfully")
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

	if m.state == renameRunning {
		s += "  Renaming..."
		return tea.NewView(s)
	}

	s += "  New name: " + m.input.View()
	s += "\n\n"
	s += helpStyle.Render("  Enter to confirm  |  Esc to cancel")
	return tea.NewView(s)
}

func renameFile(api *pcloud.API, entry msgs.Entry, toPath string) tea.Cmd {
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
		return renameFileMsg{}
	}
}

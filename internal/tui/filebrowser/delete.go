package filebrowser

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type deleteState int

const (
	deleteConfirm deleteState = iota
	deleteRunning
	deleteDone
)

// DeleteDialog asks for confirmation before deleting a file.
type DeleteDialog struct {
	input   textinput.Model
	initCmd tea.Cmd
	api     *pcloud.API
	entry   msgs.Entry
	state   deleteState
	err     error
}

// NewDeleteDialog creates a delete confirmation dialog.
func NewDeleteDialog(api *pcloud.API, entry msgs.Entry) DeleteDialog {
	ti := textinput.New()
	ti.CharLimit = 1
	ti.SetWidth(5)
	ti.Placeholder = "N"
	// Focus() has a pointer receiver; calling it here (while ti is addressable)
	// sets ti.focus = true on the stored value.
	initCmd := ti.Focus()
	return DeleteDialog{
		input:   ti,
		initCmd: initCmd,
		api:     api,
		entry:   entry,
		state:   deleteConfirm,
	}
}

type deleteFileMsg struct{}

func (m DeleteDialog) Init() tea.Cmd {
	return m.initCmd
}

func (m DeleteDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == deleteDone || m.err != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, func() tea.Msg {
				return msgs.CloseDialogMsg{Result: "deleted"}
			}
		}
		return m, nil
	}

	if m.state == deleteRunning {
		switch msg := msg.(type) {
		case deleteFileMsg:
			m.state = deleteDone
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
			val := strings.TrimSpace(m.input.Value())
			if strings.EqualFold(val, "Y") {
				m.input.Blur()
				m.state = deleteRunning
				return m, deleteFile(m.api, m.entry)
			}
			// Any other value dismisses without deleting.
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m DeleteDialog) View() tea.View {
	s := titleStyle.Render("pCloud") + "  "
	s += errorStyle.Render("Delete File")
	s += "\n\n"
	kind := "File"
	if m.entry.IsFolder {
		kind = "Folder"
	}
	s += "  " + kind + ": " + pathStyle.Render(m.entry.Path) + "\n\n"

	if m.state == deleteDone {
		s += successStyle.Render("  Deleted successfully")
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

	if m.state == deleteRunning {
		s += "  Deleting..."
		return tea.NewView(s)
	}

	s += errorStyle.Render("  This action cannot be undone!") + "\n\n"
	s += "  Type Y to confirm: " + m.input.View()
	s += "\n\n"
	s += helpStyle.Render("  Enter to confirm  |  Esc to cancel")
	return tea.NewView(s)
}

func deleteFile(api *pcloud.API, entry msgs.Entry) tea.Cmd {
	return func() tea.Msg {
		var err error
		if entry.IsFolder {
			_, err = api.DeleteFolderRecursive(entry.Path)
		} else {
			_, err = api.DeleteFile(entry.Path)
		}
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		return deleteFileMsg{}
	}
}

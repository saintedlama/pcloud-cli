package filebrowser

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

type action struct {
	label string
	key   string
}

var fileActions = []action{
	{label: "Preview", key: "preview"},
	{label: "Download", key: "download"},
	{label: "Rename", key: "rename"},
	{label: "Move", key: "move"},
	{label: "Delete", key: "rm"},
}

var folderActions = []action{
	{label: "Open", key: "open"},
	{label: "Download", key: "download"},
	{label: "Rename", key: "rename"},
	{label: "Move", key: "move"},
	{label: "Delete", key: "rm"},
}

// ActionsDialog lets the user pick an action to perform on a file or folder.
type ActionsDialog struct {
	api     *pcloud.API
	entry   msgs.Entry
	actions []action
	cursor  int
	width   int
	height  int
}

// NewActionsDialog creates an action picker for the given entry.
// Folder-specific actions are shown when entry.IsFolder is true.
// width and height are the current terminal dimensions, passed through to
// sub-dialogs (e.g. the preview dialog) that need them at construction time.
func NewActionsDialog(api *pcloud.API, entry msgs.Entry, width, height int) ActionsDialog {
	actions := fileActions
	if entry.IsFolder {
		actions = folderActions
	}
	return ActionsDialog{
		api:     api,
		entry:   entry,
		actions: actions,
		width:   width,
		height:  height,
	}
}

func (m ActionsDialog) Init() tea.Cmd {
	return nil
}

func (m ActionsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch kMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case "enter":
			selected := m.actions[m.cursor]
			switch selected.key {
			case "preview":
				dialog := NewPreviewDialog(m.api, m.entry, m.width, m.height)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
				}
			case "open":
				path := m.entry.Path
				return m, func() tea.Msg {
					return msgs.CloseDialogMsg{Result: msgs.NavigateFolderResult{Path: path}}
				}
			case "download":
				if m.entry.IsFolder {
					dialog := NewFolderDownloadDialog(m.api, m.entry)
					return m, func() tea.Msg {
						return msgs.ShowDialogMsg{Content: dialog}
					}
				}
				dialog := NewDownloadDialog(m.api, m.entry)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
				}
			case "rename":
				dialog := NewRenameDialog(m.api, m.entry)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
				}
			case "move":
				dialog := NewMoveDialog(m.api, m.entry)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
				}
			case "rm":
				dialog := NewDeleteDialog(m.api, m.entry)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
				}
			}
		}
	}
	return m, nil
}

var (
	actionNormalStyle   = lipgloss.NewStyle().PaddingLeft(2)
	actionSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("62"))
	dangerSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("9"))
)

func (m ActionsDialog) View() tea.View {
	s := titleStyle.Render("pCloud") + "  "
	s += dialogTitleStyle.Render("Actions")
	s += "\n\n"
	kind := "File"
	if m.entry.IsFolder {
		kind = "Folder"
	}
	s += "  " + kind + ": " + pathStyle.Render(m.entry.Path) + "\n\n"

	for i, a := range m.actions {
		label := fmt.Sprintf(" %-10s", a.label)
		if i == m.cursor {
			if a.key == "rm" {
				s += dangerSelectedStyle.Render(label)
			} else {
				s += actionSelectedStyle.Render(label)
			}
		} else {
			s += actionNormalStyle.Render(label)
		}
		s += "\n"
	}

	s += "\n"
	s += helpStyle.Render("  up/down select  |  enter confirm  |  esc cancel")
	return tea.NewView(s)
}

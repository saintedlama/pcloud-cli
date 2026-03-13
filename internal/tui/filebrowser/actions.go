package filebrowser

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/preview"
)

type action struct {
	label    string
	key      string
	disabled bool
}

var folderActions = []action{
	{label: "Open", key: "open"},
	{label: "Download", key: "download"},
	{label: "Rename", key: "rename"},
	{label: "Move", key: "move"},
	{label: "Delete", key: "rm"},
	{label: "Sync", key: "sync"},
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
	var actions []action
	if entry.IsFolder {
		actions = folderActions
	} else {
		actions = []action{
			{label: "Preview", key: "preview", disabled: preview.GetPreviewType(entry.Name) == preview.PreviewUnsupported},
			{label: "Download", key: "download"},
			{label: "Rename", key: "rename"},
			{label: "Move", key: "move"},
			{label: "Delete", key: "rm"},
		}
	}
	cursor := 0
	for i, a := range actions {
		if !a.disabled {
			cursor = i
			break
		}
	}
	return ActionsDialog{
		api:     api,
		entry:   entry,
		actions: actions,
		cursor:  cursor,
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
			for prev := m.cursor - 1; prev >= 0; prev-- {
				if !m.actions[prev].disabled {
					m.cursor = prev
					break
				}
			}
		case "down", "j":
			for next := m.cursor + 1; next < len(m.actions); next++ {
				if !m.actions[next].disabled {
					m.cursor = next
					break
				}
			}
		case "enter":
			selected := m.actions[m.cursor]
			if selected.disabled {
				return m, nil
			}
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
			case "sync":
				dialog := NewSyncDialog(m.entry.Path)
				return m, func() tea.Msg {
					return msgs.ShowDialogMsg{Content: dialog}
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
	actionDisabledStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("240"))
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
		switch {
		case a.disabled:
			s += actionDisabledStyle.Render(label)
		case i == m.cursor && a.key == "rm":
			s += dangerSelectedStyle.Render(label)
		case i == m.cursor:
			s += actionSelectedStyle.Render(label)
		default:
			s += actionNormalStyle.Render(label)
		}
		s += "\n"
	}

	s += "\n"
	s += helpStyle.Render("  up/down select  |  enter confirm  |  esc cancel")
	return tea.NewView(s)
}

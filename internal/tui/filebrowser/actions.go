package filebrowser

import (
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/preview"
	"github.com/saintedlama/pcloud-cli/internal/tui/selector"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

type action struct {
	label    string
	key      string
	disabled bool
	style    selector.ItemStyle
}

var folderActions = []action{
	{label: "Open", key: "open"},
	{label: "Download", key: "download"},
	{label: "Rename", key: "rename"},
	{label: "Move", key: "move"},
	{label: "Delete", key: "rm", style: selector.StyleDanger},
	{label: "Sync", key: "sync"},
}

// ActionsDialog lets the user pick an action to perform on a file or folder.
type ActionsDialog struct {
	api    pcloud.CloudAPI
	entry  msgs.Entry
	list   selector.Selector
	width  int
	height int
}

// NewActionsDialog creates an action picker for the given entry.
// Folder-specific actions are shown when entry.IsFolder is true.
// width and height are the current terminal dimensions, passed through to
// sub-dialogs (e.g. the preview dialog) that need them at construction time.
func NewActionsDialog(api pcloud.CloudAPI, entry msgs.Entry, width, height int) ActionsDialog {
	var actions []action
	if entry.IsFolder {
		actions = folderActions
	} else {
		actions = []action{
			{label: "Preview", key: "preview", disabled: preview.GetPreviewType(entry.Name) == preview.PreviewUnsupported},
			{label: "Download", key: "download"},
			{label: "Rename", key: "rename"},
			{label: "Move", key: "move"},
			{label: "Delete", key: "rm", style: selector.StyleDanger},
		}
	}
	cursor := 0
	for i, a := range actions {
		if !a.disabled {
			cursor = i
			break
		}
	}
	items := make([]selector.Item, len(actions))
	for i, a := range actions {
		items[i] = selector.Item{Label: a.label, Key: a.key, Disabled: a.disabled, Style: a.style}
	}
	return ActionsDialog{
		api:    api,
		entry:  entry,
		list:   selector.New(items, cursor),
		width:  width,
		height: height,
	}
}

func (m ActionsDialog) Init() tea.Cmd {
	return nil
}

func (m ActionsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.list, _ = m.list.Update(msg)
		switch kMsg.String() {
		case "enter":
			sel, _ := m.list.Selected()
			if sel.Disabled {
				return m, nil
			}
			switch sel.Key {
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

func (m ActionsDialog) View() tea.View {
	s := tuistyles.Title.Render("pCloud") + "  "
	s += tuistyles.DialogTitle.Render("Actions")
	s += "\n\n"
	kind := "File"
	if m.entry.IsFolder {
		kind = "Folder"
	}
	s += "  " + kind + ": " + tuistyles.Path.Render(m.entry.Path) + "\n\n"
	s += m.list.View()
	s += "\n"
	s += tuistyles.Help.Render("  up/down select  |  enter confirm  |  esc cancel")
	return tea.NewView(s)
}

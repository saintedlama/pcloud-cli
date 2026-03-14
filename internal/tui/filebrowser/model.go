package filebrowser

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// Model is the filebrowser component.
type Model struct {
	list      list.Model
	spinner   spinner.Model
	api       pcloud.CloudAPI
	path      string
	history   []string
	loading   bool
	statusMsg string
	err       error
	width     int
	height    int
}

// New creates a new filebrowser starting at root.
func New(api pcloud.CloudAPI, width, height int) Model {
	l := list.New(nil, tabularDelegate{}, width, height-4)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		list:    l,
		spinner: s,
		api:     api,
		width:   width,
		height:  height,
	}
}

// Init kicks off the initial folder load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchFolder(m.api, "/"))
}

// SetSize updates the component dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h-4)
}

// Update handles messages for the filebrowser component.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case msgs.CloseDialogMsg:
		// Navigate into a folder (from the actions dialog "Open" item).
		if nav, ok := msg.Result.(msgs.NavigateFolderResult); ok {
			m.history = append(m.history, m.path)
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, fetchFolder(m.api, nav.Path))
		}
		// A mutating action finished — refresh the current folder.
		if msg.Result != nil {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, fetchFolder(m.api, m.path))
		}
		return m, nil

	case msgs.FolderLoadedMsg:
		m.loading = false
		m.err = nil
		m.path = msg.Path
		items := make([]list.Item, 0, len(msg.Items)+1)
		if msg.Path != "/" {
			items = append(items, newItem(msgs.Entry{Name: ".."}))
		}
		for _, e := range msg.Items {
			items = append(items, newItem(e))
		}
		cmd := m.list.SetItems(items)
		return m, cmd

	case msgs.ErrMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		m.statusMsg = ""

		// Don't intercept keys while the list is filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "enter", "right":
			if sel, ok := m.list.SelectedItem().(item); ok {
				if sel.entry.Name == ".." {
					return m.navigateUp()
				}
				if sel.entry.IsFolder {
					// 'right' navigates directly; 'enter' opens the actions menu.
					if msg.String() == "right" {
						return m.navigateInto(sel.entry.Path)
					}
					dialog := NewActionsDialog(m.api, sel.entry, m.width, m.height)
					return m, func() tea.Msg {
						return msgs.ShowDialogMsg{Content: dialog}
					}
				}
				// File selected with 'enter' -> show action picker.
				if msg.String() == "enter" {
					dialog := NewActionsDialog(m.api, sel.entry, m.width, m.height)
					return m, func() tea.Msg {
						return msgs.ShowDialogMsg{Content: dialog}
					}
				}
			}

		case "p":
			if sel, ok := m.list.SelectedItem().(item); ok {
				if !sel.entry.IsFolder && sel.entry.Name != ".." {
					dialog := NewPreviewDialog(m.api, sel.entry, m.width, m.height)
					return m, func() tea.Msg {
						return msgs.ShowDialogMsg{Content: dialog}
					}
				}
			}

		case "backspace", "left":
			return m.navigateUp()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the filebrowser.
func (m Model) View() string {
	header := titleStyle.Render("pCloud") + "  " + pathStyle.Render(m.path)
	if m.list.FilterState() == list.FilterApplied {
		header += "  " + m.list.FilterInput.View()
	}
	header += "\n\n"

	if m.err != nil {
		return header +
			errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n" +
			helpStyle.Render("Press backspace to go back, q to quit")
	}

	if m.loading {
		return header + "  " + m.spinner.View() + "  Loading..."
	}

	var footer string
	if m.statusMsg != "" {
		footer = successStyle.Render(m.statusMsg)
	} else {
		footer = helpStyle.Render("up/down navigate  |  right/enter open folder  |  enter download file  |  p preview  |  left/backspace go up  |  / filter  |  q quit")
	}
	return header + m.list.View() + "\n" + footer
}

// navigateInto pushes the current path onto history and loads the given path.
func (m Model) navigateInto(path string) (Model, tea.Cmd) {
	m.history = append(m.history, m.path)
	m.loading = true
	return m, tea.Batch(m.spinner.Tick, fetchFolder(m.api, path))
}

// navigateUp pops history or computes the parent path and navigates there.
func (m Model) navigateUp() (Model, tea.Cmd) {
	var target string
	if len(m.history) > 0 {
		target = m.history[len(m.history)-1]
		m.history = m.history[:len(m.history)-1]
	} else {
		target = parentPath(m.path)
		if target == m.path {
			return m, nil
		}
	}
	m.loading = true
	return m, tea.Batch(m.spinner.Tick, fetchFolder(m.api, target))
}

// parentPath returns the path of the parent directory.
func parentPath(p string) string {
	if p == "/" || p == "" {
		return "/"
	}
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return "/"
	}
	return p[:i]
}

// fetchFolder returns a command that loads the given path from the API.
func fetchFolder(api pcloud.CloudAPI, path string) tea.Cmd {
	return func() tea.Msg {
		resp, err := api.ListFolder(path, pcloud.ListFolderOptions{})
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}

		entries := make([]msgs.Entry, 0, len(resp.Metadata.Contents))
		for _, c := range resp.Metadata.Contents {
			entries = append(entries, msgs.Entry{
				Name:     c.Name,
				Path:     c.Path,
				IsFolder: c.IsFolder,
				Size:     c.Size,
				Modified: c.Modified,
			})
		}

		return msgs.FolderLoadedMsg{
			Path:  resp.Metadata.Path,
			Items: entries,
		}
	}
}

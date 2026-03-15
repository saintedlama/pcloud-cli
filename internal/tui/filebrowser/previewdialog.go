package filebrowser

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/saintedlama/pcloud-cli/internal/tui/preview"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

var (
	previewBorderStyle = tuistyles.DialogBorder
)

// PreviewDialog is a dialog-compatible tea.Model that fetches and displays
// a scrollable file preview inside a viewport.
type PreviewDialog struct {
	api      pcloud.CloudAPI
	entry    msgs.Entry
	viewport viewport.Model
	spinner  spinner.Model
	loading  bool
	err      error
	width    int
	height   int
}

// NewPreviewDialog builds a preview dialog for the given file entry.
func NewPreviewDialog(api pcloud.CloudAPI, entry msgs.Entry, width, height int) PreviewDialog {
	// Reserve space for border (2) + title row (2) + help row (1).
	vpW := width - 4
	vpH := height - 7
	if vpW < 10 {
		vpW = 10
	}
	if vpH < 3 {
		vpH = 3
	}

	vp := viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
	vp.SoftWrap = false // keep lines intact; ASCII art and syntax-highlighted code must not be re-wrapped

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return PreviewDialog{
		api:      api,
		entry:    entry,
		viewport: vp,
		spinner:  s,
		loading:  true,
		width:    width,
		height:   height,
	}
}

func (m PreviewDialog) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchPreview(m.api, m.entry, m.viewport.Width(), m.viewport.Height()))
}

func (m PreviewDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.PreviewReadyMsg:
		m.loading = false
		m.viewport.SetContent(msg.Content)
		m.viewport.GotoTop()
		return m, nil

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
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m PreviewDialog) View() tea.View {
	title := tuistyles.Title.Render("Preview") + "  " + tuistyles.Path.Render(m.entry.Name)

	var body string
	switch {
	case m.err != nil:
		body = tuistyles.Error.Render(fmt.Sprintf("Error: %v", m.err))
	case m.loading:
		body = tuistyles.Help.Render("  " + m.spinner.View() + "  Loading preview…")
	default:
		body = m.viewport.View()
	}

	scrollInfo := ""
	if !m.loading && m.err == nil {
		pct := int(m.viewport.ScrollPercent() * 100)
		scrollInfo = fmt.Sprintf(" %d%%", pct)
	}

	help := tuistyles.Help.Render("↑/↓ scroll  |  esc close") + scrollInfo

	inner := title + "\n\n" + body + "\n" + help

	// Pad/wrap inner to the target size minus border.
	innerW := m.width - 4
	innerH := m.height - 4
	if innerW < 4 {
		innerW = 4
	}
	if innerH < 2 {
		innerH = 2
	}

	// Ensure the inner block is exactly innerH lines tall so the border fits.
	lines := strings.Split(inner, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	inner = strings.Join(lines[:innerH], "\n")

	bordered := previewBorderStyle.
		Width(innerW).
		Height(innerH).
		Render(inner)

	return tea.NewView(bordered)
}

// fetchPreview is a tea.Cmd that fetches and renders the file preview.
func fetchPreview(api pcloud.CloudAPI, entry msgs.Entry, width, height int) tea.Cmd {
	return func() (result tea.Msg) {
		// Recover from panics inside third-party decoders (e.g. bad JPEG/PNG
		// data) so a corrupt file never crashes the TUI.
		defer func() {
			if r := recover(); r != nil {
				result = msgs.ErrMsg{Err: fmt.Errorf("preview panic: %v", r)}
			}
		}()

		link, err := api.GetFileLink(entry.Path)
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		if len(link.Hosts) == 0 {
			return msgs.ErrMsg{Err: fmt.Errorf("no download hosts returned")}
		}

		downloadURL := "https://" + link.Hosts[0] + link.Path
		content, err := preview.RenderFromURL(downloadURL, entry.Name, width, height)
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}

		return msgs.PreviewReadyMsg{Name: entry.Name, Content: content}
	}
}

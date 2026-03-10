package filebrowser

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// item implements list.Item for a file or folder entry.
type item struct {
	entry msgs.Entry
}

func newItem(e msgs.Entry) item {
	return item{entry: e}
}

func (i item) Title() string       { return i.entry.Name }
func (i item) FilterValue() string { return i.entry.Name }

// tabularDelegate renders each entry as a single-line row with aligned columns.
type tabularDelegate struct{}

const (
	colSizeWidth = 9  // "9999.9 MB"
	colDateWidth = 12 // "Mar 10, 2026"
)

func (d tabularDelegate) Height() int                             { return 1 }
func (d tabularDelegate) Spacing() int                            { return 0 }
func (d tabularDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d tabularDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	// 1(margin) + 2(prefix) + name(flex) + 2(gap) + colSizeWidth + 2(gap) + colDateWidth
	nameWidth := m.Width() - 1 - 2 - 2 - colSizeWidth - 2 - colDateWidth
	if nameWidth < 4 {
		nameWidth = 4
	}

	var prefix, sizeStr, dateStr string
	switch {
	case i.entry.Name == "..":
		prefix = "^ "
		sizeStr = strings.Repeat(" ", colSizeWidth)
		dateStr = strings.Repeat(" ", colDateWidth)
	case i.entry.IsFolder:
		prefix = "> "
		sizeStr = strings.Repeat(" ", colSizeWidth)
		dateStr = fmt.Sprintf("%-*s", colDateWidth, formatDate(i.entry.Modified))
	default:
		prefix = "  "
		sizeStr = fmt.Sprintf("%*s", colSizeWidth, formatSize(i.entry.Size))
		dateStr = fmt.Sprintf("%-*s", colDateWidth, formatDate(i.entry.Modified))
	}

	name := padOrTruncate(i.entry.Name, nameWidth)
	row := fmt.Sprintf(" %s%s  %s  %s", prefix, name, sizeStr, dateStr)

	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(row))
	} else if i.entry.IsFolder || i.entry.Name == ".." {
		fmt.Fprint(w, folderStyle.Render(row))
	} else {
		fmt.Fprint(w, normalStyle.Render(row))
	}
}

func padOrTruncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		if width > 1 {
			return string(runes[:width-1]) + "\u2026"
		}
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func formatDate(s string) string {
	t, err := time.Parse(time.RFC1123Z, s)
	if err != nil {
		if len(s) > colDateWidth {
			return s[:colDateWidth]
		}
		return s
	}
	return t.Format("Jan 02, 2006")
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// getLines splits a string into lines and returns the widest line width.
func getLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")
	for _, l := range lines {
		w := ansi.StringWidth(l)
		if widest < w {
			widest = w
		}
	}
	return lines, widest
}

// OverlayOption configures overlay behavior.
type OverlayOption func(*overlayOptions)

type overlayOptions struct {
	dim bool
}

// WithDim dims the background behind the overlay.
func WithDim() OverlayOption {
	return func(o *overlayOptions) {
		o.dim = true
	}
}

func dimContent(s string) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = dimStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}

// Overlay places fg on top of bg at position (x, y).
// This is an ANSI-aware overlay that correctly handles styled text.
// Based on https://github.com/charmbracelet/lipgloss/pull/102
// with the buggy cutLeft replaced by charmbracelet/x/ansi.TruncateLeft.
func Overlay(x, y int, fg, bg string, opts ...OverlayOption) string {
	var o overlayOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.dim {
		bg = dimContent(bg)
	}
	fgLines, fgWidth := getLines(fg)
	bgLines, bgWidth := getLines(bg)
	bgHeight := len(bgLines)
	fgHeight := len(fgLines)

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return fg
	}

	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	var b strings.Builder
	for i, bgLine := range bgLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			b.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := ansi.Truncate(bgLine, x, "")
			pos = ansi.StringWidth(left)
			b.WriteString(left)
			if pos < x {
				b.WriteString(strings.Repeat(" ", x-pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		b.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		right := ansi.TruncateLeft(bgLine, pos, "")
		bgLineWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgLineWidth-pos {
			b.WriteString(strings.Repeat(" ", bgLineWidth-rightWidth-pos))
		}

		b.WriteString(right)
	}

	return b.String()
}

// OverlayCenter places fg centered on top of bg within the given dimensions.
func OverlayCenter(width, height int, fg, bg string, opts ...OverlayOption) string {
	fgLines, fgWidth := getLines(fg)
	fgHeight := len(fgLines)

	x := (width - fgWidth) / 2
	y := (height - fgHeight) / 2

	return Overlay(x, y, fg, bg, opts...)
}

func clamp(v, lower, upper int) int {
	return min(max(v, lower), upper)
}

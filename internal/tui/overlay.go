package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

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

// OverlayCenter places fg centered on top of bg within the given dimensions.
// Uses lipgloss's built-in cell-based compositor for ANSI-aware compositing.
func OverlayCenter(width, height int, fg, bg string, opts ...OverlayOption) string {
	var o overlayOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.dim {
		bg = dimContent(bg)
	}

	fgLayer := lipgloss.NewLayer(fg)
	x := (width - fgLayer.Width()) / 2
	y := (height - fgLayer.Height()) / 2

	bgLayer := lipgloss.NewLayer(bg).Z(0)
	fgLayer = fgLayer.X(x).Y(y).Z(1)

	return lipgloss.NewCanvas(width, height).
		Compose(lipgloss.NewCompositor(bgLayer, fgLayer)).
		Render()
}

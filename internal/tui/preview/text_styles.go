package preview

import "charm.land/lipgloss/v2"

// Terminal styles shared by all renderers (markdown, PDF, …).
// Renderers that need heuristic mapping (e.g. PDF) call headingStyle(level);
// renderers that know the exact semantic node (e.g. Markdown AST) use the
// vars directly.

var (
	// Heading levels — H1 most prominent, H6 faintest.
	styleH1 = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("12"))
	styleH2 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleH3 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleH4 = lipgloss.NewStyle().Bold(true)
	styleH5 = lipgloss.NewStyle().Bold(true).Faint(true)
	styleH6 = lipgloss.NewStyle().Faint(true)

	// Inline emphasis.
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleItalic = lipgloss.NewStyle().Italic(true)

	// Code — inline span uses reverse video; blocks use faint monospace colour.
	styleCodeSpan  = lipgloss.NewStyle().Reverse(true)
	styleCodeBlock = lipgloss.NewStyle().Faint(true)

	// Block elements.
	styleBlockquote = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleListBullet = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	// Captions, footnotes, faint auxiliary text.
	styleFaint = lipgloss.NewStyle().Faint(true)
)

// headingStyle returns the style for heading levels 1–6.
func headingStyle(level int) lipgloss.Style {
	switch level {
	case 1:
		return styleH1
	case 2:
		return styleH2
	case 3:
		return styleH3
	case 4:
		return styleH4
	case 5:
		return styleH5
	default:
		return styleH6
	}
}

// Package styles defines the shared visual tokens for the pcloud-cli TUI.
// All sub-packages (filebrowser, systemd, selector, …) reference these
// variables instead of redefining identical lipgloss styles locally.
package styles

import "charm.land/lipgloss/v2"

var (
	// Title is the primary heading style: bold bright-cyan.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Padding(0, 1)

	// Help is the muted hint/key-binding footer style.
	Help = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	// Error is used for error messages and destructive labels.
	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Padding(0, 1)

	// Success is used for confirmation and success messages.
	Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Padding(0, 1)

	// Selection is the highlighted-row/item style (violet bg, white fg).
	Selection = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("62"))

	// DialogBorder is a rounded violet border used by all dialog frames.
	// Callers may chain additional options (e.g. .Padding(1, 2)) as needed.
	DialogBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	// Path is the muted italic style used to display file/folder paths.
	Path = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Padding(0, 1)

	// Folder is the bright-blue style used for folder names in list views.
	Folder = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	// Normal is a plain unstyled default used for regular file names.
	Normal = lipgloss.NewStyle()

	// DialogTitle is the bright-yellow bold style used for dialog section headings.
	DialogTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11"))
)

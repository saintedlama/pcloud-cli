package filebrowser

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")). // bright cyan
			Padding(0, 1)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")). // grey
			Italic(true).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // bright red
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")). // dark grey
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("62"))

	folderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")) // bright blue

	normalStyle = lipgloss.NewStyle()

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11")) // bright yellow

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // bright green
			Padding(0, 1)
)

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

var dialogBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1, 2)

// dialogModel wraps a dialog content model on top of a background model.
// Keys and messages are forwarded to the dialog content model.
// When the dialog sends CloseDialogMsg or the user presses Esc
// the dialog is dismissed and the background model is restored.
type dialogModel struct {
	dialog tea.Model
	main   tea.Model
	width  int
	height int
}

func NewDialogModel(dialog, main tea.Model, width, height int) dialogModel {
	return dialogModel{
		dialog: dialog,
		main:   main,
		width:  width,
		height: height,
	}
}

func (m dialogModel) Init() tea.Cmd {
	return m.dialog.Init()
}

func (m dialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.CloseDialogMsg:
		return m.main, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Also forward to main so it stays in sync.
		m.main, _ = m.main.Update(msg)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m.main, nil
		}
	}

	var cmd tea.Cmd
	m.dialog, cmd = m.dialog.Update(msg)
	return m, cmd
}

func (m dialogModel) View() string {
	bg := m.main.View()
	dialogContent := dialogBoxStyle.Render(m.dialog.View())

	return OverlayCenter(m.width, m.height, dialogContent, bg, WithDim())
}

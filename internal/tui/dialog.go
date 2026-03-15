package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	tuistyles "github.com/saintedlama/pcloud-cli/internal/tui/styles"
)

var dialogBoxStyle = tuistyles.DialogBorder.Padding(1, 2)

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
	case msgs.ShowDialogMsg:
		// Replace the inner dialog (e.g. action picker -> sub-dialog).
		m.dialog = msg.Content
		return m, m.dialog.Init()
	case msgs.CloseDialogMsg:
		// Forward results that need parent handling (e.g. folder refresh).
		if msg.Result != nil {
			var cmd tea.Cmd
			m.main, cmd = m.main.Update(msg)
			return m.main, cmd
		}
		return m.main, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.main, _ = m.main.Update(msg)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m.main, nil
		}
	}

	var cmd tea.Cmd
	m.dialog, cmd = m.dialog.Update(msg)
	return m, cmd
}

func (m dialogModel) View() tea.View {
	bg := m.main.View().Content
	dialogContent := dialogBoxStyle.Render(m.dialog.View().Content)

	v := tea.NewView(OverlayCenter(m.width, m.height, dialogContent, bg, WithDim()))
	v.AltScreen = true
	return v
}

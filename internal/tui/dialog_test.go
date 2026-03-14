package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModel is a minimal tea.Model used to stand in for both the main model
// and dialog content in orchestration tests. It records the last message it
// received so tests can assert forwarding behaviour.
type fakeModel struct {
	id      string
	lastMsg tea.Msg
}

func (f *fakeModel) Init() tea.Cmd                           { return nil }
func (f *fakeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { f.lastMsg = msg; return f, nil }
func (f *fakeModel) View() tea.View                          { return tea.NewView(f.id) }

func TestDialogModel_ShowDialogMsg_ReplacesDialog(t *testing.T) {
	main := &fakeModel{id: "main"}
	orig := &fakeModel{id: "orig-dialog"}
	replacement := &fakeModel{id: "replacement-dialog"}

	dm := NewDialogModel(orig, main, 80, 24)

	updated, _ := dm.Update(msgs.ShowDialogMsg{Content: replacement})
	newDm := updated.(dialogModel)

	assert.Equal(t, "replacement-dialog", newDm.dialog.View().Content)
}

func TestDialogModel_CloseDialogMsg_NilResult_ReturnsMain(t *testing.T) {
	main := &fakeModel{id: "main"}
	dlg := &fakeModel{id: "dialog"}

	dm := NewDialogModel(dlg, main, 80, 24)

	updated, _ := dm.Update(msgs.CloseDialogMsg{})
	// With nil Result, the main model is returned directly.
	fm, ok := updated.(*fakeModel)
	require.True(t, ok, "expected main *fakeModel to be returned")
	assert.Equal(t, "main", fm.id)
}

func TestDialogModel_CloseDialogMsg_NonNilResult_ForwardsToMain(t *testing.T) {
	main := &fakeModel{id: "main"}
	dlg := &fakeModel{id: "dialog"}

	dm := NewDialogModel(dlg, main, 80, 24)

	closeMsg := msgs.CloseDialogMsg{Result: "mutated"}
	updated, _ := dm.Update(closeMsg)

	fm, ok := updated.(*fakeModel)
	require.True(t, ok, "expected main *fakeModel to be returned after forwarding")
	// The main model should have received the CloseDialogMsg.
	assert.Equal(t, closeMsg, fm.lastMsg)
}

func TestDialogModel_Esc_DismissesToMain(t *testing.T) {
	main := &fakeModel{id: "main"}
	dlg := &fakeModel{id: "dialog"}

	dm := NewDialogModel(dlg, main, 80, 24)

	escKey := tea.KeyPressMsg{Code: tea.KeyEscape}
	updated, _ := dm.Update(escKey)

	fm, ok := updated.(*fakeModel)
	require.True(t, ok, "esc should return the main model")
	assert.Equal(t, "main", fm.id)
}

func TestDialogModel_OtherKeys_ForwardedToDialog(t *testing.T) {
	main := &fakeModel{id: "main"}
	dlg := &fakeModel{id: "dialog"}

	dm := NewDialogModel(dlg, main, 80, 24)

	keyMsg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	dm.Update(keyMsg)

	// The dialog should have received the key message.
	assert.Equal(t, keyMsg, dlg.lastMsg)
}

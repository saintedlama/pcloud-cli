package filebrowser

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enterKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func anyKey() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: 'q', Text: "q"} }

func TestDeleteDialog_ConfirmY_File(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDeleteDialog(api, entry)
	d.input.SetValue("Y")

	// Enter → deleteRunning; returned cmd performs the actual deletion.
	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)
	dlg := m.(*DeleteDialog)
	assert.Equal(t, deleteRunning, dlg.state)

	// Execute the cmd — calls api.DeleteFile.
	resultMsg := cmd()
	assert.IsType(t, deleteFileMsg{}, resultMsg)
	assert.Equal(t, []string{"/foo/bar.txt"}, api.DeleteFileCalls)
	assert.Empty(t, api.DeleteFolderCalls)

	// Deliver result → deleteDone.
	m, _ = dlg.Update(resultMsg)
	dlg = m.(*DeleteDialog)
	assert.Equal(t, deleteDone, dlg.state)
	assert.Contains(t, dlg.View().Content, "Deleted successfully")

	// Any key after done emits CloseDialogMsg with "deleted" result.
	_, cmd = dlg.Update(anyKey())
	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Equal(t, "deleted", closeMsg.(msgs.CloseDialogMsg).Result)
}

func TestDeleteDialog_ConfirmY_Folder(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs", Name: "docs", IsFolder: true}

	d := NewDeleteDialog(api, entry)
	d.input.SetValue("Y")

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	resultMsg := cmd()
	assert.IsType(t, deleteFileMsg{}, resultMsg)
	assert.Equal(t, []string{"/docs"}, api.DeleteFolderCalls)
	assert.Empty(t, api.DeleteFileCalls)
}

func TestDeleteDialog_Cancel(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDeleteDialog(api, entry)
	// Empty input → any enter dismisses without deleting.
	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Nil(t, closeMsg.(msgs.CloseDialogMsg).Result)
	assert.Empty(t, api.DeleteFileCalls)
}

func TestDeleteDialog_APIError(t *testing.T) {
	api := &pcloudtest.StubAPI{DeleteFileErr: errors.New("permission denied")}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDeleteDialog(api, entry)
	d.input.SetValue("Y")

	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	// Execute cmd — api returns error.
	errMsg := cmd()
	assert.IsType(t, msgs.ErrMsg{}, errMsg)

	// Deliver error → error state.
	m, _ = m.(*DeleteDialog).Update(errMsg)
	dlg := m.(*DeleteDialog)
	assert.NotNil(t, dlg.err)
	assert.Contains(t, dlg.View().Content, "permission denied")

	// Any key dismisses.
	_, cmd = dlg.Update(anyKey())
	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
}

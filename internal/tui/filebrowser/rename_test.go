package filebrowser

import (
	"errors"
	"testing"

	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameDialog_Rename_File(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/old.txt", Name: "old.txt"}

	d := NewRenameDialog(api, entry)
	d.input.SetValue("new.txt")

	// Enter → renameRunning.
	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)
	dlg := m.(*RenameDialog)
	assert.Equal(t, renameRunning, dlg.state)

	// Execute cmd — calls api.RenameFile.
	resultMsg := cmd()
	assert.IsType(t, renameFileMsg{}, resultMsg)
	require.Len(t, api.RenameFileCalls, 1)
	assert.Equal(t, "/docs/old.txt", api.RenameFileCalls[0][0])
	assert.Equal(t, "/docs/new.txt", api.RenameFileCalls[0][1])

	// Deliver result → renameDone.
	m, _ = dlg.Update(resultMsg)
	dlg = m.(*RenameDialog)
	assert.Equal(t, renameDone, dlg.state)
	assert.Contains(t, dlg.View().Content, "Renamed successfully")

	// Any key → CloseDialogMsg with "renamed".
	_, cmd = dlg.Update(anyKey())
	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Equal(t, "renamed", closeMsg.(msgs.CloseDialogMsg).Result)
}

func TestRenameDialog_Rename_Folder(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/music/jazz", Name: "jazz", IsFolder: true}

	d := NewRenameDialog(api, entry)
	d.input.SetValue("blues")

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	resultMsg := cmd()
	assert.IsType(t, renameFileMsg{}, resultMsg)
	require.Len(t, api.RenameFolderCalls, 1)
	assert.Equal(t, "/music/jazz", api.RenameFolderCalls[0][0])
	assert.Equal(t, "/music/blues", api.RenameFolderCalls[0][1])
}

func TestRenameDialog_CancelEmpty(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/old.txt", Name: "old.txt"}

	d := NewRenameDialog(api, entry)
	d.input.SetValue("")

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Nil(t, closeMsg.(msgs.CloseDialogMsg).Result)
	assert.Empty(t, api.RenameFileCalls)
}

func TestRenameDialog_CancelSameName(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/old.txt", Name: "old.txt"}

	d := NewRenameDialog(api, entry)
	// Value is pre-populated with entry.Name; pressing enter without change cancels.

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Nil(t, closeMsg.(msgs.CloseDialogMsg).Result)
	assert.Empty(t, api.RenameFileCalls)
}

func TestRenameDialog_APIError(t *testing.T) {
	api := &pcloudtest.StubAPI{RenameFileErr: errors.New("not found")}
	entry := msgs.Entry{Path: "/docs/old.txt", Name: "old.txt"}

	d := NewRenameDialog(api, entry)
	d.input.SetValue("other.txt")

	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	errMsg := cmd()
	assert.IsType(t, msgs.ErrMsg{}, errMsg)

	m, _ = m.(*RenameDialog).Update(errMsg)
	dlg := m.(*RenameDialog)
	assert.NotNil(t, dlg.err)
	assert.Contains(t, dlg.View().Content, "not found")
}

package filebrowser

import (
	"errors"
	"testing"

	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveDialog_Move_File(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/report.txt", Name: "report.txt"}

	d := NewMoveDialog(api, entry)
	d.input.SetValue("/archive")

	// Enter → moveRunning.
	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)
	dlg := m.(*MoveDialog)
	assert.Equal(t, moveRunning, dlg.state)

	// Execute cmd — calls api.RenameFile (move = rename to new path).
	resultMsg := cmd()
	assert.IsType(t, moveFileMsg{}, resultMsg)
	require.Len(t, api.RenameFileCalls, 1)
	assert.Equal(t, "/docs/report.txt", api.RenameFileCalls[0][0])
	assert.Equal(t, "/archive/report.txt", api.RenameFileCalls[0][1])

	// Deliver result → moveDone.
	m, _ = dlg.Update(resultMsg)
	dlg = m.(*MoveDialog)
	assert.Equal(t, moveDone, dlg.state)
	assert.Contains(t, dlg.View().Content, "Moved successfully")

	// Any key → CloseDialogMsg with "moved".
	_, cmd = dlg.Update(anyKey())
	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Equal(t, "moved", closeMsg.(msgs.CloseDialogMsg).Result)
}

func TestMoveDialog_Move_Folder(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/music/jazz", Name: "jazz", IsFolder: true}

	d := NewMoveDialog(api, entry)
	d.input.SetValue("/archive")

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	resultMsg := cmd()
	assert.IsType(t, moveFileMsg{}, resultMsg)
	require.Len(t, api.RenameFolderCalls, 1)
	assert.Equal(t, "/music/jazz", api.RenameFolderCalls[0][0])
	assert.Equal(t, "/archive/jazz", api.RenameFolderCalls[0][1])
}

func TestMoveDialog_CancelEmpty(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/report.txt", Name: "report.txt"}

	d := NewMoveDialog(api, entry)
	d.input.SetValue("")

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Nil(t, closeMsg.(msgs.CloseDialogMsg).Result)
	assert.Empty(t, api.RenameFileCalls)
}

func TestMoveDialog_CancelSamePath(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs/report.txt", Name: "report.txt"}

	// Default value is the parent path "/docs"; entering the same parent cancels.
	d := NewMoveDialog(api, entry)
	// input pre-filled with "/docs" (the parent dir).

	_, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
	assert.Nil(t, closeMsg.(msgs.CloseDialogMsg).Result)
	assert.Empty(t, api.RenameFileCalls)
}

func TestMoveDialog_APIError(t *testing.T) {
	api := &pcloudtest.StubAPI{RenameFileErr: errors.New("quota exceeded")}
	entry := msgs.Entry{Path: "/docs/report.txt", Name: "report.txt"}

	d := NewMoveDialog(api, entry)
	d.input.SetValue("/archive")

	m, cmd := d.Update(enterKey())
	require.NotNil(t, cmd)

	errMsg := cmd()
	assert.IsType(t, msgs.ErrMsg{}, errMsg)

	m, _ = m.(*MoveDialog).Update(errMsg)
	dlg := m.(*MoveDialog)
	assert.NotNil(t, dlg.err)
	assert.Contains(t, dlg.View().Content, "quota exceeded")
}

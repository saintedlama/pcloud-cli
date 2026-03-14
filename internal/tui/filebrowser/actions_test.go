package filebrowser

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: 'j', Text: "j"} }
func kKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: 'k', Text: "k"} }

func TestActionsDialog_FileActions_DefaultCursor(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewActionsDialog(api, entry, 80, 24)
	// First enabled action for a non-.txt file with no preview support is "Download" (idx 1),
	// but for a .txt file preview IS supported so cursor starts at index 0 ("Preview").
	assert.Equal(t, 0, d.cursor)
}

func TestActionsDialog_Navigation_JK(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	m := NewActionsDialog(api, entry, 80, 24)
	assert.Equal(t, 0, m.cursor)

	// j moves down.
	updated, _ := m.Update(jKey())
	m = updated.(ActionsDialog)
	assert.Equal(t, 1, m.cursor)

	// k moves back up.
	updated, _ = m.Update(kKey())
	m = updated.(ActionsDialog)
	assert.Equal(t, 0, m.cursor)

	// k at top stays at 0.
	updated, _ = m.Update(kKey())
	m = updated.(ActionsDialog)
	assert.Equal(t, 0, m.cursor)
}

func TestActionsDialog_SelectDelete_ShowsDeleteDialog(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewActionsDialog(api, entry, 80, 24)
	// Navigate to "rm" (Delete): Preview(0), Download(1), Rename(2), Move(3), Delete(4).
	for i := 0; i < 4; i++ {
		m, _ := tea.Model(d).Update(jKey())
		d = m.(ActionsDialog)
	}
	assert.Equal(t, 4, d.cursor)

	_, cmd := tea.Model(d).Update(enterKey())
	require.NotNil(t, cmd)

	showMsg := cmd()
	require.IsType(t, msgs.ShowDialogMsg{}, showMsg)
	assert.IsType(t, &DeleteDialog{}, showMsg.(msgs.ShowDialogMsg).Content)
}

func TestActionsDialog_SelectRename_ShowsRenameDialog(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewActionsDialog(api, entry, 80, 24)
	// Navigate to Rename (idx 2).
	for i := 0; i < 2; i++ {
		m, _ := tea.Model(d).Update(jKey())
		d = m.(ActionsDialog)
	}

	_, cmd := tea.Model(d).Update(enterKey())
	require.NotNil(t, cmd)

	showMsg := cmd()
	require.IsType(t, msgs.ShowDialogMsg{}, showMsg)
	assert.IsType(t, &RenameDialog{}, showMsg.(msgs.ShowDialogMsg).Content)
}

func TestActionsDialog_SelectMove_ShowsMoveDialog(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewActionsDialog(api, entry, 80, 24)
	// Navigate to Move (idx 3).
	for i := 0; i < 3; i++ {
		m, _ := tea.Model(d).Update(jKey())
		d = m.(ActionsDialog)
	}

	_, cmd := tea.Model(d).Update(enterKey())
	require.NotNil(t, cmd)

	showMsg := cmd()
	require.IsType(t, msgs.ShowDialogMsg{}, showMsg)
	assert.IsType(t, &MoveDialog{}, showMsg.(msgs.ShowDialogMsg).Content)
}

func TestActionsDialog_FolderActions_DeleteShowsDeleteDialog(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/docs", Name: "docs", IsFolder: true}

	d := NewActionsDialog(api, entry, 80, 24)
	// Folder actions: Open(0), Download(1), Rename(2), Move(3), Delete(4), Sync(5).
	// Navigate to Delete (idx 4).
	for i := 0; i < 4; i++ {
		m, _ := tea.Model(d).Update(jKey())
		d = m.(ActionsDialog)
	}

	_, cmd := tea.Model(d).Update(enterKey())
	require.NotNil(t, cmd)

	showMsg := cmd()
	require.IsType(t, msgs.ShowDialogMsg{}, showMsg)
	assert.IsType(t, &DeleteDialog{}, showMsg.(msgs.ShowDialogMsg).Content)
}

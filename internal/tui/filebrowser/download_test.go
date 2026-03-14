package filebrowser

import (
	"errors"
	"testing"

	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadDialog_Done(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDownloadDialog(api, entry)
	d.downloading = true

	m, _ := d.Update(msgs.DownloadDoneMsg{LocalPath: "bar.txt"})
	dlg := m.(DownloadDialog)

	assert.True(t, dlg.done)
	assert.Equal(t, "bar.txt", dlg.localPath)
	assert.Contains(t, dlg.View().Content, "Downloaded")
}

func TestDownloadDialog_Error(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDownloadDialog(api, entry)
	d.downloading = true

	m, _ := d.Update(msgs.ErrMsg{Err: errors.New("network error")})
	dlg := m.(DownloadDialog)

	assert.NotNil(t, dlg.err)
	assert.Contains(t, dlg.View().Content, "network error")
}

func TestDownloadDialog_DoneKeyDismisses(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/foo/bar.txt", Name: "bar.txt"}

	d := NewDownloadDialog(api, entry)
	d.done = true

	_, cmd := d.Update(anyKey())
	require.NotNil(t, cmd)

	closeMsg := cmd()
	require.IsType(t, msgs.CloseDialogMsg{}, closeMsg)
}

func TestFolderDownloadDialog_Done(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/music", Name: "music", IsFolder: true}

	d := NewFolderDownloadDialog(api, entry)
	d.downloading = true

	m, _ := d.Update(msgs.DownloadDoneMsg{LocalPath: "./music"})
	dlg := m.(FolderDownloadDialog)

	assert.True(t, dlg.done)
	assert.Contains(t, dlg.View().Content, "Downloaded")
}

func TestFolderDownloadDialog_Error(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	entry := msgs.Entry{Path: "/music", Name: "music", IsFolder: true}

	d := NewFolderDownloadDialog(api, entry)
	d.downloading = true

	m, _ := d.Update(msgs.ErrMsg{Err: errors.New("zip failed")})
	dlg := m.(FolderDownloadDialog)

	assert.NotNil(t, dlg.err)
	assert.Contains(t, dlg.View().Content, "zip failed")
}

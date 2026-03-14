package filebrowser

import (
	"errors"
	"testing"

	"github.com/saintedlama/pcloud-cli/internal/pcloudtest"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_FolderLoadedMsg_UpdatesState(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)

	items := []msgs.Entry{
		{Name: "file.txt", Path: "/file.txt"},
		{Name: "dir", Path: "/dir", IsFolder: true},
	}

	m2, _ := m.Update(msgs.FolderLoadedMsg{Path: "/", Items: items})

	assert.Equal(t, "/", m2.path)
	assert.False(t, m2.loading)
	assert.Nil(t, m2.err)
}

func TestModel_FolderLoadedMsg_SubdirAddsParent(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)

	m2, _ := m.Update(msgs.FolderLoadedMsg{
		Path:  "/docs",
		Items: []msgs.Entry{{Name: "readme.txt", Path: "/docs/readme.txt"}},
	})

	assert.Equal(t, "/docs", m2.path)
	// The ".." entry is prepended for non-root paths.
	require.Greater(t, len(m2.list.Items()), 0)
	first := m2.list.Items()[0].(item)
	assert.Equal(t, "..", first.entry.Name)
}

func TestModel_ErrMsg_SetsErrorState(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)

	m2, _ := m.Update(msgs.ErrMsg{Err: errors.New("connection refused")})

	assert.NotNil(t, m2.err)
	assert.False(t, m2.loading)
	assert.Contains(t, m2.View(), "connection refused")
}

func TestModel_CloseDialogMsg_MutatingResult_TriggersRefresh(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)
	m.path = "/photos"

	m2, cmd := m.Update(msgs.CloseDialogMsg{Result: "deleted"})

	assert.True(t, m2.loading)
	// A cmd must be returned to reload the folder.
	assert.NotNil(t, cmd)
}

func TestModel_CloseDialogMsg_NavigateFolderResult_PushesHistory(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)
	m.path = "/music"

	m2, cmd := m.Update(msgs.CloseDialogMsg{
		Result: msgs.NavigateFolderResult{Path: "/music/jazz"},
	})

	assert.True(t, m2.loading)
	assert.NotNil(t, cmd)
	require.Len(t, m2.history, 1)
	assert.Equal(t, "/music", m2.history[0])
}

func TestModel_CloseDialogMsg_NilResult_NoOp(t *testing.T) {
	api := &pcloudtest.StubAPI{}
	m := New(api, 80, 24)
	m.path = "/photos"

	m2, cmd := m.Update(msgs.CloseDialogMsg{})

	assert.False(t, m2.loading)
	assert.Nil(t, cmd)
}

func TestParentPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"root stays at root", "/", "/"},
		{"empty stays at root", "", "/"},
		{"one level up", "/docs", "/"},
		{"two levels up", "/docs/archive", "/docs"},
		{"trailing slash returns dir", "/docs/archive/", "/docs/archive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parentPath(tt.in))
		})
	}
}

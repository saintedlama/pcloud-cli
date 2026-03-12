package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintedlama/pcloud-cli/internal/pcloud/models"
)

// ---- formatSize ------------------------------------------------------------

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"just below KB", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"fractional KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.0 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSize(tt.input))
		})
	}
}

// ---- toLocalRel ------------------------------------------------------------

func TestToLocalRel(t *testing.T) {
	s := &Syncer{cloudRoot: "/Music", localRoot: "/local"}

	tests := []struct {
		name      string
		cloudPath string
		want      string
	}{
		{"nested path", "/Music/Rock/song.mp3", filepath.FromSlash("Rock/song.mp3")},
		{"file at root", "/Music/song.mp3", filepath.FromSlash("song.mp3")},
		{"deeply nested", "/Music/a/b/c.flac", filepath.FromSlash("a/b/c.flac")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.toLocalRel(tt.cloudPath))
		})
	}
}

func TestToLocalRel_TrailingSlashOnRoot(t *testing.T) {
	// New() strips trailing slash from cloudRoot; verify the conversion still works.
	s := New(nil, "/Music/", "/local", os.Stderr)
	assert.Equal(t, filepath.FromSlash("song.mp3"), s.toLocalRel("/Music/song.mp3"))
}

// ---- needsDownload ---------------------------------------------------------

func TestNeedsDownload_AbsentFile(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{cloudRoot: "/Music", localRoot: dir}

	e := fileEntry{localRel: "absent.mp3", modified: time.Now()}
	assert.True(t, s.needsDownload(e), "absent file should be downloaded")
}

func TestNeedsDownload_UpToDate(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{cloudRoot: "/Music", localRoot: dir}

	localPath := filepath.Join(dir, "song.mp3")
	require.NoError(t, os.WriteFile(localPath, []byte("data"), 0o644))
	now := time.Now()
	require.NoError(t, os.Chtimes(localPath, now, now))

	e := fileEntry{localRel: "song.mp3", modified: now.Add(-1 * time.Hour)}
	assert.False(t, s.needsDownload(e), "up-to-date file should not be downloaded")
}

func TestNeedsDownload_Stale(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{cloudRoot: "/Music", localRoot: dir}

	localPath := filepath.Join(dir, "song.mp3")
	require.NoError(t, os.WriteFile(localPath, []byte("data"), 0o644))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(localPath, old, old))

	e := fileEntry{localRel: "song.mp3", modified: time.Now()}
	assert.True(t, s.needsDownload(e), "stale file should be downloaded")
}

func TestNeedsDownload_ZeroModified(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{cloudRoot: "/Music", localRoot: dir}

	localPath := filepath.Join(dir, "song.mp3")
	require.NoError(t, os.WriteFile(localPath, []byte("data"), 0o644))

	e := fileEntry{localRel: "song.mp3", modified: time.Time{}}
	assert.False(t, s.needsDownload(e), "zero mtime on existing file should not trigger download")
}

// ---- countLocalFiles -------------------------------------------------------

func TestCountLocalFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	for _, name := range []string{"a.txt", "b.txt", filepath.Join("sub", "c.txt")} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
	}

	s := &Syncer{localRoot: dir}
	assert.Equal(t, 3, s.countLocalFiles())
}

func TestCountLocalFiles_MissingDir(t *testing.T) {
	s := &Syncer{localRoot: "/nonexistent-dir-xyz"}
	assert.Equal(t, 0, s.countLocalFiles())
}

// ---- pruneLocal ------------------------------------------------------------

func TestPruneLocal_RemovesOrphanFiles(t *testing.T) {
	dir := t.TempDir()
	s := New(nil, "/r", dir, os.Stderr)

	for _, name := range []string{"keep.txt", "orphan.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
	}

	deleted, err := s.pruneLocal(map[string]struct{}{"keep.txt": {}})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	assert.NoFileExists(t, filepath.Join(dir, "orphan.txt"))
	assert.FileExists(t, filepath.Join(dir, "keep.txt"))
}

func TestPruneLocal_EmptyRemoteSetRemovesAll(t *testing.T) {
	dir := t.TempDir()
	s := New(nil, "/r", dir, os.Stderr)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gone.txt"), []byte("x"), 0o644))

	deleted, err := s.pruneLocal(map[string]struct{}{})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
}

// ---- collectFiles ----------------------------------------------------------

func TestCollectFiles_FlatList(t *testing.T) {
	s := &Syncer{cloudRoot: "/Music", localRoot: "/local"}
	items := []models.FolderItem{
		{Name: "song.mp3", IsFolder: false, FileID: 1, Modified: "Mon, 01 Jan 2024 10:00:00 +0000"},
		{Name: "cover.jpg", IsFolder: false, FileID: 2, Modified: "Mon, 01 Jan 2024 10:00:00 +0000"},
	}

	var entries []fileEntry
	collectFiles(items, &entries, s, "/Music")

	require.Len(t, entries, 2)
	assert.Equal(t, "/Music/song.mp3", entries[0].cloudPath)
	assert.Equal(t, 1, entries[0].fileID)
}

func TestCollectFiles_SkipsFolders(t *testing.T) {
	s := &Syncer{cloudRoot: "/Music", localRoot: "/local"}
	items := []models.FolderItem{
		{Name: "Rock", IsFolder: true, Contents: []models.FolderItem{
			{Name: "song.mp3", IsFolder: false, FileID: 10},
		}},
	}

	var entries []fileEntry
	collectFiles(items, &entries, s, "/Music")

	require.Len(t, entries, 1, "folder itself should not be counted")
	assert.Equal(t, "/Music/Rock/song.mp3", entries[0].cloudPath)
}

func TestCollectFiles_InvalidModifiedTimeTreatedAsZero(t *testing.T) {
	s := &Syncer{cloudRoot: "/Music", localRoot: "/local"}
	items := []models.FolderItem{
		{Name: "song.mp3", IsFolder: false, FileID: 1, Modified: "not-a-date"},
	}

	var entries []fileEntry
	collectFiles(items, &entries, s, "/Music")

	require.Len(t, entries, 1)
	assert.True(t, entries[0].modified.IsZero(), "unparseable date should result in zero time")
}

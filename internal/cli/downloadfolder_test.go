package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildZip creates a zip archive at dest containing the given files.
// Each entry is a map of name -> content.
func buildZip(t *testing.T, dest string, files map[string]string) {
	t.Helper()
	f, err := os.Create(dest)
	require.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range files {
		fw, err := w.Create(name)
		require.NoError(t, err)
		_, err = fw.Write([]byte(content))
		require.NoError(t, err)
	}
}

func TestExtractZipToDir_Basic(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "archive.zip")
	buildZip(t, zipPath, map[string]string{
		"file.txt":       "hello",
		"sub/nested.txt": "world",
	})

	dest := filepath.Join(dir, "out")
	require.NoError(t, extractZipToDir(zipPath, dest))

	assert.FileExists(t, filepath.Join(dest, "file.txt"))
	assert.FileExists(t, filepath.Join(dest, filepath.Join("sub", "nested.txt")))
}

func TestExtractZipToDir_FileContent(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "archive.zip")
	buildZip(t, zipPath, map[string]string{"data.txt": "expected content"})

	dest := filepath.Join(dir, "out")
	require.NoError(t, extractZipToDir(zipPath, dest))

	got, err := os.ReadFile(filepath.Join(dest, "data.txt"))
	require.NoError(t, err)
	assert.Equal(t, "expected content", string(got))
}

// TestExtractZipToDir_PathTraversal verifies that zip slip attacks are rejected.
func TestExtractZipToDir_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")

	// Manually craft a zip with a path-traversal entry ("../../escape.txt").
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	fw, err := w.Create("../../escape.txt")
	require.NoError(t, err)
	_, err = fw.Write([]byte("pwned"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	err = extractZipToDir(zipPath, filepath.Join(dir, "out"))
	require.Error(t, err, "path traversal zip must be rejected")
}

func TestExtractZipToDir_InvalidZip(t *testing.T) {
	dir := t.TempDir()
	badZipPath := filepath.Join(dir, "bad.zip")
	require.NoError(t, os.WriteFile(badZipPath, []byte("this is not a zip"), 0o644))

	err := extractZipToDir(badZipPath, filepath.Join(dir, "out"))
	require.Error(t, err, "invalid zip should return an error")
}

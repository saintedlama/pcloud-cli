package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSyncArgs_ExplicitLocalDir(t *testing.T) {
	cloud, local := parseSyncArgs([]string{"/Music", "/home/user/music"})
	assert.Equal(t, "/Music", cloud)
	assert.Equal(t, "/home/user/music", local)
}

func TestParseSyncArgs_DeriveFromCloudPath(t *testing.T) {
	tests := []struct {
		name      string
		cloudPath string
		wantLocal string
	}{
		{"top-level dir", "/Music", "Music"},
		{"nested dir", "/Music/Rock", "Rock"},
		{"deeply nested", "/a/b/c", "c"},
		{"trailing slash stripped", "/Music/", "Music"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, local := parseSyncArgs([]string{tt.cloudPath})
			assert.Equal(t, tt.wantLocal, local)
		})
	}
}

func TestParseSyncArgs_FallbackName(t *testing.T) {
	// A bare slash yields "." or "/" from filepath.Base; fallback to "pcloud-sync".
	_, local := parseSyncArgs([]string{"/"})
	require.Equal(t, "pcloud-sync", local)
}

func TestParseUploadArgs_TwoArgs(t *testing.T) {
	local, cloud, err := parseUploadArgs([]string{"/pcloud/photos", "/home/user/photos"})
	require.NoError(t, err)
	assert.Equal(t, "/pcloud/photos", cloud)
	assert.Equal(t, "/home/user/photos", local)
}

func TestParseUploadArgs_DeriveCloudFromLocal(t *testing.T) {
	tests := []struct {
		name      string
		localArg  string
		wantCloud string
	}{
		{"absolute path", "/testdata/upload-fixture", "/upload-fixture"},
		{"simple name", "testdata", "/testdata"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			local, cloud, err := parseUploadArgs([]string{tt.localArg})
			require.NoError(t, err)
			assert.Equal(t, tt.wantCloud, cloud)
			absWant, _ := filepath.Abs(tt.localArg)
			assert.Equal(t, absWant, local)
		})
	}
}

func TestParseUploadArgs_DotResolvesToCwd(t *testing.T) {
	local, cloud, err := parseUploadArgs([]string{"."})
	require.NoError(t, err)
	absWant, _ := filepath.Abs(".")
	assert.Equal(t, absWant, local)
	assert.Equal(t, "/"+filepath.Base(absWant), cloud)
}

func TestParseUploadArgs_RootErrors(t *testing.T) {
	_, _, err := parseUploadArgs([]string{"/"})
	require.Error(t, err)
}

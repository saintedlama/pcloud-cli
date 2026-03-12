package cli

import (
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

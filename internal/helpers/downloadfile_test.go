package helpers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := []byte("hello download content")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(want)
		}))
		defer srv.Close()

		dest := filepath.Join(t.TempDir(), "out.txt")
		err := DownloadFile(srv.URL, dest)
		require.NoError(t, err)

		got, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("connection refused", func(t *testing.T) {
		// Close the server before downloading so the connection is refused.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		dest := filepath.Join(t.TempDir(), "out.txt")
		err := DownloadFile(url, dest)
		assert.Error(t, err)
	})

	t.Run("unwritable destination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("data"))
		}))
		defer srv.Close()

		err := DownloadFile(srv.URL, "/nonexistent/dir/out.txt")
		assert.Error(t, err)
	})
}

package pcloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- normalizePath ---------------------------------------------------------

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already prefixed", "/Music", "/Music"},
		{"missing slash", "Music", "/Music"},
		{"whitespace trimmed", "  Music  ", "/Music"},
		{"root", "/", "/"},
		{"empty string", "", "/"},
		{"nested without slash", "a/b/c", "/a/b/c"},
		{"nested with slash", "/a/b/c", "/a/b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizePath(tt.in))
		})
	}
}

// ---- IsConfigured ----------------------------------------------------------

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		token string
		want  bool
	}{
		{"both set", "https://api.pcloud.com", "tok123", true},
		{"no base url", "", "tok123", false},
		{"no token", "https://api.pcloud.com", "", false},
		{"whitespace base url", "  ", "tok123", false},
		{"whitespace token", "https://api.pcloud.com", "  ", false},
		{"both empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{BaseURL: tt.base, AuthToken: tt.token}
			assert.Equal(t, tt.want, api.IsConfigured())
		})
	}
}

// ---- Query -----------------------------------------------------------------

// newTestServer builds a minimal pCloud-like JSON response and returns a test
// server plus the API client pointing at it.
func newTestServer(t *testing.T, result int, extra map[string]any) (*httptest.Server, *API) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{"result": float64(result)}
		for k, v := range extra {
			body[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}))
	api := &API{BaseURL: srv.URL, AuthToken: "test-token"}
	return srv, api
}

func TestQuery_Success(t *testing.T) {
	srv, api := newTestServer(t, 0, map[string]any{"somekey": "val"})
	defer srv.Close()

	raw, err := api.Query(&Request{Endpoint: "/test", Parameters: url.Values{}})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "val", got["somekey"])
}

func TestQuery_APIError(t *testing.T) {
	srv, api := newTestServer(t, 2005, map[string]any{"error": "File or folder not found."})
	defer srv.Close()

	_, err := api.Query(&Request{Endpoint: "/test", Parameters: url.Values{}})
	require.Error(t, err)
	assert.ErrorContains(t, err, "2005")
}

func TestQuery_NoBaseURL(t *testing.T) {
	api := &API{}
	_, err := api.Query(&Request{Endpoint: "/test"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "base URL")
}

func TestQuery_AuthTokenAddedToQueryString(t *testing.T) {
	var capturedURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": float64(0)}) //nolint:errcheck
	}))
	defer srv.Close()

	api := &API{BaseURL: srv.URL, AuthToken: "mytoken"}
	_, err := api.Query(&Request{Endpoint: "/test", Parameters: url.Values{}})
	require.NoError(t, err)
	assert.Equal(t, "mytoken", capturedURL.Query().Get("auth"))
}

func TestQuery_NoAuthTokenWhenEmpty(t *testing.T) {
	var capturedURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": float64(0)}) //nolint:errcheck
	}))
	defer srv.Close()

	api := &API{BaseURL: srv.URL, AuthToken: ""}
	_, err := api.Query(&Request{Endpoint: "/test", Parameters: url.Values{}})
	require.NoError(t, err)
	assert.False(t, capturedURL.Query().Has("auth"), "auth param should be absent when token is empty")
}

// ---- LoginWithPassword validation ------------------------------------------

func TestLoginWithPassword_EmptyUsername(t *testing.T) {
	api := &API{BaseURL: "https://api.pcloud.com"}
	_, _, err := api.LoginWithPassword("", "secret")
	require.Error(t, err)
	assert.ErrorContains(t, err, "username")
}

func TestLoginWithPassword_EmptyPassword(t *testing.T) {
	api := &API{BaseURL: "https://api.pcloud.com"}
	_, _, err := api.LoginWithPassword("user@example.com", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "password")
}

// ---- GetZipLinkByFolderID validation ---------------------------------------

func TestGetZipLinkByFolderID_NegativeID(t *testing.T) {
	api := &API{BaseURL: "https://api.pcloud.com", AuthToken: "tok"}
	_, err := api.GetZipLinkByFolderID(-1, "archive.zip", false)
	require.Error(t, err)
}

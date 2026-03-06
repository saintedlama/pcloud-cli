package models

// GetziplinkResponse contains server response after getziplink call.
type GetziplinkResponse struct {
	Path    string   `json:"path"`
	Expires string   `json:"expires"`
	Hosts   []string `json:"hosts"`
}

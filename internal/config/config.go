package config

// File containing configuration data
type File struct {
	UserID      int    `json:"userid"`
	AccessToken string `json:"access_token"`
	BaseURL     string `json:"base_url,omitempty"`
}

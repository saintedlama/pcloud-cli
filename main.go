package main

import "github.com/storvik/pcloud-cli/commands"

var (
	// TODO: European pCloud users should use eapi
	// BaseURL to pCloud API
	BaseURL = "https://eapi.pcloud.com"
	// ClientID is pCloud ID of pcloud-cli
	ClientID = "wMJTDKXtja"
	// ClientSecret is secret key needed to identify app
	ClientSecret = "bCS3k9W89t0zL51qpcL2Ck3bjnF7"
)

func main() {
	commands.Execute(BaseURL, ClientID, ClientSecret)
}

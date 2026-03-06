package pcloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/storvik/pcloud-cli/internal/pcloud/models"
)

var (
	baseURL string
	verbose bool
)

func SetBaseURL(apiBaseURL string) {
	baseURL = apiBaseURL
}

func SetVerbose(enabled bool) {
	verbose = enabled
}

// API struct containing requests.
//
// Keep exported field names for existing CLI command usage.
//
//nolint:revive
type API struct {
	Endpoint    string
	Parameters  url.Values
	AccessToken string
	Body        io.Reader
	Headers     map[string]string
}

// NewAPI returns a new API struct.
func NewAPI() *API {
	return &API{Headers: make(map[string]string)}
}

// Query API endpoint with url parameters. If authorization is true, the authorization
// header is set. Returns json []byte and optional error from server.
func (p *API) Query() ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return []byte{}, errors.New("pCloud API base URL is not configured")
	}

	requestURL, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println("Error: Could not parse base url")
		os.Exit(1)
	}

	requestURL.Path += p.Endpoint
	requestURL.RawQuery = p.Parameters.Encode()

	if verbose {
		fmt.Println("Query path: " + requestURL.String())
	}

	request, err := http.NewRequest("POST", requestURL.String(), p.Body)
	for key, value := range p.Headers {
		request.Header.Add(key, value)
	}
	if p.AccessToken != "" {
		request.Header.Add("Authorization", "Bearer "+p.AccessToken)
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println("Error: Could not query endpoint")
		os.Exit(1)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Println("Response Status:", resp.Status)
	}

	var dat map[string]any
	if err := json.Unmarshal(responseBody, &dat); err != nil {
		panic(err)
	}

	if dat["result"].(float64) != 0 {
		return []byte{}, errors.New("Error " + strconv.FormatFloat(dat["result"].(float64), 'f', 0, 64) + ": " + dat["error"].(string))
	}

	return responseBody, nil
}

// Checksum fetches MD5 and SHA1 checksums for a remote file path.
func (p *API) Checksum(path, accessToken string) (models.ChecksumfileResponse, error) {
	if strings.TrimSpace(path) == "" {
		return models.ChecksumfileResponse{}, errors.New("path cannot be empty")
	}

	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/checksumfile"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.ChecksumfileResponse{}, err
	}

	var response models.ChecksumfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.ChecksumfileResponse{}, err
	}

	return response, nil
}

const (
	oauthClientID     = "wMJTDKXtja"
	oauthClientSecret = "bCS3k9W89t0zL51qpcL2Ck3bjnF7"
)

// OAuthURL returns the browser authorization URL for the OAuth2 flow.
func OAuthURL() string {
	return "https://my.pcloud.com/oauth2/authorize?client_id=" + oauthClientID + "&response_type=code"
}

type AuthorizeResponse struct {
	UserID      int
	AccessToken string
}

// Authorize exchanges an OAuth2 authorization code for an access token.
// tokenEndpoint should be the full base URL for the token exchange, e.g.
// "https://api.pcloud.com/oauth2_token" (US) or "https://eapi.pcloud.com/oauth2_token" (EU).
func (p *API) Authorize(tokenEndpoint, code string) (AuthorizeResponse, error) {
	parameters := url.Values{}
	parameters.Add("client_id", oauthClientID)
	parameters.Add("client_secret", oauthClientSecret)
	parameters.Add("code", strings.TrimSpace(code))

	requestURL := tokenEndpoint + "?" + parameters.Encode()
	if verbose {
		fmt.Println("Authorize URL: " + requestURL)
	}

	req, err := http.NewRequest("POST", requestURL, nil)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		return AuthorizeResponse{}, err
	}
	defer httpResp.Body.Close()

	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return AuthorizeResponse{}, err
	}
	if verbose {
		fmt.Println("Authorize response status:", httpResp.Status)
	}

	var dat map[string]interface{}
	if err := json.Unmarshal(resp, &dat); err != nil {
		return AuthorizeResponse{}, err
	}

	if result, ok := dat["result"].(float64); ok && result != 0 {
		errMsg, _ := dat["error"].(string)
		if strings.TrimSpace(errMsg) == "" {
			errMsg = "oauth2_token request failed"
		}
		return AuthorizeResponse{}, errors.New("Error " + strconv.FormatFloat(result, 'f', 0, 64) + ": " + errMsg)
	}

	var userID int
	if uid, ok := dat["uid"].(float64); ok {
		userID = int(uid)
	} else if uid, ok := dat["userid"].(float64); ok {
		userID = int(uid)
	} else {
		return AuthorizeResponse{}, errors.New("oauth2_token response missing uid")
	}

	accessToken, ok := dat["access_token"].(string)
	if !ok || strings.TrimSpace(accessToken) == "" {
		return AuthorizeResponse{}, errors.New("oauth2_token response missing access_token")
	}

	return AuthorizeResponse{
		UserID:      userID,
		AccessToken: accessToken,
	}, nil
}

func (p *API) GetFileLink(path, accessToken string) (models.GetfileResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/getfilelink"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.GetfileResponse{}, err
	}

	var response models.GetfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.GetfileResponse{}, err
	}

	return response, nil
}

func (p *API) UploadFile(localPath, remotePath string, renameIfExists bool, accessToken string) (models.UploadfileResponse, error) {
	fileContents, err := os.ReadFile(localPath)
	if err != nil {
		return models.UploadfileResponse{}, err
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("filename", filepath.Base(localPath))
	if err != nil {
		return models.UploadfileResponse{}, err
	}

	if _, err := part.Write(fileContents); err != nil {
		return models.UploadfileResponse{}, err
	}

	if err := writer.Close(); err != nil {
		return models.UploadfileResponse{}, err
	}

	parameters := url.Values{}
	parameters.Add("nopartial", "1")
	if renameIfExists {
		parameters.Add("renameifexists", "1")
	}
	if strings.TrimSpace(remotePath) == "" {
		parameters.Add("path", "/")
	} else {
		parameters.Add("path", normalizePath(remotePath))
	}

	p.Endpoint = "/uploadfile"
	p.Parameters = parameters
	p.AccessToken = accessToken
	p.Body = body
	p.Headers["Content-Type"] = writer.FormDataContentType()

	resp, err := p.Query()
	if err != nil {
		return models.UploadfileResponse{}, err
	}

	var response models.UploadfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.UploadfileResponse{}, err
	}

	return response, nil
}

func (p *API) CopyFile(sourcePath, destinationPath string, overwrite bool, accessToken string) (models.CopyfileResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(sourcePath))
	parameters.Add("topath", normalizePath(destinationPath))
	if !overwrite {
		parameters.Add("noover", "1")
	}

	p.Endpoint = "/copyfile"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.CopyfileResponse{}, err
	}

	var response models.CopyfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.CopyfileResponse{}, err
	}

	return response, nil
}

func (p *API) CreateFolder(path, accessToken string) (models.CreatefolderResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/createfolder"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.CreatefolderResponse{}, err
	}

	var response models.CreatefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.CreatefolderResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFile(path, accessToken string) (models.DeletefileResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/deletefile"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.DeletefileResponse{}, err
	}

	var response models.DeletefileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefileResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFolder(path, accessToken string) (models.DeletefolderResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/deletefolder"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.DeletefolderResponse{}, err
	}

	var response models.DeletefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefolderResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFolderRecursive(path, accessToken string) (models.DeletefolderRecursiveResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(path))

	p.Endpoint = "/deletefolderrecursive"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.DeletefolderRecursiveResponse{}, err
	}

	var response models.DeletefolderRecursiveResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefolderRecursiveResponse{}, err
	}

	return response, nil
}

func (p *API) ListFolder(path string, nofiles, showdeleted bool, accessToken string) (models.ListfolderResponse, error) {
	parameters := url.Values{}
	if strings.TrimSpace(path) == "" {
		parameters.Add("path", "/")
	} else {
		parameters.Add("path", normalizePath(path))
	}
	if nofiles {
		parameters.Add("nofiles", "1")
	}
	if showdeleted {
		parameters.Add("showdeleted", "1")
	}

	p.Endpoint = "/listfolder"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.ListfolderResponse{}, err
	}

	var response models.ListfolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.ListfolderResponse{}, err
	}

	return response, nil
}

func (p *API) RenameFile(sourcePath, destinationPath, accessToken string) (models.RenamefileResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(sourcePath))
	parameters.Add("topath", normalizePath(destinationPath))

	p.Endpoint = "/renamefile"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.RenamefileResponse{}, err
	}

	var response models.RenamefileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.RenamefileResponse{}, err
	}

	return response, nil
}

func (p *API) RenameFolder(sourcePath, destinationPath, accessToken string) (models.RenamefolderResponse, error) {
	parameters := url.Values{}
	parameters.Add("path", normalizePath(sourcePath))
	parameters.Add("topath", normalizePath(destinationPath))

	p.Endpoint = "/renamefolder"
	p.Parameters = parameters
	p.AccessToken = accessToken

	resp, err := p.Query()
	if err != nil {
		return models.RenamefolderResponse{}, err
	}

	var response models.RenamefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.RenamefolderResponse{}, err
	}

	return response, nil
}

func (p *API) GetZipLinkByFolderID(folderID int, filename string, forceDownload bool, accessToken string) (models.GetziplinkResponse, error) {
	if folderID < 0 {
		return models.GetziplinkResponse{}, errors.New("folder id must be >= 0")
	}

	parameters := url.Values{}
	parameters.Add("folderid", strconv.Itoa(folderID))
	if strings.TrimSpace(filename) != "" {
		parameters.Add("filename", filename)
	}
	if forceDownload {
		parameters.Add("forcedownload", "1")
	}

	p.Endpoint = "/getziplink"
	p.Parameters = parameters
	p.AccessToken = accessToken
	p.Body = nil

	resp, err := p.Query()
	if err != nil {
		return models.GetziplinkResponse{}, err
	}

	var response models.GetziplinkResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.GetziplinkResponse{}, err
	}

	return response, nil
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return "/" + trimmed
}

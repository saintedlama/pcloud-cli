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

const (
	oauthClientID     = "wMJTDKXtja"
	oauthClientSecret = "bCS3k9W89t0zL51qpcL2Ck3bjnF7"
)

// API is a pCloud client holding session-level configuration.
type API struct {
	BaseURL     string
	AccessToken string
}

// Request holds the per-call data passed to Query.
type Request struct {
	Endpoint   string
	Parameters url.Values
	Body       io.Reader
	Headers    map[string]string
}

// NewAPI returns a new API client.
func NewAPI() *API {
	return &API{}
}

// Query executes req against the API and returns the raw JSON response body.
func (p *API) Query(req *Request) ([]byte, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return []byte{}, errors.New("pCloud API base URL is not configured")
	}

	requestURL, err := url.Parse(p.BaseURL)
	if err != nil {
		return []byte{}, fmt.Errorf("could not parse base URL: %w", err)
	}

	requestURL.Path += req.Endpoint
	requestURL.RawQuery = req.Parameters.Encode()

	httpReq, err := http.NewRequest("POST", requestURL.String(), req.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("could not create request: %w", err)
	}
	for key, value := range req.Headers {
		httpReq.Header.Add(key, value)
	}
	if p.AccessToken != "" {
		httpReq.Header.Add("Authorization", "Bearer "+p.AccessToken)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return []byte{}, fmt.Errorf("could not query endpoint: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("could not read response body: %w", err)
	}

	var dat map[string]any
	if err := json.Unmarshal(responseBody, &dat); err != nil {
		return []byte{}, fmt.Errorf("could not parse response JSON: %w", err)
	}

	if dat["result"].(float64) != 0 {
		return []byte{}, errors.New("Error " + strconv.FormatFloat(dat["result"].(float64), 'f', 0, 64) + ": " + dat["error"].(string))
	}

	return responseBody, nil
}

// Checksum fetches MD5 and SHA1 checksums for a remote file path.
func (p *API) Checksum(path string) (models.ChecksumfileResponse, error) {
	if strings.TrimSpace(path) == "" {
		return models.ChecksumfileResponse{}, errors.New("path cannot be empty")
	}

	req := &Request{
		Endpoint:   "/checksumfile",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.ChecksumfileResponse{}, err
	}

	var response models.ChecksumfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.ChecksumfileResponse{}, err
	}

	return response, nil
}

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

func (p *API) GetFileLink(path string) (models.GetfileResponse, error) {
	req := &Request{
		Endpoint:   "/getfilelink",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.GetfileResponse{}, err
	}

	var response models.GetfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.GetfileResponse{}, err
	}

	return response, nil
}

func (p *API) UploadFile(localPath, remotePath string, renameIfExists bool) (models.UploadfileResponse, error) {
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

	parameters := url.Values{"nopartial": {"1"}}
	if renameIfExists {
		parameters.Add("renameifexists", "1")
	}
	if strings.TrimSpace(remotePath) == "" {
		parameters.Add("path", "/")
	} else {
		parameters.Add("path", normalizePath(remotePath))
	}

	req := &Request{
		Endpoint:   "/uploadfile",
		Parameters: parameters,
		Body:       body,
		Headers:    map[string]string{"Content-Type": writer.FormDataContentType()},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.UploadfileResponse{}, err
	}

	var response models.UploadfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.UploadfileResponse{}, err
	}

	return response, nil
}

func (p *API) CopyFile(sourcePath, destinationPath string, overwrite bool) (models.CopyfileResponse, error) {
	parameters := url.Values{
		"path":   {normalizePath(sourcePath)},
		"topath": {normalizePath(destinationPath)},
	}
	if !overwrite {
		parameters.Add("noover", "1")
	}

	req := &Request{
		Endpoint:   "/copyfile",
		Parameters: parameters,
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.CopyfileResponse{}, err
	}

	var response models.CopyfileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.CopyfileResponse{}, err
	}

	return response, nil
}

func (p *API) CreateFolder(path string) (models.CreatefolderResponse, error) {
	req := &Request{
		Endpoint:   "/createfolder",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.CreatefolderResponse{}, err
	}

	var response models.CreatefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.CreatefolderResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFile(path string) (models.DeletefileResponse, error) {
	req := &Request{
		Endpoint:   "/deletefile",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.DeletefileResponse{}, err
	}

	var response models.DeletefileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefileResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFolder(path string) (models.DeletefolderResponse, error) {
	req := &Request{
		Endpoint:   "/deletefolder",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.DeletefolderResponse{}, err
	}

	var response models.DeletefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefolderResponse{}, err
	}

	return response, nil
}

func (p *API) DeleteFolderRecursive(path string) (models.DeletefolderRecursiveResponse, error) {
	req := &Request{
		Endpoint:   "/deletefolderrecursive",
		Parameters: url.Values{"path": {normalizePath(path)}},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.DeletefolderRecursiveResponse{}, err
	}

	var response models.DeletefolderRecursiveResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.DeletefolderRecursiveResponse{}, err
	}

	return response, nil
}

func (p *API) ListFolder(path string, nofiles, showdeleted bool) (models.ListfolderResponse, error) {
	if strings.TrimSpace(path) == "" {
		path = "/"
	} else {
		path = normalizePath(path)
	}
	parameters := url.Values{"path": {path}}
	if nofiles {
		parameters.Add("nofiles", "1")
	}
	if showdeleted {
		parameters.Add("showdeleted", "1")
	}

	req := &Request{
		Endpoint:   "/listfolder",
		Parameters: parameters,
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.ListfolderResponse{}, err
	}

	var response models.ListfolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.ListfolderResponse{}, err
	}

	return response, nil
}

func (p *API) RenameFile(sourcePath, destinationPath string) (models.RenamefileResponse, error) {
	req := &Request{
		Endpoint: "/renamefile",
		Parameters: url.Values{
			"path":   {normalizePath(sourcePath)},
			"topath": {normalizePath(destinationPath)},
		},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.RenamefileResponse{}, err
	}

	var response models.RenamefileResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.RenamefileResponse{}, err
	}

	return response, nil
}

func (p *API) RenameFolder(sourcePath, destinationPath string) (models.RenamefolderResponse, error) {
	req := &Request{
		Endpoint: "/renamefolder",
		Parameters: url.Values{
			"path":   {normalizePath(sourcePath)},
			"topath": {normalizePath(destinationPath)},
		},
	}

	resp, err := p.Query(req)
	if err != nil {
		return models.RenamefolderResponse{}, err
	}

	var response models.RenamefolderResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return models.RenamefolderResponse{}, err
	}

	return response, nil
}

func (p *API) GetZipLinkByFolderID(folderID int, filename string, forceDownload bool) (models.GetziplinkResponse, error) {
	if folderID < 0 {
		return models.GetziplinkResponse{}, errors.New("folder id must be >= 0")
	}

	parameters := url.Values{"folderid": {strconv.Itoa(folderID)}}
	if strings.TrimSpace(filename) != "" {
		parameters.Add("filename", filename)
	}
	if forceDownload {
		parameters.Add("forcedownload", "1")
	}

	req := &Request{
		Endpoint:   "/getziplink",
		Parameters: parameters,
	}

	resp, err := p.Query(req)
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

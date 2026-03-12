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

	"github.com/saintedlama/pcloud-cli/internal/pcloud/models"
)

// API is a pCloud client holding session-level configuration.
type API struct {
	BaseURL   string
	AuthToken string
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

// IsConfigured reports whether the client has both a base URL and an auth token.
func (p *API) IsConfigured() bool {
	return strings.TrimSpace(p.BaseURL) != "" && strings.TrimSpace(p.AuthToken) != ""
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
	params := req.Parameters
	if params == nil {
		params = url.Values{}
	}
	if p.AuthToken != "" {
		params.Set("auth", p.AuthToken)
	}
	requestURL.RawQuery = params.Encode()

	httpReq, err := http.NewRequest("POST", requestURL.String(), req.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("could not create request: %w", err)
	}
	for key, value := range req.Headers {
		httpReq.Header.Add(key, value)
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

// UserinfoResponse holds the fields returned by /userinfo when called with getauth=1.
type UserinfoResponse struct {
	UserID          int    `json:"userid"`
	Email           string `json:"email"`
	Auth            string `json:"auth"`
	Quota           int64  `json:"quota"`
	UsedQuota       int64  `json:"usedquota"`
	Plan            int    `json:"plan"`
	Premium         bool   `json:"premium"`
	PremiumLifetime bool   `json:"premiumlifetime"`
	PremiumExpires  string `json:"premiumexpires"`
	Currency        string `json:"currency"`
}

// GetUserInfo fetches account information for the authenticated user.
func (p *API) GetUserInfo() (UserinfoResponse, error) {
	req := &Request{
		Endpoint:   "/userinfo",
		Parameters: url.Values{},
	}

	raw, err := p.Query(req)
	if err != nil {
		return UserinfoResponse{}, err
	}

	var response UserinfoResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return UserinfoResponse{}, err
	}

	return response, nil
}

// LoginWithPassword authenticates with username+password and returns the parsed
// response, the raw JSON bytes (useful for debugging), and any error.
// A bare API client (no Bearer token) is used deliberately: if an OAuth
// Authorization header is present, pCloud validates the existing session instead
// of issuing a new auth session token, leaving the `auth` field empty.
func (p *API) LoginWithPassword(username, password string) (UserinfoResponse, []byte, error) {
	if strings.TrimSpace(username) == "" {
		return UserinfoResponse{}, nil, errors.New("username cannot be empty")
	}
	if strings.TrimSpace(password) == "" {
		return UserinfoResponse{}, nil, errors.New("password cannot be empty")
	}

	// Use a bare client with no AccessToken so Query() does not attach an
	// Authorization: Bearer header. pCloud only returns `auth` when the request
	// is authenticated purely by username+password.
	bare := &API{BaseURL: p.BaseURL}

	req := &Request{
		Endpoint: "/userinfo",
		Parameters: url.Values{
			"username": {username},
			"password": {password},
			"getauth":  {"1"},
			"logout":   {"1"},
		},
	}

	raw, err := bare.Query(req)
	if err != nil {
		return UserinfoResponse{}, nil, err
	}

	var response UserinfoResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return UserinfoResponse{}, raw, err
	}

	return response, raw, nil
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

// GetFileLinkByID returns a download link for the file identified by its numeric file ID.
func (p *API) GetFileLinkByID(fileID int) (models.GetfileResponse, error) {
	req := &Request{
		Endpoint:   "/getfilelink",
		Parameters: url.Values{"fileid": {strconv.Itoa(fileID)}},
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

// ListFolderOptions controls optional parameters for the listfolder API call.
type ListFolderOptions struct {
	// Recursive returns the full directory tree when true (recursive=1).
	Recursive bool
	// ShowDeleted includes deleted files and folders that can be undeleted.
	ShowDeleted bool
	// NoFiles returns only the folder (sub)structure, omitting files.
	NoFiles bool
	// NoShares returns only the user's own folders and files, hiding shared items.
	NoShares bool
}

func (p *API) ListFolder(path string, opts ListFolderOptions) (models.ListfolderResponse, error) {
	if strings.TrimSpace(path) == "" {
		path = "/"
	} else {
		path = normalizePath(path)
	}
	parameters := url.Values{"path": {path}}
	if opts.Recursive {
		parameters.Add("recursive", "1")
	}
	if opts.ShowDeleted {
		parameters.Add("showdeleted", "1")
	}
	if opts.NoFiles {
		parameters.Add("nofiles", "1")
	}
	if opts.NoShares {
		parameters.Add("noshares", "1")
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

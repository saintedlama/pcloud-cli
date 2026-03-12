package models

// FolderItem represents a single file or sub-folder entry in a listing.
// When ListFolder is called with Recursive=true the Contents field is populated
// for sub-folders, allowing the entire tree to be traversed in one API call.
type FolderItem struct {
	Path           string       `json:"path"`
	Name           string       `json:"name"`
	Modified       string       `json:"modified"`
	Size           int64        `json:"size"`
	IsMine         bool         `json:"ismine"`
	ID             string       `json:"id"`
	IsShared       bool         `json:"isshared"`
	IsFolder       bool         `json:"isfolder"`
	ParentFolderID int          `json:"parentfolderid"`
	FolderID       int          `json:"folderid"`
	FileID         int          `json:"fileid"`
	Contents       []FolderItem `json:"contents"` // only populated with Recursive=true
}

// ListfolderResponse contains server response after listfolder call
type ListfolderResponse struct {
	Metadata struct {
		Path           string       `json:"path"`
		Name           string       `json:"name"`
		Modified       string       `json:"modified"`
		IsMine         bool         `json:"ismine"`
		ID             string       `json:"id"`
		IsShared       bool         `json:"isshared"`
		IsFolder       bool         `json:"isfolder"`
		ParentFolderID int          `json:"parentfolderid"`
		FolderID       int          `json:"folderid"`
		Contents       []FolderItem `json:"contents"`
	} `json:"metadata"`
}

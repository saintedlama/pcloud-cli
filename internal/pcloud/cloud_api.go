package pcloud

import "github.com/saintedlama/pcloud-cli/internal/pcloud/models"

// CloudAPI is the subset of pCloud API operations required by the TUI layer.
// *API satisfies this interface; tests can supply a StubAPI instead.
type CloudAPI interface {
	ListFolder(path string, opts ListFolderOptions) (models.ListfolderResponse, error)
	DeleteFile(path string) (models.DeletefileResponse, error)
	DeleteFolderRecursive(path string) (models.DeletefolderRecursiveResponse, error)
	RenameFile(sourcePath, destinationPath string) (models.RenamefileResponse, error)
	RenameFolder(sourcePath, destinationPath string) (models.RenamefolderResponse, error)
	GetFileLink(path string) (models.GetfileResponse, error)
	GetZipLinkByFolderID(folderID int, filename string, forceDownload bool) (models.GetziplinkResponse, error)
}

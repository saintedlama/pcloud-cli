// Package pcloudtest provides test helpers for the pcloud package,
// following the net/http → net/http/httptest convention.
package pcloudtest

import (
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/pcloud/models"
)

// StubAPI is a configurable fake that satisfies pcloud.CloudAPI.
// Set the Err* fields to inject errors and *Result fields to control return
// payloads. The *Calls slices record arguments for test assertions.
type StubAPI struct {
	ListFolderResult models.ListfolderResponse
	ListFolderErr    error

	DeleteFileErr   error
	DeleteFolderErr error

	RenameFileErr   error
	RenameFolderErr error

	GetFileLinkResult models.GetfileResponse
	GetFileLinkErr    error

	GetZipLinkResult models.GetziplinkResponse
	GetZipLinkErr    error

	// call recorders
	ListFolderCalls   []string
	DeleteFileCalls   []string
	DeleteFolderCalls []string
	RenameFileCalls   [][2]string
	RenameFolderCalls [][2]string
	GetFileLinkCalls  []string
}

func (s *StubAPI) ListFolder(path string, opts pcloud.ListFolderOptions) (models.ListfolderResponse, error) {
	s.ListFolderCalls = append(s.ListFolderCalls, path)
	return s.ListFolderResult, s.ListFolderErr
}

func (s *StubAPI) DeleteFile(path string) (models.DeletefileResponse, error) {
	s.DeleteFileCalls = append(s.DeleteFileCalls, path)
	return models.DeletefileResponse{}, s.DeleteFileErr
}

func (s *StubAPI) DeleteFolderRecursive(path string) (models.DeletefolderRecursiveResponse, error) {
	s.DeleteFolderCalls = append(s.DeleteFolderCalls, path)
	return models.DeletefolderRecursiveResponse{}, s.DeleteFolderErr
}

func (s *StubAPI) RenameFile(src, dst string) (models.RenamefileResponse, error) {
	s.RenameFileCalls = append(s.RenameFileCalls, [2]string{src, dst})
	return models.RenamefileResponse{}, s.RenameFileErr
}

func (s *StubAPI) RenameFolder(src, dst string) (models.RenamefolderResponse, error) {
	s.RenameFolderCalls = append(s.RenameFolderCalls, [2]string{src, dst})
	return models.RenamefolderResponse{}, s.RenameFolderErr
}

func (s *StubAPI) GetFileLink(path string) (models.GetfileResponse, error) {
	s.GetFileLinkCalls = append(s.GetFileLinkCalls, path)
	return s.GetFileLinkResult, s.GetFileLinkErr
}

func (s *StubAPI) GetZipLinkByFolderID(folderID int, filename string, forceDownload bool) (models.GetziplinkResponse, error) {
	return s.GetZipLinkResult, s.GetZipLinkErr
}

package msgs

import tea "github.com/charmbracelet/bubbletea"

// FolderLoadedMsg is sent when a folder listing has been fetched from the API.
type FolderLoadedMsg struct {
	Path  string
	Items []Entry
}

// Entry is a single file or folder returned by the API.
type Entry struct {
	Name     string
	Path     string
	IsFolder bool
	Size     int64
	Modified string
}

// DownloadDoneMsg is sent when a file has been downloaded successfully.
type DownloadDoneMsg struct {
	LocalPath string
}

// ShowDialogMsg is returned by a component to request a dialog overlay.
// Content is the tea.Model that renders the dialog body.
type ShowDialogMsg struct {
	Content tea.Model
}

// CloseDialogMsg is returned by a dialog model when it should be dismissed.
// Result carries an optional value back to the parent (may be nil).
type CloseDialogMsg struct {
	Result any
}

// ErrMsg is sent when an API call fails.
type ErrMsg struct {
	Err error
}

func (e ErrMsg) Error() string { return e.Err.Error() }

package filebrowser

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saintedlama/pcloud-cli/internal/helpers"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/tui/msgs"
)

// DownloadDialog is a standalone tea.Model for the file-download prompt.
type DownloadDialog struct {
	input       textinput.Model
	spinner     spinner.Model
	api         *pcloud.API
	entry       msgs.Entry
	downloading bool
	done        bool
	localPath   string
	err         error
}

// NewDownloadDialog creates a download dialog for the given file entry.
func NewDownloadDialog(api *pcloud.API, entry msgs.Entry) DownloadDialog {
	ti := textinput.New()
	ti.CharLimit = 255
	ti.Width = 40
	ti.Placeholder = "filename"
	ti.SetValue(entry.Name)

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return DownloadDialog{
		input:   ti,
		spinner: s,
		api:     api,
		entry:   entry,
	}
}

func (m DownloadDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m DownloadDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		// Any key after "done" dismisses the dialog.
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		}
	}

	if m.downloading {
		switch msg := msg.(type) {
		case msgs.DownloadDoneMsg:
			m.downloading = false
			m.done = true
			m.localPath = msg.LocalPath
			return m, nil
		case msgs.ErrMsg:
			m.downloading = false
			m.err = msg.Err
			return m, nil
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyMsg); ok {
		switch kMsg.String() {
		case "enter":
			localPath := m.input.Value()
			if localPath == "" {
				localPath = m.entry.Name
			}
			m.input.Blur()
			m.downloading = true
			return m, tea.Batch(m.spinner.Tick, downloadFile(m.api, m.entry.Path, localPath))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m DownloadDialog) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("pCloud") + "  ")
	sb.WriteString(dialogTitleStyle.Render("Download File"))
	sb.WriteString("\n\n")
	sb.WriteString("  Source:   ")
	sb.WriteString(pathStyle.Render(m.entry.Path))
	sb.WriteString("\n\n")

	if m.done {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  Downloaded: %s", m.localPath)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		// Allow dismiss on next keypress.
		m.done = true
		return sb.String()
	}

	if m.downloading {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString(fmt.Sprintf("  Downloading %s…", m.entry.Name))
		return sb.String()
	}

	sb.WriteString("  Save as:  ")
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("  Enter to confirm  |  Esc to cancel"))
	return sb.String()
}

// downloadFile returns a command that fetches a download link and saves the file locally.
func downloadFile(api *pcloud.API, remotePath, localPath string) tea.Cmd {
	return func() tea.Msg {
		link, err := api.GetFileLink(remotePath)
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		if len(link.Hosts) == 0 {
			return msgs.ErrMsg{Err: fmt.Errorf("no download hosts returned for %s", remotePath)}
		}
		fileURL := "http://" + link.Hosts[0] + link.Path
		if err := helpers.DownloadFile(fileURL, localPath); err != nil {
			return msgs.ErrMsg{Err: err}
		}
		return msgs.DownloadDoneMsg{LocalPath: localPath}
	}
}

// FolderDownloadDialog downloads a remote folder as a zip and extracts it locally.
type FolderDownloadDialog struct {
	input       textinput.Model
	spinner     spinner.Model
	api         *pcloud.API
	entry       msgs.Entry
	downloading bool
	done        bool
	localPath   string
	err         error
}

// NewFolderDownloadDialog creates a download dialog for the given folder entry.
func NewFolderDownloadDialog(api *pcloud.API, entry msgs.Entry) FolderDownloadDialog {
	ti := textinput.New()
	ti.CharLimit = 255
	ti.Width = 40
	ti.Placeholder = "destination directory"
	ti.SetValue(".")

	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return FolderDownloadDialog{
		input:   ti,
		spinner: s,
		api:     api,
		entry:   entry,
	}
}

func (m FolderDownloadDialog) Init() tea.Cmd {
	return m.input.Focus()
}

func (m FolderDownloadDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done || m.err != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, func() tea.Msg { return msgs.CloseDialogMsg{} }
		}
	}

	if m.downloading {
		switch msg := msg.(type) {
		case msgs.DownloadDoneMsg:
			m.downloading = false
			m.done = true
			m.localPath = msg.LocalPath
			return m, nil
		case msgs.ErrMsg:
			m.downloading = false
			m.err = msg.Err
			return m, nil
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if kMsg, ok := msg.(tea.KeyMsg); ok {
		if kMsg.String() == "enter" {
			destDir := m.input.Value()
			if destDir == "" {
				destDir = "."
			}
			m.input.Blur()
			m.downloading = true
			return m, tea.Batch(m.spinner.Tick, downloadFolder(m.api, m.entry, destDir))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m FolderDownloadDialog) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("pCloud") + "  ")
	sb.WriteString(dialogTitleStyle.Render("Download Folder"))
	sb.WriteString("\n\n")
	sb.WriteString("  Source:      ")
	sb.WriteString(pathStyle.Render(m.entry.Path))
	sb.WriteString("\n\n")

	if m.done {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  Downloaded: %s", m.localPath)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("  Press any key to continue"))
		return sb.String()
	}

	if m.downloading {
		sb.WriteString("  ")
		sb.WriteString(m.spinner.View())
		sb.WriteString(fmt.Sprintf("  Downloading %s…", m.entry.Name))
		return sb.String()
	}

	sb.WriteString("  Save to:     ")
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("  Enter to confirm  |  Esc to cancel"))
	return sb.String()
}

// downloadFolder fetches the folder as a zip via getziplink and extracts it locally.
func downloadFolder(api *pcloud.API, entry msgs.Entry, destDir string) tea.Cmd {
	return func() tea.Msg {
		folderData, err := api.ListFolder(entry.Path, true, false)
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}

		folderName := strings.TrimSpace(folderData.Metadata.Name)
		if folderName == "" {
			folderName = entry.Name
		}

		zipLink, err := api.GetZipLinkByFolderID(folderData.Metadata.FolderID, folderName+".zip", false)
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		if len(zipLink.Hosts) == 0 {
			return msgs.ErrMsg{Err: fmt.Errorf("no download hosts returned for %s", entry.Path)}
		}

		downloadURL := "https://" + zipLink.Hosts[0] + zipLink.Path

		resp, err := http.Get(downloadURL) // #nosec G107 — URL comes from pCloud API response
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return msgs.ErrMsg{Err: fmt.Errorf("download failed with status: %s", resp.Status)}
		}

		tmpFile, err := os.CreateTemp("", "pcloud-folder-*.zip")
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			tmpFile.Close()
			return msgs.ErrMsg{Err: err}
		}
		tmpFile.Close()

		targetDir := filepath.Join(destDir, folderName)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return msgs.ErrMsg{Err: err}
		}

		if err := extractZip(tmpPath, targetDir); err != nil {
			return msgs.ErrMsg{Err: err}
		}

		return msgs.DownloadDoneMsg{LocalPath: targetDir}
	}
}

func extractZip(zipPath, destination string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanDest := filepath.Clean(destination)
	prefix := cleanDest + string(os.PathSeparator)

	for _, f := range reader.File {
		target := filepath.Join(cleanDest, f.Name)
		clean := filepath.Clean(target)
		// Guard against zip slip.
		if clean != cleanDest && !strings.HasPrefix(clean, prefix) {
			return fmt.Errorf("invalid archive path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(clean, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
			return err
		}

		src, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(clean, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		srcErr := src.Close()
		dstErr := dst.Close()

		if copyErr != nil {
			return copyErr
		}
		if srcErr != nil {
			return srcErr
		}
		if dstErr != nil {
			return dstErr
		}
	}

	return nil
}

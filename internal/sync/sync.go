// Package sync provides a one-way sync engine that mirrors a pCloud directory
// tree to a local directory. Only files that are newer on pCloud are downloaded.
// Files deleted from pCloud are removed locally.
package sync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/saintedlama/pcloud-cli/internal/pcloud/models"
)

var (
	indexStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // bright yellow
	totalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("130")) // orange-brown
	sizeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))  // cyan
	speedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("32"))  // green
)

// Syncer mirrors a pCloud directory tree to a local directory.
type Syncer struct {
	api       *pcloud.API
	cloudRoot string // normalised (no trailing slash)
	localRoot string
	log       io.Writer
}

type fileEntry struct {
	cloudPath string
	fileID    int    // pCloud numeric file ID; used for GetFileLinkByID
	localRel  string // relative to localRoot, OS path separators
	modified  time.Time
}

// New creates a Syncer. cloudRoot is the pCloud source path; localRoot is the
// destination directory on disk. Progress messages are written to log.
func New(api *pcloud.API, cloudRoot, localRoot string, log io.Writer) *Syncer {
	return &Syncer{
		api:       api,
		cloudRoot: strings.TrimRight(cloudRoot, "/"),
		localRoot: localRoot,
		log:       log,
	}
}

// RunResult summarises the outcome of a single sync pass.
type RunResult struct {
	Total      int // total files found on pCloud
	Downloaded int // files downloaded (new or updated)
	Deleted    int // local files removed (no longer on pCloud)
	Warnings   int // files that failed with a non-fatal error
}

// Run performs one sync pass and returns when it is finished.
func (s *Syncer) Run(ctx context.Context) (RunResult, error) {
	fmt.Fprintf(s.log, "Scanning %s ...\n", s.cloudRoot)

	var remote []fileEntry
	if err := s.walk(ctx, s.cloudRoot, &remote); err != nil {
		return RunResult{}, fmt.Errorf("listing pCloud tree: %w", err)
	}

	remoteSet := make(map[string]struct{}, len(remote))
	for _, e := range remote {
		remoteSet[e.localRel] = struct{}{}
	}

	// Count local files and determine which remote files need downloading.
	localCount := s.countLocalFiles()
	var toDownload []fileEntry
	for _, e := range remote {
		if s.needsDownload(e) {
			toDownload = append(toDownload, e)
		}
	}
	fmt.Fprintf(s.log, "Found %d file(s) on pCloud, found %d local\n",
		len(remote), localCount)

	var res RunResult
	res.Total = len(remote)

	for i, e := range toDownload {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		downloaded, err := s.syncFile(ctx, e, i+1, len(toDownload))
		if err != nil {
			fmt.Fprintf(s.log, "warn: %s: %v\n", e.cloudPath, err)
			res.Warnings++
		} else if downloaded {
			res.Downloaded++
		}
	}

	deleted, err := s.pruneLocal(remoteSet)
	res.Deleted = deleted
	if err != nil {
		return res, err
	}

	return res, nil
}

// Watch calls Run repeatedly, sleeping interval between passes, until ctx is
// cancelled. It returns nil on clean shutdown.
func (s *Syncer) Watch(ctx context.Context, interval time.Duration) error {
	for {
		start := time.Now()
		fmt.Fprintf(s.log, "\n[%s] === sync start  %s -> %s ===\n",
			start.Format(time.RFC3339), s.cloudRoot, s.localRoot)

		res, err := s.Run(ctx)
		if ctx.Err() != nil {
			return nil
		} else if err != nil {
			fmt.Fprintf(s.log, "[%s] sync error: %v\n",
				time.Now().Format(time.RFC3339), err)
		} else {
			fmt.Fprintf(s.log,
				"[%s] sync done   %d downloaded, %d deleted, %d up to date  (%.1fs)\n",
				time.Now().Format(time.RFC3339),
				res.Downloaded, res.Deleted, res.Total-res.Downloaded-res.Warnings,
				time.Since(start).Seconds())
		}

		fmt.Fprintf(s.log, "Next sync in %s\n", interval)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// walk fetches the full tree from pCloud in a single recursive API call and
// appends every file entry to entries.
func (s *Syncer) walk(ctx context.Context, cloudPath string, entries *[]fileEntry) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	resp, err := s.api.ListFolder(cloudPath, pcloud.ListFolderOptions{Recursive: true})
	if err != nil {
		return fmt.Errorf("listfolder %s: %w", cloudPath, err)
	}

	collectFiles(resp.Metadata.Contents, entries, s, strings.TrimRight(resp.Metadata.Path, "/"))
	return nil
}

// collectFiles recursively traverses the tree returned by a recursive listfolder
// call and appends file entries to the slice.
// parentPath is the absolute pCloud path of the folder whose Contents are being iterated;
// it is used to construct full paths for items that lack a populated Path field.
func collectFiles(items []models.FolderItem, entries *[]fileEntry, s *Syncer, parentPath string) {
	for _, item := range items {
		// Build the full path from parent + name so we never depend on
		// item.Path, which pCloud may omit for deeply nested items in a
		// recursive listing.
		fullPath := parentPath + "/" + item.Name
		if item.IsFolder {
			collectFiles(item.Contents, entries, s, fullPath)
		} else {
			modified, _ := time.Parse(time.RFC1123Z, item.Modified)
			*entries = append(*entries, fileEntry{
				cloudPath: fullPath,
				fileID:    item.FileID,
				localRel:  s.toLocalRel(fullPath),
				modified:  modified,
			})
		}
	}
}

// toLocalRel converts an absolute pCloud path to a path relative to localRoot.
// e.g. cloudRoot="/Music", cloudPath="/Music/Rock/song.mp3" -> "Rock/song.mp3"
// countLocalFiles counts the number of regular files under localRoot.
// Returns 0 if the directory does not exist yet.
func (s *Syncer) countLocalFiles() int {
	count := 0
	_ = filepath.WalkDir(s.localRoot, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// needsDownload reports whether the remote entry is absent or newer than the
// corresponding local file.
func (s *Syncer) needsDownload(e fileEntry) bool {
	localPath := filepath.Join(s.localRoot, e.localRel)
	info, err := os.Stat(localPath)
	if err != nil {
		return true // file absent
	}
	if e.modified.IsZero() {
		return false
	}
	return info.ModTime().Before(e.modified)
}

func (s *Syncer) toLocalRel(cloudPath string) string {
	rel := strings.TrimPrefix(cloudPath, s.cloudRoot)
	rel = strings.TrimPrefix(rel, "/")
	return filepath.FromSlash(rel)
}

// syncFile downloads the remote file if it is absent or newer than the local copy.
// Returns (true, nil) when the file was downloaded, (false, nil) when already up to date.
func (s *Syncer) syncFile(ctx context.Context, e fileEntry, idx, total int) (bool, error) {
	localPath := filepath.Join(s.localRoot, e.localRel)

	if info, err := os.Stat(localPath); err == nil && !e.modified.IsZero() {
		if !info.ModTime().Before(e.modified) {
			return false, nil // local file is current
		}
	}

	link, err := s.api.GetFileLinkByID(e.fileID)
	if err != nil {
		return false, fmt.Errorf("get file link: %w", err)
	}
	if len(link.Hosts) == 0 {
		return false, fmt.Errorf("no download hosts returned for %s", e.cloudPath)
	}
	downloadURL := "https://" + link.Hosts[0] + link.Path

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return false, fmt.Errorf("create directories: %w", err)
	}

	// Atomic write: download into a sibling temp file, then rename.
	tmp, err := os.CreateTemp(filepath.Dir(localPath), ".pcloud-sync-*")
	if err != nil {
		return false, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		tmp.Close()
		if !committed {
			os.Remove(tmpName)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil) //nolint:gosec // URL from trusted pCloud API
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	start := time.Now()
	n, err := io.Copy(tmp, resp.Body)
	if err != nil {
		return false, fmt.Errorf("write temp file: %w", err)
	}
	elapsed := time.Since(start)
	if elapsed < time.Millisecond {
		elapsed = time.Millisecond
	}

	speedMBs := float64(n) / elapsed.Seconds() / (1024 * 1024)
	fmt.Fprintf(s.log, "[%s/%s] %s  %s  %s\n",
		indexStyle.Render(fmt.Sprintf("%d", idx)),
		totalStyle.Render(fmt.Sprintf("%d", total)),
		e.localRel,
		sizeStyle.Render(formatSize(n)),
		speedStyle.Render(fmt.Sprintf("%.1f MB/s", speedMBs)))
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, localPath); err != nil {
		return false, fmt.Errorf("rename into place: %w", err)
	}
	committed = true

	if e.modified.IsZero() {
		fmt.Fprintf(s.log, "warn: %s: no modification time from pCloud; local mtime will be now\n", e.cloudPath)
	} else if err := os.Chtimes(localPath, e.modified, e.modified); err != nil {
		fmt.Fprintf(s.log, "warn: %s: could not set mtime: %v\n", e.cloudPath, err)
	}

	return true, nil
}

// pruneLocal removes local files that are no longer present in the pCloud tree.
// Returns the number of files deleted.
func (s *Syncer) pruneLocal(remoteSet map[string]struct{}) (int, error) {
	deleted := 0
	err := filepath.Walk(s.localRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.localRoot, path)
		if err != nil {
			return nil
		}
		if _, exists := remoteSet[rel]; !exists {
			if removeErr := os.Remove(path); removeErr == nil {
				fmt.Fprintf(s.log, "deleted      %s\n", path)
				deleted++
			}
		}
		return nil
	})
	return deleted, err
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

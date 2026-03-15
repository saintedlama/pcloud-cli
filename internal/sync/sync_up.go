package sync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saintedlama/pcloud-cli/internal/pcloud"
)

// Uploader syncs a local directory tree to a pCloud directory.
// Files absent on pCloud or locally newer are uploaded.
// Remote files no longer present locally are deleted from pCloud.
type Uploader struct {
	api       *pcloud.API
	cloudRoot string
	localRoot string
	log       io.Writer
}

// NewUploader creates an Uploader. localRoot is the source on disk;
// cloudRoot is the pCloud destination path.
func NewUploader(api *pcloud.API, localRoot, cloudRoot string, log io.Writer) *Uploader {
	return &Uploader{
		api:       api,
		cloudRoot: "/" + strings.Trim(cloudRoot, "/"),
		localRoot: localRoot,
		log:       log,
	}
}

// Run performs one upload sync pass and returns when it is finished.
// In RunResult: Downloaded counts uploaded files, Deleted counts remote deletions.
func (u *Uploader) Run(ctx context.Context) (RunResult, error) {
	fmt.Fprintf(u.log, "Scanning local  %s ...\n", u.localRoot)
	fmt.Fprintf(u.log, "Scanning remote %s ...\n", u.cloudRoot)

	// Ensure the cloud root exists before listing.
	if err := u.ensureCloudPath(u.cloudRoot, make(map[string]struct{})); err != nil {
		return RunResult{}, fmt.Errorf("ensure cloud root: %w", err)
	}

	// Index the remote tree by relative path.
	scanner := &Syncer{api: u.api, cloudRoot: u.cloudRoot, localRoot: u.localRoot, log: u.log}
	var remoteEntries []fileEntry
	if err := scanner.walk(ctx, u.cloudRoot, &remoteEntries); err != nil {
		return RunResult{}, fmt.Errorf("listing pCloud tree: %w", err)
	}
	remoteByRel := make(map[string]fileEntry, len(remoteEntries))
	for _, e := range remoteEntries {
		remoteByRel[e.localRel] = e
	}

	// Walk the local tree.
	type localFile struct {
		rel     string
		absPath string
		modTime time.Time
	}
	var localFiles []localFile
	if err := filepath.WalkDir(u.localRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(u.localRoot, path)
		localFiles = append(localFiles, localFile{rel: rel, absPath: path, modTime: info.ModTime()})
		return nil
	}); err != nil {
		return RunResult{}, fmt.Errorf("walking local tree: %w", err)
	}

	localSet := make(map[string]struct{}, len(localFiles))
	for _, f := range localFiles {
		localSet[f.rel] = struct{}{}
	}

	fmt.Fprintf(u.log, "Found %d file(s) locally, %d on pCloud\n", len(localFiles), len(remoteEntries))

	// Determine which local files need uploading.
	var toUpload []localFile
	for _, f := range localFiles {
		re, exists := remoteByRel[f.rel]
		if !exists || (!re.modified.IsZero() && f.modTime.After(re.modified)) {
			toUpload = append(toUpload, f)
		}
	}

	// Determine which remote files need deleting.
	var toDelete []fileEntry
	for rel, re := range remoteByRel {
		if _, exists := localSet[rel]; !exists {
			toDelete = append(toDelete, re)
		}
	}

	var res RunResult
	res.Total = len(localFiles)

	// Upload.
	ensuredDirs := make(map[string]struct{})
	for i, f := range toUpload {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		cloudDir := u.toCloudDir(f.rel)
		if err := u.ensureCloudPath(cloudDir, ensuredDirs); err != nil {
			fmt.Fprintf(u.log, "warn: ensure dir %s: %v\n", cloudDir, err)
			res.Warnings++
			continue
		}
		if err := u.uploadFile(f.absPath, cloudDir, f.rel, i+1, len(toUpload)); err != nil {
			fmt.Fprintf(u.log, "warn: %s: %v\n", f.rel, err)
			res.Warnings++
		} else {
			res.Downloaded++
		}
	}

	// Delete remote-only files.
	for _, e := range toDelete {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		if _, err := u.api.DeleteFile(e.cloudPath); err != nil {
			fmt.Fprintf(u.log, "warn: delete %s: %v\n", e.cloudPath, err)
			res.Warnings++
		} else {
			fmt.Fprintf(u.log, "deleted      %s\n", e.cloudPath)
			res.Deleted++
		}
	}

	return res, nil
}

// Watch calls Run repeatedly, sleeping interval between passes, until ctx is
// cancelled. It returns nil on clean shutdown.
func (u *Uploader) Watch(ctx context.Context, interval time.Duration) error {
	for {
		start := time.Now()
		fmt.Fprintf(u.log, "\n[%s] === sync start  %s -> %s ===\n",
			start.Format(time.RFC3339), u.localRoot, u.cloudRoot)

		res, err := u.Run(ctx)
		if ctx.Err() != nil {
			return nil
		} else if err != nil {
			fmt.Fprintf(u.log, "[%s] sync error: %v\n",
				time.Now().Format(time.RFC3339), err)
		} else {
			fmt.Fprintf(u.log,
				"[%s] sync done   %d uploaded, %d deleted, %d up to date  (%.1fs)\n",
				time.Now().Format(time.RFC3339),
				res.Downloaded, res.Deleted, res.Total-res.Downloaded-res.Warnings,
				time.Since(start).Seconds())
		}

		fmt.Fprintf(u.log, "Next sync in %s\n", interval)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// toCloudDir converts a local-relative file path to the absolute pCloud
// directory that should contain it.
// e.g. localRel="Rock/song.mp3" → "/CloudMusic/Rock"
func (u *Uploader) toCloudDir(localRel string) string {
	dir := filepath.Dir(localRel)
	if dir == "." {
		return u.cloudRoot
	}
	return u.cloudRoot + "/" + filepath.ToSlash(dir)
}

// ensureCloudPath makes sure every component of cloudDir exists on pCloud,
// creating from cloudRoot outward. ensured is a caller-maintained cache that
// avoids redundant API calls within a single sync pass.
func (u *Uploader) ensureCloudPath(cloudDir string, ensured map[string]struct{}) error {
	rel := strings.TrimPrefix(cloudDir, u.cloudRoot)
	rel = strings.TrimPrefix(rel, "/")

	current := u.cloudRoot
	if _, ok := ensured[current]; !ok {
		if err := u.api.EnsureFolder(current); err != nil {
			return err
		}
		ensured[current] = struct{}{}
	}

	if rel == "" {
		return nil
	}

	for _, part := range strings.Split(rel, "/") {
		current = current + "/" + part
		if _, ok := ensured[current]; !ok {
			if err := u.api.EnsureFolder(current); err != nil {
				return err
			}
			ensured[current] = struct{}{}
		}
	}
	return nil
}

// uploadFile uploads one local file to cloudDir and logs the result.
func (u *Uploader) uploadFile(absPath, cloudDir, rel string, idx, total int) error {
	info, _ := os.Stat(absPath)
	var size int64
	if info != nil {
		size = info.Size()
	}

	start := time.Now()
	if _, err := u.api.UploadFile(absPath, cloudDir, false); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	elapsed := time.Since(start)
	if elapsed < time.Millisecond {
		elapsed = time.Millisecond
	}

	speedMBs := float64(size) / elapsed.Seconds() / (1024 * 1024)
	fmt.Fprintf(u.log, "[%s/%s] %s  %s  %s\n",
		indexStyle.Render(fmt.Sprintf("%d", idx)),
		totalStyle.Render(fmt.Sprintf("%d", total)),
		rel,
		sizeStyle.Render(formatSize(size)),
		speedStyle.Render(fmt.Sprintf("%.1f MB/s", speedMBs)))
	return nil
}

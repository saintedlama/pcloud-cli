// Package sync – TwoWaySyncer
//
// Performs bidirectional sync between a local directory and a pCloud folder.
// On each run it pulls remote changes down first (same as Syncer), then
// watches the local file-system for events using fsnotify and mirrors those
// changes back to pCloud. A periodic pull pass keeps the local tree in sync
// with changes made to pCloud from other clients.
package sync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	gosync "sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/saintedlama/pcloud-cli/internal/pcloud"
)

const twoWayDebounce = 300 * time.Millisecond

// TwoWaySyncer mirrors files in both directions between a local directory and
// a pCloud folder. It always pulls the cloud state first to resolve any remote
// changes before acting on local events.
type TwoWaySyncer struct {
	api       *pcloud.API
	cloudRoot string // no trailing slash
	localRoot string // no trailing slash
	log       io.Writer

	// pullSnapshot records the mtime of every local file captured immediately
	// after each cloud→local pull pass.  The fsnotify handler uses it to
	// distinguish files we just downloaded from genuine user edits.
	pullSnapshot gosync.Map // abs path (string) → time.Time

	ensured   map[string]struct{} // cloud dirs confirmed to exist this session
	ensuredMu gosync.Mutex
}

// NewTwoWay creates a TwoWaySyncer. cloudRoot is the pCloud folder path;
// localRoot is the local directory. Progress messages are written to log.
func NewTwoWay(api *pcloud.API, cloudRoot, localRoot string, log io.Writer) *TwoWaySyncer {
	return &TwoWaySyncer{
		api:       api,
		cloudRoot: "/" + strings.Trim(cloudRoot, "/"),
		localRoot: strings.TrimRight(localRoot, "/"),
		log:       log,
		ensured:   make(map[string]struct{}),
	}
}

// Run performs an initial pull (pCloud → local), then enters a loop that:
//   - watches the local file-system for changes and pushes them to pCloud;
//   - periodically re-runs a pull pass to incorporate remote-only changes.
//
// Run blocks until ctx is cancelled and returns nil on clean shutdown.
func (t *TwoWaySyncer) Run(ctx context.Context, pullInterval time.Duration) error {
	if err := os.MkdirAll(t.localRoot, 0o755); err != nil {
		return fmt.Errorf("creating local root: %w", err)
	}

	// 1. Initial pull: cloud → local.
	fmt.Fprintf(t.log, "\n[two-way] initial pull  %s → %s\n", t.cloudRoot, t.localRoot)
	syncer := New(t.api, t.cloudRoot, t.localRoot, t.log)
	if _, err := syncer.Run(ctx); err != nil {
		return fmt.Errorf("initial pull: %w", err)
	}
	t.snapshotLocal()

	// 2. Open fsnotify watcher and watch the full local tree.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fs watcher: %w", err)
	}
	defer watcher.Close()

	if err := t.addDirsToWatcher(watcher); err != nil {
		return fmt.Errorf("watching local tree: %w", err)
	}

	// 3. Per-file debounce: rapid consecutive writes to the same file are
	//    collapsed into a single upload triggered after twoWayDebounce.
	debounced := make(map[string]*time.Timer)
	var debounceMu gosync.Mutex

	scheduleUpload := func(absPath string) {
		debounceMu.Lock()
		defer debounceMu.Unlock()
		if tm, ok := debounced[absPath]; ok {
			tm.Reset(twoWayDebounce)
			return
		}
		debounced[absPath] = time.AfterFunc(twoWayDebounce, func() {
			debounceMu.Lock()
			delete(debounced, absPath)
			debounceMu.Unlock()
			if ctx.Err() != nil {
				return
			}
			if err := t.uploadToCloud(absPath); err != nil {
				fmt.Fprintf(t.log, "[two-way] warn: upload %s: %v\n", absPath, err)
			}
		})
	}

	pullTicker := time.NewTicker(pullInterval)
	defer pullTicker.Stop()

	fmt.Fprintf(t.log, "[two-way] watching %s  (pull every %s)\n", t.localRoot, pullInterval)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-pullTicker.C:
			fmt.Fprintf(t.log, "\n[two-way] pull pass\n")
			if _, err := syncer.Run(ctx); err != nil {
				fmt.Fprintf(t.log, "[two-way] warn: pull: %v\n", err)
			} else {
				t.snapshotLocal()
				// Pick up any new subdirectories created by the pull.
				_ = t.addDirsToWatcher(watcher)
			}

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			t.handleFSEvent(ctx, event, watcher, scheduleUpload)

		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(t.log, "[two-way] warn: watcher: %v\n", watchErr)
		}
	}
}

// handleFSEvent dispatches a single fsnotify event.
func (t *TwoWaySyncer) handleFSEvent(
	ctx context.Context,
	event fsnotify.Event,
	watcher *fsnotify.Watcher,
	scheduleUpload func(string),
) {
	absPath := event.Name

	switch {
	case event.Has(fsnotify.Create):
		fi, err := os.Lstat(absPath)
		if err != nil {
			// File may have been removed in the meantime.
			return
		}
		if fi.IsDir() {
			// Start watching any directory created inside the tree.
			_ = watcher.Add(absPath)
			return
		}
		if t.isJustPulled(absPath) {
			return
		}
		scheduleUpload(absPath)

	case event.Has(fsnotify.Write):
		fi, err := os.Lstat(absPath)
		if err != nil || fi.IsDir() {
			return
		}
		if t.isJustPulled(absPath) {
			return
		}
		scheduleUpload(absPath)

	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		// Rename fires on the *old* path (it disappears); the new path arrives
		// as a subsequent Create event and will be uploaded then.
		t.pullSnapshot.Delete(absPath)
		cloudPath := t.toCloudPath(absPath)
		if err := t.deleteFromCloud(cloudPath); err != nil {
			fmt.Fprintf(t.log, "[two-way] warn: delete %s: %v\n", cloudPath, err)
		}
	}
}

// snapshotLocal records the mtime of every local file right after a pull pass
// so that fsnotify events caused by those downloads can be suppressed.
func (t *TwoWaySyncer) snapshotLocal() {
	t.pullSnapshot.Range(func(k, _ any) bool {
		t.pullSnapshot.Delete(k)
		return true
	})
	_ = filepath.WalkDir(t.localRoot, func(fpath string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err == nil {
			t.pullSnapshot.Store(fpath, fi.ModTime())
		}
		return nil
	})
}

// isJustPulled returns true when absPath exists in the pull snapshot with a
// matching mtime, meaning it was last written by a pull pass and not by the
// user.
func (t *TwoWaySyncer) isJustPulled(absPath string) bool {
	v, ok := t.pullSnapshot.Load(absPath)
	if !ok {
		return false
	}
	fi, err := os.Lstat(absPath)
	if err != nil {
		t.pullSnapshot.Delete(absPath)
		return false
	}
	if fi.ModTime().Equal(v.(time.Time)) {
		return true
	}
	// mtime changed — user modified the file; remove the stale snapshot entry.
	t.pullSnapshot.Delete(absPath)
	return false
}

// addDirsToWatcher adds every subdirectory under localRoot to watcher.
// Paths already watched by fsnotify are silently ignored.
func (t *TwoWaySyncer) addDirsToWatcher(watcher *fsnotify.Watcher) error {
	return filepath.WalkDir(t.localRoot, func(fpath string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		return watcher.Add(fpath)
	})
}

// uploadToCloud uploads a single local file to the corresponding pCloud path.
func (t *TwoWaySyncer) uploadToCloud(absPath string) error {
	cloudDir := t.toCloudDir(absPath)
	if err := t.ensureCloudDir(cloudDir); err != nil {
		return fmt.Errorf("ensure cloud dir %s: %w", cloudDir, err)
	}
	if _, err := t.api.UploadFile(absPath, cloudDir, false); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	rel := strings.TrimPrefix(absPath, t.localRoot+string(filepath.Separator))
	fmt.Fprintf(t.log, "[two-way] uploaded  %s → %s\n", rel, cloudDir)
	return nil
}

// deleteFromCloud removes a file or folder at cloudPath from pCloud. It
// attempts file deletion first; if that fails it tries recursive folder
// deletion. Errors from both attempts are reported only if both fail.
func (t *TwoWaySyncer) deleteFromCloud(cloudPath string) error {
	if _, err := t.api.DeleteFile(cloudPath); err == nil {
		fmt.Fprintf(t.log, "[two-way] deleted file   %s\n", cloudPath)
		return nil
	}
	if _, err := t.api.DeleteFolderRecursive(cloudPath); err != nil {
		return fmt.Errorf("deleting %s: %w", cloudPath, err)
	}
	fmt.Fprintf(t.log, "[two-way] deleted folder %s\n", cloudPath)
	return nil
}

// ensureCloudDir creates each path component of cloudDir on pCloud if it does
// not already exist, starting from cloudRoot outward. A session-scoped cache
// prevents redundant API calls.
func (t *TwoWaySyncer) ensureCloudDir(cloudDir string) error {
	t.ensuredMu.Lock()
	defer t.ensuredMu.Unlock()

	rel := strings.TrimPrefix(cloudDir, t.cloudRoot)
	rel = strings.TrimPrefix(rel, "/")

	current := t.cloudRoot
	if _, ok := t.ensured[current]; !ok {
		if err := t.api.EnsureFolder(current); err != nil {
			return fmt.Errorf("ensuring %s: %w", current, err)
		}
		t.ensured[current] = struct{}{}
	}

	if rel == "" {
		return nil
	}

	for _, part := range strings.Split(rel, "/") {
		current += "/" + part
		if _, ok := t.ensured[current]; !ok {
			if err := t.api.EnsureFolder(current); err != nil {
				return fmt.Errorf("ensuring %s: %w", current, err)
			}
			t.ensured[current] = struct{}{}
		}
	}
	return nil
}

// toCloudPath maps an absolute local path to its corresponding pCloud path.
func (t *TwoWaySyncer) toCloudPath(absPath string) string {
	rel := strings.TrimPrefix(absPath, t.localRoot)
	return t.cloudRoot + filepath.ToSlash(rel)
}

// toCloudDir returns the pCloud directory that should contain the file at absPath.
func (t *TwoWaySyncer) toCloudDir(absPath string) string {
	return path.Dir(t.toCloudPath(absPath))
}

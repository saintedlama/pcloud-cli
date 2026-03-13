package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	gosync "github.com/saintedlama/pcloud-cli/internal/sync"
)

var syncInterval time.Duration
var syncDryRun bool
var syncMode string

// SyncCmd is the top-level "sync" command. Default mode is "down" (pCloud → local).
var SyncCmd = &cobra.Command{
	Use:   "sync <pcloud-path|local-dir> [local-dir|pcloud-path]",
	Short: "Sync between a pCloud folder and a local directory.",
	Long: `Sync files between pCloud and a local directory.

Default mode is "down": files newer on pCloud are downloaded and files deleted
from pCloud are removed locally. Argument order: <pcloud-path> [local-dir].

With --mode up the direction is reversed: files locally newer or absent on pCloud
are uploaded, and remote files no longer present locally are deleted.

With --mode up and a single argument that argument is treated as the local
directory and the pCloud target name is derived from its last path component:

  sync ./my-photos --mode up       # uploads to /my-photos on pCloud
  sync /data/project --mode up     # uploads to /project on pCloud

With --mode up and two arguments the first is the pCloud path and the second
is the local directory (same order as --mode down).`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runSyncOnce,
}

var syncDaemonCmd = &cobra.Command{
	Use:   "daemon <pcloud-path> [local-dir]",
	Short: "Sync continuously, polling pCloud for changes.",
	Long: `Run a continuous sync daemon that polls pCloud on a fixed interval.

The daemon logs every file synced or deleted and runs until interrupted
(Ctrl-C / SIGTERM). Use --interval to change the polling frequency.`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runSyncDaemon,
}

var syncSystemdCmd = &cobra.Command{
	Use:   "systemd <pcloud-path> [local-dir]",
	Short: "Install a systemd user service for continuous sync.",
	Long: `Write a systemd user service unit that runs the sync daemon, then
enable and start it with systemctl --user.

The local-dir path is resolved to an absolute path before being embedded in
the unit file, so it works regardless of where the service is started from.
Use --interval to set the polling frequency written into the unit.
Use --dry-run to print the unit file and describe what would be done without
making any changes.`,

	Args: cobra.RangeArgs(1, 2),
	Run:  runSyncSystemd,
}

func init() {
	RootCmd.AddCommand(SyncCmd)
	SyncCmd.AddCommand(syncDaemonCmd)
	SyncCmd.AddCommand(syncSystemdCmd)
	SyncCmd.PersistentFlags().DurationVar(&syncInterval, "interval", 60*time.Second,
		"polling interval used by the daemon (e.g. 30s, 5m)")
	SyncCmd.PersistentFlags().StringVar(&syncMode, "mode", "down",
		`sync direction: "down" (pCloud → local) or "up" (local → pCloud)`)
	syncSystemdCmd.Flags().BoolVar(&syncDryRun, "dry-run", false,
		"print the unit file and describe what would be done without making any changes")
}

// parseSyncArgs extracts the pCloud path and local directory from args,
// defaulting the local directory to the last component of the cloud path.
// Used for --mode down.
func parseSyncArgs(args []string) (cloudPath, localDir string) {

	cloudPath = args[0]
	if len(args) >= 2 {
		localDir = args[1]
		return
	}
	base := filepath.Base(strings.TrimRight(filepath.ToSlash(cloudPath), "/"))
	if base == "" || base == "." || base == "/" {
		base = "pcloud-sync"
	}
	localDir = base
	return
}

// parseUploadArgs extracts the local directory and pCloud path for --mode up.
// With two arguments the first is the pCloud path and the second is the local
// directory (matching the down-mode argument order).
// With one argument the argument is treated as the local path; the pCloud target
// is derived from the last component of the resolved absolute path.
// Returns an error when the derived name would be empty or a bare root ("/").
func parseUploadArgs(args []string) (localDir, cloudPath string, err error) {
	if len(args) == 2 {
		return args[1], args[0], nil
	}
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return "", "", fmt.Errorf("could not resolve local path: %w", err)
	}
	base := filepath.Base(abs)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", "", errors.New("cannot derive pCloud path from root directory; provide <pcloud-path> and <local-dir> explicitly")
	}
	return abs, "/" + base, nil
}

func runSyncOnce(cmd *cobra.Command, args []string) {
	cloudPath, localDir := parseSyncArgs(args)

	switch syncMode {
	case "down":
		absLocal, err := filepath.Abs(localDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not resolve local dir:", err)
			os.Exit(1)
		}
		fmt.Printf("Syncing  %s  ->  %s\n\n", cloudPath, absLocal)
		syncer := gosync.New(API, cloudPath, absLocal, os.Stdout)
		res, err := syncer.Run(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, "sync failed:", err)
			os.Exit(1)
		}
		upToDate := res.Total - res.Downloaded - res.Warnings
		fmt.Printf("\nDone: %d downloaded, %d deleted, %d up to date",
			res.Downloaded, res.Deleted, upToDate)
		if res.Warnings > 0 {
			fmt.Printf(", %d warning(s)", res.Warnings)
		}
		fmt.Println()

	case "up":
		absLocal, cloudPath, err := parseUploadArgs(args)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		absLocal, err = filepath.Abs(absLocal)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not resolve local dir:", err)
			os.Exit(1)
		}
		fmt.Printf("Syncing  %s  ->  %s\n\n", absLocal, cloudPath)
		uploader := gosync.NewUploader(API, absLocal, cloudPath, os.Stdout)
		res, err := uploader.Run(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, "sync failed:", err)
			os.Exit(1)
		}
		upToDate := res.Total - res.Downloaded - res.Warnings
		fmt.Printf("\nDone: %d uploaded, %d deleted, %d up to date",
			res.Downloaded, res.Deleted, upToDate)
		if res.Warnings > 0 {
			fmt.Printf(", %d warning(s)", res.Warnings)
		}
		fmt.Println()

	default:
		fmt.Fprintf(os.Stderr, "unknown sync mode %q: must be \"down\" or \"up\"\n", syncMode)
		os.Exit(1)
	}
}

func runSyncDaemon(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch syncMode {
	case "down":
		cloudPath, localDir := parseSyncArgs(args)
		absLocal, err := filepath.Abs(localDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not resolve local dir:", err)
			os.Exit(1)
		}
		fmt.Printf("Starting sync daemon (down): %s → %s (interval: %s)\n",
			cloudPath, absLocal, syncInterval)
		syncer := gosync.New(API, cloudPath, absLocal, os.Stdout)
		if err := syncer.Watch(ctx, syncInterval); err != nil {
			fmt.Fprintln(os.Stderr, "daemon error:", err)
			os.Exit(1)
		}
	case "up":
		localDir, cloudPath, err := parseUploadArgs(args)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		absLocal, err := filepath.Abs(localDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not resolve local dir:", err)
			os.Exit(1)
		}
		fmt.Printf("Starting sync daemon (up): %s → %s (interval: %s)\n",
			absLocal, cloudPath, syncInterval)
		uploader := gosync.NewUploader(API, absLocal, cloudPath, os.Stdout)
		if err := uploader.Watch(ctx, syncInterval); err != nil {
			fmt.Fprintln(os.Stderr, "daemon error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown sync mode %q: must be \"down\" or \"up\"\n", syncMode)
		os.Exit(1)
	}
}

// unitTemplate is the systemd service unit template.
// Paths that may contain spaces are quoted via the "q" function.
const unitTemplate = `[Unit]
Description=pCloud sync {{q .CloudPath}} to {{q .LocalDir}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{q .ExecPath}} sync daemon {{q .CloudPath}} {{q .LocalDir}} --interval {{.Interval}} --mode {{.Mode}}
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`

type unitVars struct {
	CloudPath string
	LocalDir  string
	ExecPath  string
	Interval  string
	Mode      string
}

func runSyncSystemd(cmd *cobra.Command, args []string) {
	var cloudPath, localDir string
	if syncMode == "up" {
		var err error
		localDir, cloudPath, err = parseUploadArgs(args)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		cloudPath, localDir = parseSyncArgs(args)
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not determine executable path:", err)
		os.Exit(1)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not resolve executable symlinks:", err)
		os.Exit(1)
	}

	absLocal, err := filepath.Abs(localDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not resolve local dir:", err)
		os.Exit(1)
	}

	// Sanitize cloud path into a valid unit name component.
	sanitized := strings.NewReplacer("/", "-", " ", "_").Replace(strings.Trim(cloudPath, "/"))
	unitName := "pcloud-sync-" + sanitized + ".service"

	unitDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, unitName)

	tmpl := template.Must(template.New("unit").Funcs(template.FuncMap{
		// q wraps a value in double quotes if it contains whitespace.
		"q": func(s string) string {
			if strings.ContainsAny(s, " \t") {
				return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
			}
			return s
		},
	}).Parse(unitTemplate))

	vars := unitVars{
		CloudPath: cloudPath,
		LocalDir:  absLocal,
		ExecPath:  execPath,
		Interval:  syncInterval.String(),
		Mode:      syncMode,
	}

	if syncDryRun {
		fmt.Print("=== Dry run — no files will be written and no commands will be run ===\n\n")
		fmt.Printf("Unit file path : %s\n\n", unitPath)
		fmt.Println("--- unit file contents ---")
		if err := tmpl.Execute(os.Stdout, vars); err != nil {
			fmt.Fprintln(os.Stderr, "could not render unit template:", err)
			os.Exit(1)
		}
		fmt.Print("--- end unit file ---\n\n")
		fmt.Println("Steps that would be executed:")
		fmt.Printf("  1. Write unit file to %s\n", unitPath)
		fmt.Println("  2. systemctl --user daemon-reload")
		fmt.Printf("  3. systemctl --user enable --now %s\n", unitName)
		return
	}

	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "could not create systemd user directory:", err)
		os.Exit(1)
	}

	f, err := os.Create(unitPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not write unit file:", err)
		os.Exit(1)
	}
	if err := tmpl.Execute(f, vars); err != nil {
		f.Close()
		fmt.Fprintln(os.Stderr, "could not render unit template:", err)
		os.Exit(1)
	}
	f.Close()

	fmt.Printf("Unit file written: %s\n\n", unitPath)

	for _, scArgs := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", unitName},
	} {
		out, err := exec.Command("systemctl", scArgs...).CombinedOutput() //nolint:gosec
		if err != nil {
			fmt.Fprintf(os.Stderr, "systemctl %s: %v\n%s\n",
				strings.Join(scArgs, " "), err, out)
			os.Exit(1)
		}
	}

	fmt.Printf("Service %s enabled and started.\n\n", unitName)

	if out, err := exec.Command("systemctl", "--user", "status", unitName).CombinedOutput(); err == nil { //nolint:gosec
		fmt.Println(string(out))
	}
}

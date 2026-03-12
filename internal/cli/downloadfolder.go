package cli

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/saintedlama/pcloud-cli/internal/pcloud"
	"github.com/spf13/cobra"
)

var (
	downloadfolderCmd = &cobra.Command{
		Use:   "download [remote folder path] [optional local destination dir]",
		Short: "Download and extract a remote folder.",
		Long: `Downloads a remote folder as a zip archive and extracts it locally.
The archive is always extracted into a local folder named like the remote folder.`,
		Run: downloadfolder,
	}
	downloadForce bool
)

func init() {
	FolderCmd.AddCommand(downloadfolderCmd)
	downloadfolderCmd.Flags().BoolVarP(&downloadForce, "force", "f", false, "overwrite existing destination folder")
}

func downloadfolder(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Invalid input. See 'pcloud-cli folder download --help'.")
		os.Exit(1)
	}

	remotePath := args[0]
	localDestination := "."
	if len(args) > 1 {
		localDestination = args[1]
	}

	folderData, err := API.ListFolder(remotePath, pcloud.ListFolderOptions{NoFiles: true})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	folderName := strings.TrimSpace(folderData.Metadata.Name)
	if folderName == "" {
		folderName = filepath.Base(strings.TrimRight(remotePath, "/"))
	}
	if folderName == "" || folderName == "." || folderName == string(os.PathSeparator) {
		fmt.Println("Could not determine folder name from remote path.")
		os.Exit(1)
	}

	zipLink, err := API.GetZipLinkByFolderID(folderData.Metadata.FolderID, folderName+".zip", false)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(zipLink.Hosts) < 1 {
		fmt.Println("No download host was returned by the API.")
		os.Exit(1)
	}

	downloadURL := "https://" + zipLink.Hosts[0] + zipLink.Path
	zipFilePath, err := downloadTempFile(downloadURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer os.Remove(zipFilePath)

	targetDir := filepath.Join(localDestination, folderName)
	if info, statErr := os.Stat(targetDir); statErr == nil {
		if !downloadForce {
			fmt.Println("Destination folder already exists. Use --force to overwrite.")
			os.Exit(1)
		}

		if info.IsDir() {
			if err := os.RemoveAll(targetDir); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			if err := os.Remove(targetDir); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	} else if !os.IsNotExist(statErr) {
		fmt.Println(statErr)
		os.Exit(1)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := extractZipToDir(zipFilePath, targetDir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Folder successfully downloaded and extracted")
	fmt.Println("Destination: " + targetDir)
}

func downloadTempFile(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download failed with status: %s", response.Status)
	}

	tempFile, err := os.CreateTemp("", "pcloud-folder-*.zip")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, response.Body); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func extractZipToDir(zipPath, destination string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanDestination := filepath.Clean(destination)
	prefix := cleanDestination + string(os.PathSeparator)

	for _, file := range reader.File {
		targetPath := filepath.Join(cleanDestination, file.Name)
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget != cleanDestination && !strings.HasPrefix(cleanTarget, prefix) {
			return fmt.Errorf("invalid archive path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, file.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}

		sourceFile, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			sourceFile.Close()
			return err
		}

		_, copyErr := io.Copy(targetFile, sourceFile)
		sourceCloseErr := sourceFile.Close()
		targetCloseErr := targetFile.Close()

		if copyErr != nil {
			return copyErr
		}
		if sourceCloseErr != nil {
			return sourceCloseErr
		}
		if targetCloseErr != nil {
			return targetCloseErr
		}
	}

	return nil
}

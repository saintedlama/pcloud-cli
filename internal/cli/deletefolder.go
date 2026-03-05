package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	deletefolderCmd = &cobra.Command{
		Use:   "delete [path to folder]",
		Short: "Delete folder.",
		Long: `Delete folder
The given folder must be empty, if not the -r flag should be set.
Paths containing spaces should be wrapped in double quotes.`,

		Run: deletefolder,
	}
)

var (
	deleterecursive bool
)

func init() {
	FolderCmd.AddCommand(deletefolderCmd)
	deletefolderCmd.Flags().BoolVarP(&deleterecursive, "recursive", "r", false, "perform recursive delete")

	// Hidden / Aliased
}

func deletefolder(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Invalid input. See 'pcloud-cli deletefolder --help'.")
		os.Exit(1)
	}

	api := pcloud.NewAPI()

	switch {
	case deleterecursive:
		response, err := api.DeleteFolderRecursive(args[0], AccessToken)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Successfully deleted folder.")
		if verbose {
			fmt.Println("Deleted files: " + strconv.Itoa(response.DeletedFiles))
			fmt.Println("Deleted folders: " + strconv.Itoa(response.DeletedFolders))
		}
	default:
		response, err := api.DeleteFolder(args[0], AccessToken)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Successfully deleted folder.")
		if verbose {
			fmt.Println("Name: " + response.Metadata.Name)
			fmt.Println("Path: " + response.Metadata.Path)
			fmt.Println("Modified: " + response.Metadata.Modified)
			fmt.Println("Folder ID: " + strconv.Itoa(response.Metadata.FolderID))
		}
	}
}

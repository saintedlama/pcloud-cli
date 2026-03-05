package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	renamefolderCmd = &cobra.Command{
		Use:   "rename [source folder] [destination folder]",
		Short: "Rename / Move folder.",
		Long: `Rename or move source folder.
If source is to be moved to destination without changing name,
destination must end with /. This to avoid name change.
Paths containing spaces should be wrapped in double quotes.`,

		Run: renamefolder,
	}

	mvCmd = &cobra.Command{
		Hidden: true,

		Use:   "mv [source folder] [destination folder]",
		Short: "Move / Rename folder.",
		Long: `Rename or move source folder.
If source is to be moved to destination without changing name,
destination must end with /. This to avoid name change.
Paths containing spaces should be wrapped in double quotes.`,

		Run: renamefolder,
	}
)

func init() {
	FolderCmd.AddCommand(renamefolderCmd)

	// Hidden / Aliased
	FolderCmd.AddCommand(mvCmd)
}

func renamefolder(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Invalid input. See 'pcloud-cli renamefolder --help'.")
		os.Exit(1)
	}

	api := pcloud.NewAPI()
	response, err := api.RenameFolder(args[0], args[1], AccessToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if args[1][len(args[1])-1] != 47 {
		fmt.Println("Folder renamed successfully")
	} else {
		fmt.Println("Folder moved successfully")
	}

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Path: " + response.Metadata.Path)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Folder ID: " + strconv.Itoa(response.Metadata.FolderID))
	}
}

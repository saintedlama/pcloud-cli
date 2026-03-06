package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	createfolderCmd = &cobra.Command{
		Use:   "create [path to folder]",
		Short: "Create folder.",
		Long: `Create a new folder in giver directory.
If no path is given, the top level directory is used.
Paths containing spaces should be wrapped in double quotes.`,

		Run: createfolder,
	}

	mkdirCmd = &cobra.Command{
		Hidden: true,

		Use:   "mkdir [path to folder]",
		Short: "Create folder in path",
		Long: `Create a new folder in giver directory.
If no path is given, the top level directory is used.
Paths containing spaces should be wrapped in double quotes.`,

		Run: createfolder,
	}
)

func init() {
	FolderCmd.AddCommand(createfolderCmd)

	// Hidden / Aliased
	FolderCmd.AddCommand(mkdirCmd)
}

func createfolder(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("No / Invalid folder name given.")
		os.Exit(1)
	}

	response, err := API.CreateFolder(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Folder created successfully")

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Path: " + response.Metadata.Path)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Folder ID: " + strconv.Itoa(response.Metadata.FolderID))
	}
}

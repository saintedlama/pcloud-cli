package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	deletefileCmd = &cobra.Command{
		Use:   "delete [path to file]",
		Short: "Delete file.",
		Long: `Delete given file from pCloud storage.
Paths containing spaces should be wrapped in double quotes.`,

		Run: deletefile,
	}
)

func init() {
	FileCmd.AddCommand(deletefileCmd)

	// Hidden / Aliased
}

func deletefile(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Invalid input. See 'pcloud-cli file delete --help'.")
		os.Exit(1)
	}

	response, err := API.DeleteFile(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("File deleted successfully")

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Size: " + strconv.Itoa(response.Metadata.Size))
		fmt.Println("Content type: " + response.Metadata.ContentType)
	}
}

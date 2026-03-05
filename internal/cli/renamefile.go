package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	renamefileCmd = &cobra.Command{
		Use:   "rename [source file] [destination file]",
		Short: "Rename / Move source file.",
		Long: `Rename / Move file to new location.
Paths containing spaces should be wrapped in double quotes.`,

		Run: renamefile,
	}
)

func init() {
	FileCmd.AddCommand(renamefileCmd)

	// Hidden / Aliased
}

func renamefile(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Invalid input. See 'pcloud-cli file rename --help'.")
		os.Exit(1)
	}

	api := pcloud.NewAPI()
	response, err := api.RenameFile(args[0], args[1], AccessToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("File renamed successfully")

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Size: " + strconv.Itoa(response.Metadata.Size))
		fmt.Println("Content type: " + response.Metadata.ContentType)
	}
}

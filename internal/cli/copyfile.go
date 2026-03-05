package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	copyfileCmd = &cobra.Command{
		Use:   "copy [source file] [destination file]",
		Short: "Copy file to another location.",
		Long: `Copy source to destination.
If destination file exists it will NOT overwrite it unless the
--overwrite flag i set. Paths containing spaces should be wrapped
in double quotes.`,

		Run: copyfile,
	}
)

var (
	overwrite bool
)

func init() {
	FileCmd.AddCommand(copyfileCmd)
	copyfileCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "overwrite if file exists")

	// Hidden / Aliased
}

func copyfile(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Invalid input. See 'pcloud-cli copyfile --help'.")
		os.Exit(1)
	}

	api := pcloud.NewAPI()
	response, err := api.CopyFile(args[0], args[1], overwrite, AccessToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("File copied successfully")

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Size: " + strconv.Itoa(response.Metadata.Size))
		fmt.Println("Content type: " + response.Metadata.ContentType)
	}
}

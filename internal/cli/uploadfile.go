package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	uploadfileCmd = &cobra.Command{
		Use:   "upload [path to remote folder] [path to local file]",
		Short: "Upload local file to remote folder.",
		Long: `Upload given file to remote folder in pCloud.
Paths containing spaces should be wrapped in double quotes.`,

		Run: uploadfile,
	}
)

var (
	renameifexists bool
)

func init() {
	FileCmd.AddCommand(uploadfileCmd)
	uploadfileCmd.Flags().BoolVarP(&renameifexists, "renameifexists", "", false, "rename file if it exists")

	// Hidden / Aliased
}

func uploadfile(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Invalid input. See 'pcloud-cli file upload --help'.")
		os.Exit(1)
	}

	remotePath := ""
	if len(args) > 1 {
		remotePath = args[1]
	}

	_, err := API.UploadFile(args[0], remotePath, renameifexists)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("File successfully uploaded")

}

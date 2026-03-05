package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	checksumfileCmd = &cobra.Command{
		Use:   "checksum [file]",
		Short: "Calculate chacksums of file.",
		Long: `Calculate md5 and sha1 checksums of file.
Paths containing spaces should be wrapped in double quotes.`,

		Run: checksumfile,
	}
)

func init() {
	FileCmd.AddCommand(checksumfileCmd)
	// Hidden / Aliased
}

func checksumfile(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Invalid input. See 'pcloud-cli file checksum --help'.")
		os.Exit(1)
	}

	api := pcloud.NewAPI()
	response, err := api.Checksum(args[0], AccessToken)
	if err != nil {
		fmt.Println("Could not fetch checksum.")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("MD5: " + response.Md)
	fmt.Println("SHA1: " + response.Sha)

	if verbose {
		fmt.Println("Name: " + response.Metadata.Name)
		fmt.Println("Modified: " + response.Metadata.Modified)
		fmt.Println("Size: " + strconv.Itoa(response.Metadata.Size))
		fmt.Println("Content type: " + response.Metadata.ContentType)
	}
}

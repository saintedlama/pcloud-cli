package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

var (
	listfolderCmd = &cobra.Command{
		Use:   "list [path to folder to list]",
		Short: "List folders in pCloud directory",
		Long: `List all folders in given pCloud directory.
If no path is given, the top level directory is listed.
Paths containing spaces should be wrapped in double quotes.`,

		Run: listfolder,
	}

	lsCmd = &cobra.Command{
		Hidden: true,

		Use:   "ls [path to folder to list]",
		Short: "List folders in pCloud directory",
		Long: `List all folders in given pCloud directory.
If no path is given, the top level directory is listed.
Paths containing spaces should be wrapped in double quotes.`,

		Run: listfolder,
	}
)

var (
	showdeleted bool
	nofiles     bool
)

func init() {
	FolderCmd.AddCommand(listfolderCmd)
	listfolderCmd.Flags().BoolVarP(&showdeleted, "showdeleted", "", false, "show deleted files")
	listfolderCmd.Flags().BoolVarP(&nofiles, "nofiles", "", false, "list directories only")

	// Hidden / aliased
	FolderCmd.AddCommand(lsCmd)
	lsCmd.Flags().BoolVarP(&showdeleted, "showdeleted", "", false, "show deleted files")
	lsCmd.Flags().BoolVarP(&nofiles, "nofiles", "", false, "list directories only")
}

func listfolder(cmd *cobra.Command, args []string) {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}

	api := pcloud.NewAPI()
	response, err := api.ListFolder(path, nofiles, showdeleted, AccessToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 20, 4, 4, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\n", "Name", "Modified", "ID")
	for i := range response.Metadata.Contents {
		fmt.Fprintf(w, "%s\t%s\t%d\n", response.Metadata.Contents[i].Name, response.Metadata.Contents[i].Modified, response.Metadata.Contents[i].FolderID)
	}
	w.Flush()
}

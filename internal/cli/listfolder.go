package cli

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
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

	response, err := API.ListFolder(path, nofiles, showdeleted)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	headerFmt := func(format string, a ...interface{}) string {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")).Render(fmt.Sprintf(format, a...))
	}
	fileFmt := func(format string, a ...interface{}) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(fmt.Sprintf(format, a...))
	}

	tbl := table.New("Name", "Modified", "ID")
	tbl.WithHeaderFormatter(headerFmt)
	tbl.WithFirstColumnFormatter(fileFmt)

	var fileCount, dirCount int
	for _, item := range response.Metadata.Contents {
		modified := item.Modified
		if t, err := time.Parse(time.RFC1123Z, item.Modified); err == nil {
			modified = t.Format("2006-01-02")
		}
		if item.IsFolder {
			dirCount++
		} else {
			fileCount++
		}
		tbl.AddRow(item.Name, modified, item.FolderID)
	}
	tbl.Print()

	dimSt := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldSt := lipgloss.NewStyle().Bold(true)
	fmt.Printf("\n%s %s,  %s %s,  %s %s\n",
		dimSt.Render("directories:"), boldSt.Render(fmt.Sprintf("%d", dirCount)),
		dimSt.Render("files:"), boldSt.Render(fmt.Sprintf("%d", fileCount)),
		dimSt.Render("total:"), boldSt.Render(fmt.Sprintf("%d", dirCount+fileCount)),
	)
}

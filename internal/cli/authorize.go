package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/spf13/cobra"
	"github.com/storvik/pcloud-cli/internal/config"
	"github.com/storvik/pcloud-cli/internal/helpers"
	"github.com/storvik/pcloud-cli/internal/pcloud"
)

func init() {
	RootCmd.AddCommand(authorizeCmd)
}

var authorizeCmd = &cobra.Command{
	Use:   "authorize",
	Short: "Authorize with pCloud.",
	Long: `Authorization is necessary to be able to interact with the pCloud API.
Will re-authorize with pCloud and rewrite config file. This command will
also be run if noe config file is present when running pcloud-cli`,

	Run: authorize,
}

func authorize(cmd *cobra.Command, args []string) {
	authURL := "https://my.pcloud.com/oauth2/authorize?response_type=code&client_id=" + ClientID

	fmt.Println("pCloud-cli authorization started.")
	fmt.Println("This will delete the old configuration file.")
	fmt.Println("Open URL below in browser and copy the code to authenticate.")
	fmt.Println("If clipboard utility was found the URL is automatically copied.")
	fmt.Println(authURL)
	helpers.Clipboard.Add(authURL)

	var code string
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The code you entered: %s", code)

	api := pcloud.NewAPI()
	auth, err := api.Authorize(ClientID, ClientSecret, code)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var conf config.File
	conf.UserID = auth.UserID
	conf.AccessToken = auth.AccessToken

	usr, _ := user.Current()
	configPath := usr.HomeDir

	config.WriteConfig(configPath, ".pcloud", &conf)
}

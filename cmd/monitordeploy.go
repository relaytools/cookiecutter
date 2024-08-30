package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var monitordeploy = &cobra.Command{
	Use:   "monitordeploy",
	Short: "deploy monitoring",
	Long:  `deploy monitoring`,
	Run: func(cmd *cobra.Command, args []string) {
		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)

		monitorThese := findMonitorRelays()
		//
		// output the urls for every relay that is listed in the directory
		monConfig := fmt.Sprintf(`
RELAY_URLS=%s
			`, strings.Join(monitorThese, ","))

		fmt.Println(monConfig)
	},
}

func init() {
	rootCmd.AddCommand(monitordeploy)
}

func findMonitorRelays() []string {
	var useURL string
	useURL = baseURL + "/api/sconfig/relays" + "?running=true"
	req, err := http.NewRequest("GET", useURL, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()
	var data []map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &data)
	fmt.Println(data)
	monitorThese := []string{}
	for _, r := range data {
		rn := "wss://" + r["name"].(string) + "." + r["domain"].(string)
		monitorThese = append(monitorThese, rn)
	}
	return monitorThese
}

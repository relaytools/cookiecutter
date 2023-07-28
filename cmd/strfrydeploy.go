package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var strfrydeploy = &cobra.Command{
	Use:   "strfrydeploy",
	Short: "deploy strfrys",
	Long:  `deploy strfrys`,
	Run: func(cmd *cobra.Command, args []string) {

		ev := signEventWithLoginToken()
		fmt.Println(ev)

		csrf := getCSRF()
		fmt.Println(csrf)

		performLogin(ev, csrf)

		checkAndRestartRelays()

		cleanUpDeletedRelays()
	},
}

func init() {
	rootCmd.AddCommand(strfrydeploy)
}

func getRelayList(status string) []map[string]interface{} {
	var useURL string
	if status == "provision" {
		useURL = baseURL + "/api/sconfig/relays"
	} else if status == "deleting" {
		useURL = baseURL + "/api/sconfig/relays/deleting"
	}
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
	return data
}

func getStrfryConf(relayID string) {
	configURL := fmt.Sprintf("%s/api/relay/%s/strfry", baseURL, relayID)
	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()

	configPath := fmt.Sprintf("%s/strfry.conf", relayID)
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		log.Fatalf("Error occurred while creating file. Error is: %s", err.Error())
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
	}
}

func deployStatusUpdate(relayID string, status string) {
	statusURL := fmt.Sprintf("%s/api/relay/%s/status?status=%s", baseURL, relayID, status)
	req, err := http.NewRequest("PUT", statusURL, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()
}

func cleanUpDeletedRelays() {
	relays := getRelayList("deleting")
	for _, relay := range relays {
		log.Printf("Deleting relay %s\n", relay["id"].(string))
		if relay["status"] == "deleting" {

			// working directory is /app/curldown
			runCmd("systemctl", []string{"stop", relay["id"].(string)})
			runCmd("systemctl", []string{"disable", relay["id"].(string)})
			runCmd("rm", []string{"-rf", "/lib/systemd/system/" + relay["id"].(string) + ".service"})
			runCmd("rm", []string{"-rf", relay["id"].(string)})

			// update status to deleted
			deployStatusUpdate(relay["id"].(string), "deleted")
		}
	}
}

func checkAndRestartRelays() {
	relays := getRelayList("provision")
	for _, relay := range relays {
		log.Printf("Provisioning relay %s\n", relay["id"].(string))
		if relay["status"] == "provision" {
			// working directory is /app/curldown
			// ...

			// make directory structure for strfry
			dbDir := fmt.Sprintf("%s/strfry-db", relay["id"].(string))
			runCmd("mkdir", []string{"-p", dbDir})

			// create strfry.conf
			getStrfryConf(relay["id"].(string))

			// create spamblaster.cfg
			file, err := os.OpenFile("spamblaster.cfg", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
			}
			defer file.Close()
			line := fmt.Sprintf("%s/api/sconfig/relays/%s", baseURL, relay["id"].(string))
			if _, err := file.WriteString(line); err != nil {
				log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
			}

			// create systemd unit file
			unit := fmt.Sprintf(`
[Unit]
Description=strfry
StartLimitInterval=0

[Service]
ExecStart=/app/strfry relay
Restart=always
RestartSec=1
WorkingDirectory=/app/curldown/%s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
`, relay["id"].(string))

			unitFileName := fmt.Sprintf("/lib/systemd/system/%s.service", relay["id"].(string))
			file, err = os.OpenFile(unitFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
			}
			defer file.Close()
			if _, err := file.WriteString(unit); err != nil {
				log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
			}

			// reload systemd
			runCmd("systemctl", []string{"daemon-reload"})

			// enable and start systemd unit
			runCmd("systemctl", []string{"enable", relay["id"].(string)})
			runCmd("systemctl", []string{"start", relay["id"].(string)})

			// report status to api
			deployStatusUpdate(relay["id"].(string), "running")
		}
	}
}

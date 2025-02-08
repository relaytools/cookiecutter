package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var strfrydeploy = &cobra.Command{
	Use:   "strfrydeploy",
	Short: "deploy strfrys",
	Long:  `deploy strfrys`,
	Run: func(cmd *cobra.Command, args []string) {
		ev := signEventWithLoginToken()
		csrf := getCSRF()
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
		useURL = baseURL + "/api/sconfig/relays" + "?ip=" + hostIP
	} else if status == "deleting" {
		useURL = baseURL + "/api/sconfig/relays/deleting" + "?ip=" + hostIP
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

			// working directory is the CWD
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
			// check if streaming is enabled
			streamEnabled := false
			streams := relay["streams"].([]interface{})
			if len(streams) > 0 {
				streamEnabled = true
			}

			// make directory structure for strfry
			dbDir := fmt.Sprintf("%s/strfry-db", relay["id"].(string))
			runCmd("mkdir", []string{"-p", dbDir})

			// create strfry.conf
			getStrfryConf(relay["id"].(string))

			// create spamblaster.cfg
			sbConf := fmt.Sprintf("%s/spamblaster.cfg", relay["id"].(string))
			file, err := os.OpenFile(sbConf, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
			}
			defer file.Close()
			line := fmt.Sprintf("%s/api/sconfig/relays/%s", baseURL, relay["id"].(string))
			if _, err := file.WriteString(line); err != nil {
				log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
			}

			// working directory is the CWD
			dir, err := os.Getwd()
			if err != nil {
				log.Fatalf("Error getting current directory, this should not happen: %v", err)
				return
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
WorkingDirectory=%s/%s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
`, dir, relay["id"].(string))

			unitFileName := fmt.Sprintf("/lib/systemd/system/%s.service", relay["id"].(string))
			file, err = os.OpenFile(unitFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
			}
			defer file.Close()
			if _, err := file.WriteString(unit); err != nil {
				log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
			}

			if streamEnabled {
				for i, stream := range streams {
					// create systemd unit file for streaming
					theStream := stream.(map[string]interface{})
					streamUnit := fmt.Sprintf(`
					[Unit]
					Description=strfry stream
					StartLimitInterval=0
					
					[Service]
					ExecStart=/app/strfry stream --dir=%s %s
					Restart=always
					RestartSec=1
					WorkingDirectory=%s/%s
					LimitNOFILE=infinity
					
					[Install]
					WantedBy=multi-user.target
					`, theStream["direction"].(string), theStream["url"].(string), dir, relay["id"].(string))

					streamUnitFileName := fmt.Sprintf("/lib/systemd/system/%s-stream%d.service", relay["id"].(string), i)
					file, err = os.OpenFile(streamUnitFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
					if err != nil {
						log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
					}
					defer file.Close()
					if _, err := file.WriteString(streamUnit); err != nil {
						log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
					}

					// create systemd unit file for syncing
					if theStream["sync"].(bool) {

						syncUnit := fmt.Sprintf(`
	[Unit]
	Description=strfry sync
	StartLimitInterval=0

	[Service]
	ExecStart=/app/strfry sync --dir=down %s
	WorkingDirectory=%s/%s
	LimitNOFILE=infinity
	Restart=no

	[Install]
	WantedBy=multi-user.target
	`, theStream["url"].(string), dir, relay["id"].(string))

						syncUnitFileName := fmt.Sprintf("/lib/systemd/system/%s-sync%d.service", relay["id"].(string), i)
						file, err = os.OpenFile(syncUnitFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
						if err != nil {
							log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
						}
						defer file.Close()
						if _, err := file.WriteString(syncUnit); err != nil {
							log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
						}
					}
				}
			}

			// reload systemd
			runCmd("systemctl", []string{"daemon-reload"})

			// enable and start systemd unit
			runCmd("systemctl", []string{"enable", relay["id"].(string)})
			runCmd("systemctl", []string{"restart", relay["id"].(string)})

			if streamEnabled {
				for i, stream := range streams {
					// enable and start stream
					theStream := stream.(map[string]interface{})
					runCmd("systemctl", []string{"enable", fmt.Sprintf("%s-stream%d", relay["id"].(string), i)})
					runCmd("systemctl", []string{"restart", fmt.Sprintf("%s-stream%d", relay["id"].(string), i)})

					if theStream["sync"].(bool) {
						//sleep for 3 seconds to allow stream to start
						time.Sleep(3 * time.Second)
						// then start the sync
						runCmd("systemctl", []string{"start", relay["id"].(string) + "-sync"})
					}
				}
			}

			// cleanup deleted streams if they exist
			for i := 0; i < 5; i++ {
				if i < len(streams) {
					continue
				}
				runCmd("systemctl", []string{"stop", fmt.Sprintf("%s-stream%d", relay["id"].(string), i)})
				runCmd("systemctl", []string{"disable", fmt.Sprintf("%s-stream%d", relay["id"].(string), i)})
			}

			// report status to api
			deployStatusUpdate(relay["id"].(string), "running")
		}
	}
}

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var haproxydeploy = &cobra.Command{
	Use:   "haproxydeploy",
	Short: "deploy haproxy",
	Long:  `deploy haproxy`,
	Run: func(cmd *cobra.Command, args []string) {
		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)
		checkAndRestartHaproxy()
	},
}

func init() {
	rootCmd.AddCommand(haproxydeploy)
}

func checkAndRestartHaproxy() {
	if viper.GetBool("MANAGE_SSL_CERTIFICATES") {
		checkAndRenewCerts()
	}
	getHaproxyCfg()
}

func checkAndRenewCerts() {
	certList := findCertDomains()

	certbotAddScript := fmt.Sprintf(`#!/bin/bash -e
/usr/bin/certbot certonly --config-dir="/etc/haproxy/certs" --work-dir="/etc/haproxy/certs" --logs-dir="/etc/haproxy/certs"`)

	for _, c := range certList {
		// do certbot stuff, output a service file maybe?
		// create systemd unit file
		certbotAddScript += fmt.Sprintf(` -d "%s" `, c)

	}

	certbotAddScript += fmt.Sprintf(` --agree-tos --register-unsafely-without-email --standalone --preferred-challenges http --http-01-port 10000 --non-interactive --expand

	cat /etc/haproxy/certs/live/%s/fullchain.pem /etc/haproxy/certs/live/%s/privkey.pem > /etc/haproxy/certs/bundle.pem
`, certList[0], certList[0])

	scriptFileName := fmt.Sprintf("/usr/local/bin/certbot-renew.sh.new")

	file, err := os.OpenFile(scriptFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatalf("Error occurred while opening file. Error is: %s", err.Error())
	}
	if _, err := file.WriteString(certbotAddScript); err != nil {
		log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
	}
	file.Close()

	isDiff, err := diffFiles("/usr/local/bin/certbot-renew.sh.new", "/usr/local/bin/certbot-renew.sh")
	if err == nil {
		if isDiff {
			// certbot will name the file after the first domain in the array
			runCmd("mv", []string{"/usr/local/bin/certbot-renew.sh.new", "/usr/local/bin/certbot-renew.sh"})
			renewStatus := runCmd("/usr/local/bin/certbot-renew.sh", []string{})
			if renewStatus == false {
				log.Println("error running certbot-renew.sh")
			}
			runCmd("systemctl", []string{"restart", "haproxy"})
		} else {
			log.Println("certbot-changes.. no change detected")
		}
	} else {
		log.Println("error occured while comparing files", err)
	}
}

func diffFiles(oldPath string, newPath string) (bool, error) {
	oldBytes, err := os.ReadFile(oldPath)
	if err != nil {
		log.Fatalf("Error occurred while reading old config file. Error is: %s", err.Error())
		return false, err
	}

	newBytes, err := os.ReadFile(newPath)
	if err != nil {
		log.Printf("Error occurred while reading new config file. Error is: %s", err.Error())
		return true, nil
	}

	if string(oldBytes) == string(newBytes) {
		fmt.Println("No changes detected")
		return false, nil
	} else {
		return true, nil
	}
}

func findCertDomains() []string {
	var useURL string
	rootDomains := make(map[string]bool)
	useURL = baseURL + "/api/sconfig/relays" + "?running=true&ip=" + hostIP
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
	domainsToRenew := []string{}
	for _, r := range data {
		rn := r["name"].(string) + "." + r["domain"].(string)
		domainsToRenew = append(domainsToRenew, rn)
		rootDomains[r["domain"].(string)] = true
	}
	for rd, _ := range rootDomains {
		domainsToRenew = append(domainsToRenew, rd)
	}
	return domainsToRenew
}

func getHaproxyCfg() {
	configURL := fmt.Sprintf("%s/api/sconfig/haproxy/123", baseURL)
	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()

	configPath := "haproxy.cfg.new"
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		log.Fatalf("Error occurred while creating file. Error is: %s", err.Error())
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatalf("Error occurred while writing to file. Error is: %s", err.Error())
	}

	oldPath := "haproxy.cfg"
	newPath := "haproxy.cfg.new"

	oldBytes, err := os.ReadFile(oldPath)
	if err != nil {
		log.Fatalf("Error occurred while reading old config file. Error is: %s", err.Error())
		return
	}
	newBytes, err := os.ReadFile(newPath)
	if err != nil {
		log.Fatalf("Error occurred while reading new config file. Error is: %s", err.Error())
		return
	}

	if string(oldBytes) == string(newBytes) {
		fmt.Println("No changes detected")
		return
	} else {
		fmt.Println("Changes detected")
		d := diffmatchpatch.New()
		diffs := d.DiffMain(string(oldBytes), string(newBytes), false)
		fmt.Println(d.DiffPrettyText(diffs))

		// haproxy config test
		goodConfig := runCmd("haproxy", []string{"-c", "-f", newPath})
		if goodConfig {
			// reload haproxy with new config
			os.Rename(newPath, oldPath)
			runCmd("systemctl", []string{"reload", "haproxy"})
		} else {

			log.Fatal("Invalid haproxy config, bailing out")
		}
	}
}

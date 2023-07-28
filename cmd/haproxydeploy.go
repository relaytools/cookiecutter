package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
)

var haproxydeploy = &cobra.Command{
	Use:   "haproxydeploy",
	Short: "deploy haproxy",
	Long:  `deploy haproxy`,
	Run: func(cmd *cobra.Command, args []string) {

		ev := signEventWithLoginToken()

		fmt.Println(ev)

		csrf := getCSRF()
		fmt.Println(csrf)

		performLogin(ev, csrf)

		checkAndRestartHaproxy()
	},
}

func init() {
	rootCmd.AddCommand(haproxydeploy)
}

func checkAndRestartHaproxy() {
	getHaproxyCfg()

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

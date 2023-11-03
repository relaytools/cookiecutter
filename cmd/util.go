package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
)

var baseURL string
var jar, _ = cookiejar.New(nil)

var client http.Client

func runCmd(cmd string, args []string) bool {
	log.Printf("%s %v", cmd, args)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		log.Println(string(out))
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 1 {
			return false
		}
		log.Fatal(err)
	}
	log.Printf("%s", out)
	return true
}

func signEventWithLoginToken() nostr.Event {

	privateKey := viper.GetString("PRIVATE_KEY")
	if privateKey == "" {
		fmt.Println("PRIVATE_KEY environment variable is not set")
		os.Exit(1)
	}

	req, err := http.NewRequest("GET", baseURL+"/api/auth/logintoken", nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()
	var data map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &data)

	fmt.Println(data["token"])
	pub, _ := nostr.GetPublicKey(privateKey)

	// create event to sign
	ev := nostr.Event{
		PubKey:    pub,
		CreatedAt: nostr.Now(),
		Kind:      27235,
		Tags:      nil,
		Content:   fmt.Sprint(data["token"]),
	}
	ev.Sign(privateKey)
	return ev
}

func getCSRF() string {
	req, err := http.NewRequest("GET", baseURL+"/api/auth/csrf", nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()
	var csrfData map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &csrfData)
	return fmt.Sprint(csrfData["csrfToken"])
}

func performLogin(ev nostr.Event, csrf string) {
	form := url.Values{}
	form.Set("kind", fmt.Sprint(ev.Kind))
	form.Set("content", ev.Content)
	form.Set("created_at", fmt.Sprint(ev.CreatedAt))
	form.Set("pubkey", ev.PubKey)
	form.Set("sig", ev.Sig)
	form.Set("id", ev.ID)
	form.Set("csrfToken", csrf)
	form.Set("callbackUrl", baseURL)
	form.Set("json", "true")

	req, err := http.NewRequest("POST", baseURL+"/api/auth/callback/credentials", bytes.NewBufferString(form.Encode()))
	if err != nil {
		log.Fatalf("Error occurred while creating request. Error is: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occurred while making request. Error is: %s", err.Error())
	}
	defer resp.Body.Close()
}

func init() {
	client = http.Client{
		Jar: jar,
	}

	viper.AutomaticEnv()
	viper.BindEnv("PRIVATE_KEY")
	viper.BindEnv("BASE_URL")

	baseURL = viper.GetString("BASE_URL")
	if baseURL == "" {
		fmt.Println("BASE_URL environment variable is not set")
		os.Exit(1)
	}
}

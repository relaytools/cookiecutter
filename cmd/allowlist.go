package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Define a struct for the POST body data
type PostBody struct {
	Pubkey string `json:"pubkey"`
	Reason string `json:"reason"`
}

var action = &cobra.Command{
	Use:   "action",
	Short: "action",
	Long:  `action`,
	Run: func(cmd *cobra.Command, args []string) {
		pubkey := viper.GetString("pubkey")
		relayID := viper.GetString("relay")
		log.Printf("pubkey: %s, relayID: %s", pubkey, relayID)
	},
}

func doPost(pubkey string, reason string, useURL string) {
	if pubkey != "" {
		// add the pubkey
		postBody := PostBody{
			Pubkey: pubkey,
			Reason: reason,
		}
		jsonData, err := json.Marshal(postBody)
		if err != nil {
			log.Fatalf("Error encoding JSON: %v", err)
		}

		// Create a new buffer with the JSON data
		buffer := bytes.NewBuffer(jsonData)

		req, err := http.NewRequest("POST", useURL, buffer)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()
		var data []map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &data)
		if resp.StatusCode == 200 {
			log.Println("success.")
		} else {
			log.Printf("error response: %d, %v", resp.StatusCode, data)
			os.Exit(1)
		}
	} else {
		log.Println("pubkey was empty")
	}
}

func doDelete(pubkey string, useURL string) {
	if pubkey != "" {
		// remove the pubkey
		useURL = useURL + "?pubkey=" + pubkey
		req, err := http.NewRequest("DELETE", useURL, nil)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error making request: %v", err)
		}
		defer resp.Body.Close()
		var data []map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &data)
		if resp.StatusCode == 200 {
			log.Println("success.")
		} else {
			log.Printf("error response: %d, %v", resp.StatusCode, data)
			os.Exit(1)
		}
	} else {
		log.Println("pubkey was empty")
	}

}

var allowlist = &cobra.Command{
	Use:   "allowlist",
	Short: "allowlist",
	Long:  `allowlist`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var allowlistadd = &cobra.Command{
	Use:   "add",
	Short: "add",
	Long:  `add`,
	Run: func(cmd *cobra.Command, args []string) {
		pubkey := viper.GetString("pubkey")
		relayID := viper.GetString("relay")
		reason := viper.GetString("reason")

		if relayID == "" {
			log.Fatal("--relay required")
		}

		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)

		useURL := baseURL + "/api/relay/" + relayID + "/allowlistpubkey"
		log.Println(useURL)
		doPost(pubkey, reason, useURL)
	},
}

var allowlistremove = &cobra.Command{
	Use:   "remove",
	Short: "remove",
	Long:  `remove`,
	Run: func(cmd *cobra.Command, args []string) {
		pubkey := viper.GetString("pubkey")
		relayID := viper.GetString("relay")

		if relayID == "" {
			log.Fatal("--relay required")
		}

		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)

		useURL := baseURL + "/api/relay/" + relayID + "/allowlistpubkey"
		doDelete(pubkey, useURL)
	},
}

var blocklistadd = &cobra.Command{
	Use:   "add",
	Short: "add",
	Long:  `add`,
	Run: func(cmd *cobra.Command, args []string) {
		pubkey := viper.GetString("pubkey")
		relayID := viper.GetString("relay")
		reason := viper.GetString("reason")

		if relayID == "" {
			log.Fatal("--relay required")
		}

		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)
		useURL := baseURL + "/api/relay/" + relayID + "/blocklistpubkey"
		doPost(pubkey, reason, useURL)

	},
}

var blocklist = &cobra.Command{
	Use:   "blocklist",
	Short: "blocklist",
	Long:  `blocklist`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	action.PersistentFlags().String("pubkey", "", "public key npub or hex")
	action.PersistentFlags().String("relay", "", "relay id")
	action.PersistentFlags().String("reason", "", "reason")
	viper.BindPFlag("pubkey", action.PersistentFlags().Lookup("pubkey"))
	viper.BindPFlag("relay", action.PersistentFlags().Lookup("relay"))
	viper.BindPFlag("reason", action.PersistentFlags().Lookup("reason"))
	rootCmd.AddCommand(action)
	action.AddCommand(allowlist)
	action.AddCommand(blocklist)
	allowlist.AddCommand(allowlistadd)
	allowlist.AddCommand(allowlistremove)
	blocklist.AddCommand(blocklistadd)
}

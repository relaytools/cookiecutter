package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Define a struct for the POST body data
type PostBody struct {
	Pubkey string `json:"pubkey"`
	Reason string `json:"reason"`
}

var allowlist = &cobra.Command{
	Use:   "allowlist",
	Short: "allowlist",
	Long:  `allowlist`,
	Run: func(cmd *cobra.Command, args []string) {
		pubkey := viper.GetString("pubkey")
		relayID := viper.GetString("relayid")

		if relayID == "" {
			log.Fatal("--relayid required")
		}

		ev := signEventWithLoginToken()
		csrf := getCSRF()
		performLogin(ev, csrf)

		if pubkey != "" {
			// add the pubkey
			useURL := baseURL + "/api/relay/" + relayID + "/allowlistpubkey"
			postBody := PostBody{
				Pubkey: pubkey,
				Reason: "cookie",
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

		}
	},
}

func init() {
	allowlist.PersistentFlags().String("pubkey", "", "public key to add to allowlist")
	allowlist.PersistentFlags().String("relayid", "", "relay id")
	viper.BindPFlag("pubkey", allowlist.PersistentFlags().Lookup("pubkey"))
	viper.BindPFlag("relayid", allowlist.PersistentFlags().Lookup("relayid"))
	rootCmd.AddCommand(allowlist)
}

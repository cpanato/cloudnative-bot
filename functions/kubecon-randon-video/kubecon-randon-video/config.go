package function

import (
	"encoding/json"
	"io/ioutil"
)

// HonkConfig struct to hold the honk configuration
type HonkConfig struct {
	TwitterConsumerKey       string
	TwitterConsumerSecretKey string
	TwitterAccessToken       string
	TwitterAccessSecret      string
	TwitterSearchCounts      string

	TokenAPIKey string

	YouTubeAPIKey string
}

// Config for honk bot
var Config = &HonkConfig{}

func loadConfig() error {
	secretBytes, err := ioutil.ReadFile("/var/openfaas/secrets/config.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(secretBytes, &Config)
	if err != nil {
		return err
	}

	return nil
}

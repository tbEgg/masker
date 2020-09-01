package socks

import (
	"encoding/json"
	"io/ioutil"
)

type socksConfig struct {
	Authentication string `json:"method"`
	authMethod     byte
}

func loadConfig(configFile string) (config socksConfig, err error) {
	rawData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(rawData, &config)

	if config.Authentication == "password" {
		config.authMethod = authUserPass
	} else {
		config.authMethod = authNotRequired
	}
	return
}

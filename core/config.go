package core

import (
	"encoding/json"
	"io/ioutil"
)

// the config for a node
type NodeConfig struct {
	ListenEndConfig ConnectionConfig `json:"listener"`
	CallEndConfig   ConnectionConfig `json:"caller"`
	Port            uint16           `json:"port"`
}

type ConnectionConfig struct {
	Protocol   string `json:"protocol"`
	ConfigFile string `json:"config"`
}

func LoadConfig(configFile string) (config NodeConfig, err error) {
	rawData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(rawData, &config)
	return
}

// Package config provides configuration loading for Sheepdog.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Processes []ProcessConfig `json:"processes"`
}

type ProcessConfig struct {
	Name       string          `json:"name"`
	Command    []string        `json:"command"`
	Autorun    bool            `json:"autorun"`
	Cwd        string          `json:"cwd"`
	ReadyRegexp string          `json:"readyRegexp"`
	Children   []ProcessConfig `json:"children"`
	GroupType  string          `json:"GroupType"`
}

func LoadConfig(path string) (Config, error) {
	config := Config{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, fmt.Errorf("config file '%s' does not exist", path)
		}
		return config, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return config, err
	}

	return config, nil
}

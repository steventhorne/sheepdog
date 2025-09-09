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
	Name        string          `json:"name"`        // required
	Command     []string        `json:"command"`     // required for non process groups
	Autorun     bool            `json:"autorun"`     // optional
	Cwd         string          `json:"cwd"`         // optional
	ReadyRegexp string          `json:"readyRegexp"` // optional
	Children    []ProcessConfig `json:"children"`    // required for process groups
	GroupType   string          `json:"groupType"`   // required for process groups
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

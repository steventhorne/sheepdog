package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Processes []ProcessConfig `json:"processes"`
}

type ProcessConfig struct {
	Name    string `json:"name"`
	Command []string `json:"command"`
}

func LoadConfig(path string) (Config, error) {
	config := Config{}

	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return config, err
	}

	return config, nil
}

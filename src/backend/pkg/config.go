package pkg

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
)

// Config: struct to hold the configuration.
type Config struct {
	DbURI        string `json:"db_uri"`
	DbName       string `json:"db_name"`
	JwtSecretKey string `json:"jwt_secret_key"`
	ServerHost   string `json:"server_host"`
	ServerPort   string `json:"server_port"`
	ConfigPath   string // path to the config file
}

// chooseConfigFile: chooses the config file.
// Prefers the local config over the global config.
// Returns the config file path and an error if the file is not found.
func chooseConfigFile() (string, error) {
	configPathLocal, err := filepath.Abs(constants.CONFIG_FILE_LOCAL)
	if err == nil {
		if _, err := os.Stat(configPathLocal); err == nil {
			return configPathLocal, nil
		}
	}

	configPath, err := filepath.Abs(constants.CONFIG_FILE)
	if err == nil {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", errors.New("config file not found. tried both " + configPathLocal + " and " + configPath)
}

// LoadConfig: loads the config file.
// Returns the config and an error if the file is not found.
func LoadConfig(configPath *string) (*Config, error) {
	var configPathString string
	var err error

	if configPath == nil {
		configPathString, err = chooseConfigFile()
		if err != nil {
			return nil, err
		}
	} else {
		configPathString = *configPath
	}

	jsonData, err := os.ReadFile(configPathString)
	if err != nil {
		return nil, err
	}

	var config *Config = &Config{}
	err = json.Unmarshal(jsonData, config)
	if err != nil {
		return nil, err
	}
	config.ConfigPath = configPathString
	return config, nil
}

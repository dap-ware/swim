package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config represents the configuration structure.
type Config struct {
	Database struct {
		FilePath  string `json:"filepath"`
		BatchSize int    `json:"batchsize"`
	}
	Rate struct {
		Limit     int           `json:"limit"`
		ResetTime time.Duration `json:"resettime"`
	}
	// ... future config options
}

// LoadConfig reads a JSON file and unmarshals it into a Config struct.
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetDefaultConfig provides default configuration settings.
func GetDefaultConfig() *Config {
	return &Config{
		Database: struct {
			FilePath  string `json:"filepath"`
			BatchSize int    `json:"batchsize"`
		}{
			FilePath:  "data/swim.db",
			BatchSize: 1000,
		},
		Rate: struct {
			Limit     int           `json:"limit"`
			ResetTime time.Duration `json:"resettime"`
		}{
			Limit:     1000,
			ResetTime: 60 * time.Second,
		},
	}
}

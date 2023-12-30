package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration structure.
type Config struct {
	Database struct {
		FilePath  string `yaml:"filepath"`
		BatchSize int    `yaml:"batchSize"`
	}
	Rate struct {
		Limit     int           `yaml:"limit"`
		ResetTime time.Duration `yaml:"resetTime"`
	}
	// ... future config options
}

// LoadConfig reads a YAML file and unmarshals it into a Config struct.
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

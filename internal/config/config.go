package config

import (
	"fmt"
	"os"
)

// Config holds application settings
type Config struct {
	MasterPassword string
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	masterPwd := os.Getenv("GOSECRETS_MASTER_PWD")
	if masterPwd == "" {
		return nil, fmt.Errorf("GOSECRETS_MASTER_PWD environment variable is required")
	}

	return &Config{
		MasterPassword: masterPwd,
	}, nil
}

package config

import (
	"fmt"
	"log"
	"os"
)

// Config holds application settings
type Config struct {
	MasterPassword string
	WebhookURL     string
	MetricsPort    int
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	masterPwd := os.Getenv("GOSECRETS_MASTER_PWD")
	if masterPwd == "" {
		return nil, fmt.Errorf("GOSECRETS_MASTER_PWD environment variable is required")
	}

	metricsPort := 2112
	if portStr := os.Getenv("GOSECRETS_METRICS_PORT"); portStr != "" {
		if _, err := fmt.Sscanf(portStr, "%d", &metricsPort); err != nil {
			log.Printf("invalid GOSECRETS_METRICS_PORT %q, using default %d: %v", portStr, metricsPort, err)
		}
	}

	return &Config{
		MasterPassword: masterPwd,
		WebhookURL:     os.Getenv("GOSECRETS_WEBHOOK_URL"),
		MetricsPort:    metricsPort,
	}, nil
}

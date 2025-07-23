package main

import (
	"agent/models"
	"agent/theme"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed default-config.json
var defaultConfigJSON []byte

// Config represents the persistent agent configuration
type Config struct {
	Providers     []*models.Provider `json:"providers"`
	Model         *SelectedModel     `json:"model"`
	MaxIterations int                `json:"max_iterations"`
}

// SelectedModel represents the currently selected model
type SelectedModel struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

const configFileName = "config.json"

// getConfigPath returns the path to the configuration file in ~/.agent/
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	agentDir := filepath.Join(homeDir, ".agent")

	// Create ~/.agent directory if it doesn't exist
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create agent directory: %w", err)
	}

	return filepath.Join(agentDir, configFileName), nil
}

// LoadConfig loads the configuration from file, creating defaults if it doesn't exist or is corrupted
func LoadConfig() *Config {
	configPath, err := getConfigPath()
	if err != nil {
		return createDefaultConfig()
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config := createDefaultConfig()
		SaveConfig(config)
		return config
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return createDefaultConfig()
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println(theme.WarningText("Warning: Config file is corrupted"))
		return createDefaultConfig()
	}

	return &config
}

// SaveConfig saves the configuration to file
func SaveConfig(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func createDefaultConfig() *Config {
	var config Config
	if err := json.Unmarshal(defaultConfigJSON, &config); err != nil {
		panic(err)
	}
	return &config
}

package main

import (
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
	Debug         bool                   `json:"debug"`
	Providers     json.RawMessage        `json:"providers"`
	Model         *SelectedModel         `json:"model"`
	MaxIterations int                    `json:"max_iterations"`
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
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config, err := createDefaultConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		if err := SaveConfig(config); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}
	
	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		// Config file is corrupted, create default and overwrite
		fmt.Printf("Warning: Config file is corrupted, creating new default config\n")
		defaultConfig, err := createDefaultConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		if err := SaveConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return defaultConfig, nil
	}
	
	// Set default MaxIterations if not configured
	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}
	
	return &config, nil
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

// createDefaultConfig creates a default configuration from embedded JSON
func createDefaultConfig() (*Config, error) {
	var config Config
	if err := json.Unmarshal(defaultConfigJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default config: %w", err)
	}
	
	return &config, nil
}



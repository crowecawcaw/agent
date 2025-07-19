package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadRegistry loads providers and models from JSON data
func LoadRegistry(data json.RawMessage) *Registry {
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		panic(fmt.Sprintf("Failed to parse registry JSON: %v", err))
	}

	// Set up bidirectional relationships and resolve API keys
	for _, provider := range registry.Providers {
		provider.APIKey = resolveAPIKey(provider.APIKey, provider.ID)

		for _, model := range provider.Models {
			model.Provider = provider
		}
	}

	return &registry
}

// resolveAPIKey resolves the API key from config value
func resolveAPIKey(configValue, providerID string) string {
	// Check if it's an environment variable reference
	if strings.HasPrefix(configValue, "env:") {
		envVar := strings.TrimPrefix(configValue, "env:")
		return os.Getenv(envVar)
	}

	// Return the direct key value
	return configValue
}

// Registry helper methods

// GetModel searches all providers for a model with the given ID
func (r *Registry) GetModel(modelID string) *Model {
	for _, provider := range r.Providers {
		for _, model := range provider.Models {
			if model.ID == modelID {
				return model
			}
		}
	}
	return nil
}

// GetProvider returns a provider by ID
func (r *Registry) GetProvider(providerID string) *Provider {
	for _, provider := range r.Providers {
		if provider.ID == providerID {
			return provider
		}
	}
	return nil
}

// GetModelByProviderAndID searches for a model with the given ID within a specific provider
func (r *Registry) GetModelByProviderAndID(providerID, modelID string) *Model {
	provider := r.GetProvider(providerID)
	if provider == nil {
		return nil
	}

	for _, model := range provider.Models {
		if model.ID == modelID {
			return model
		}
	}
	return nil
}

// GetModelsForProvider returns all models for a specific provider
func (r *Registry) GetModelsForProvider(providerID string) []*Model {
	provider := r.GetProvider(providerID)
	if provider == nil {
		return nil
	}
	return provider.Models
}

package models

import (
	"context"
	"time"
)

// Provider represents a static provider configuration with its models
type Provider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	BaseURL string   `json:"base_url"`
	APIKey  string   `json:"api_key,omitempty"` // Can be env:VAR_NAME or direct key
	Models  []*Model `json:"models"`
}

// Model represents a static model configuration
type Model struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Config   ModelConfig `json:"config"`
	Provider *Provider   `json:"-"` // Back-reference, not serialized
}

// ModelConfig holds model-specific configuration
type ModelConfig struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// Message represents a conversation message
type Message struct {
	ID         string     `json:"id"` // Unique ID for the message across its lifecycle
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Timestamp  time.Time  `json:"timestamp"`
	ToolName   string     `json:"tool_name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Status     string     `json:"status,omitempty"` // e.g., "active", "edited", "deleted"
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call within a tool call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// ToolFunc defines the signature for tool functions
// Returns: (userMessage, agentMessage, error)
// - userMessage: Rich message for human display (empty string if tool printed directly)
// - agentMessage: Minimal status for the agent
// - error: Any error that occurred
type ToolFunc func(ctx context.Context, params map[string]interface{}) (string, string, error)

// ToolDefinition contains metadata and function for a tool
type ToolDefinition struct {
	Name        string
	Description string
	Schema      map[string]interface{}
	Func        ToolFunc
}

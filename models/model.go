package models

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
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
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

// ToolUseResponse represents a tool use request from the model
type ToolUseResponse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// Registry manages providers and models loaded from JSON
type Registry struct {
	Providers []*Provider `json:"providers"`
}



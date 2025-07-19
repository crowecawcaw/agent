package tools

import (
	"context"
	"fmt"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	// Returns a string for the LLM to read with the result
	// statusCh receives status updates during execution
	Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error)
}

// BaseTool provides common functionality for all tools
type BaseTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

// NewBaseTool creates a new base tool with the given properties
func NewBaseTool(name, description string, schema map[string]interface{}) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
	}
}

// Name returns the tool's name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool's description
func (t *BaseTool) Description() string {
	return t.description
}

// InputSchema returns the tool's input schema
func (t *BaseTool) InputSchema() map[string]interface{} {
	return t.schema
}

// Execute provides a default implementation that sends a basic status update
func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	statusCh <- fmt.Sprintf("Executing %s", t.name)
	return "", fmt.Errorf("tool %s does not implement Execute method", t.name)
}

// ToolError represents an error that occurred during tool execution
type ToolError struct {
	ToolName string
	Message  string
	Cause    error
}

// Error implements the error interface
func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' error: %s (caused by: %v)", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s' error: %s", e.ToolName, e.Message)
}

// Unwrap returns the underlying cause
func (e *ToolError) Unwrap() error {
	return e.Cause
}

// NewToolError creates a new tool error
func NewToolError(toolName, message string, cause error) *ToolError {
	return &ToolError{
		ToolName: toolName,
		Message:  message,
		Cause:    cause,
	}
}

// WrapToolError wraps an error as a tool error
func WrapToolError(toolName string, err error) *ToolError {
	if err == nil {
		return nil
	}

	if toolErr, ok := err.(*ToolError); ok {
		return toolErr // Don't double-wrap
	}

	return &ToolError{
		ToolName: toolName,
		Message:  err.Error(),
		Cause:    err,
	}
}



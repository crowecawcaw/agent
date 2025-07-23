package tools

import (
	"fmt"
)

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

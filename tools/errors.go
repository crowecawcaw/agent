package tools

import (
	"fmt"
	"log"
)

// ErrorHandler provides centralized error handling for tools
type ErrorHandler struct {
	debugEnabled bool
}

// NewErrorHandler creates a new error handler for tools
func NewErrorHandler(debugEnabled bool) *ErrorHandler {
	return &ErrorHandler{
		debugEnabled: debugEnabled,
	}
}

// isDebugEnabled checks if debug mode is enabled
func (eh *ErrorHandler) isDebugEnabled() bool {
	return eh.debugEnabled
}

// HandleToolError handles tool-specific errors with consistent formatting
func (eh *ErrorHandler) HandleToolError(toolName string, err error) string {
	if err == nil {
		return ""
	}

	if eh.isDebugEnabled() {
		log.Printf("Tool '%s' error: %v", toolName, err)
	}
	return fmt.Sprintf("❌ %s failed: %v", toolName, err)
}

// HandleValidationError handles validation errors with consistent formatting
func (eh *ErrorHandler) HandleValidationError(toolName string, err error) string {
	if err == nil {
		return ""
	}

	if eh.isDebugEnabled() {
		log.Printf("Tool '%s' validation error: %v", toolName, err)
	}
	return fmt.Sprintf("❌ %s validation failed: %v", toolName, err)
}

// LogError logs an error without returning a formatted string
func (eh *ErrorHandler) LogError(context string, err error) {
	if err != nil && eh.isDebugEnabled() {
		log.Printf("%s error: %v", context, err)
	}
}

// Global error handler instance for tools
var globalToolErrorHandler *ErrorHandler

// InitializeToolErrorHandler initializes the global tool error handler
func InitializeToolErrorHandler(debugEnabled bool) {
	globalToolErrorHandler = NewErrorHandler(debugEnabled)
}

// Convenience functions that use the global error handler
func HandleToolError(toolName string, err error) string {
	if globalToolErrorHandler == nil {
		globalToolErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	return globalToolErrorHandler.HandleToolError(toolName, err)
}

func HandleValidationError(toolName string, err error) string {
	if globalToolErrorHandler == nil {
		globalToolErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	return globalToolErrorHandler.HandleValidationError(toolName, err)
}

func LogError(context string, err error) {
	if globalToolErrorHandler == nil {
		globalToolErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	globalToolErrorHandler.LogError(context, err)
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// ErrorContext provides structured error context
type ErrorContext struct {
	Operation string
	Component string
	Details   map[string]interface{}
}

// ErrorHandler provides centralized error handling with consistent formatting
type ErrorHandler struct {
	debugEnabled bool
}

// NewErrorHandler creates a new error handler
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

// HandleSystemError handles system-level errors with context
func (eh *ErrorHandler) HandleSystemError(ctx ErrorContext, err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, context.Canceled) {
		return "Cancelled request"
	}

	if eh.isDebugEnabled() {
		log.Printf("%s/%s error: %v", ctx.Component, ctx.Operation, err)
		if len(ctx.Details) > 0 {
			log.Printf("Error details: %+v", ctx.Details)
		}
	}

	return fmt.Sprintf("❌ %s failed: %v", ctx.Operation, err)
}

// LogWarning logs warnings consistently
func (eh *ErrorHandler) LogWarning(component, operation string, err error) {
	if err != nil && eh.isDebugEnabled() {
		log.Printf("Warning: %s/%s: %v", component, operation, err)
	}
}

// WrapError wraps an error with additional context
func (eh *ErrorHandler) WrapError(err error, operation string, details ...interface{}) error {
	if err == nil {
		return nil
	}

	if len(details) > 0 {
		return fmt.Errorf("%s: %w (details: %v)", operation, err, details)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Global error handler instance
var globalErrorHandler *ErrorHandler

// InitializeErrorHandler initializes the global error handler
func InitializeErrorHandler(debugEnabled bool) {
	globalErrorHandler = NewErrorHandler(debugEnabled)
}

// Convenience functions that use the global error handler
func HandleToolError(toolName string, err error) string {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	return globalErrorHandler.HandleToolError(toolName, err)
}

func HandleValidationError(toolName string, err error) string {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	return globalErrorHandler.HandleValidationError(toolName, err)
}

func HandleSystemError(operation string, err error) string {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	return globalErrorHandler.HandleSystemError(ErrorContext{
		Operation: operation,
		Component: "system",
	}, err)
}

func LogError(context string, err error) {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	globalErrorHandler.LogWarning("system", context, err)
}

func LogWarning(component, operation string, err error) {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(false) // Default to false if not initialized
	}
	globalErrorHandler.LogWarning(component, operation, err)
}

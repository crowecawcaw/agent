package tools

import (
	"agent/models"
	"context"
	"fmt"
)

// LiveContextManager interface for managing live context
type LiveContextManager interface {
	AddFile(path string, startLine int, endLine *int) error
	RemoveFile(path string) error
	ListFiles() []string
	AddDirectory(path string, ignoreGitignore bool, ignorePatterns ...string) error
	RemoveDirectory(path string) error
	ListDirectories() []string
	SerializeFiles() string
	SerializeDirectories() string
}

// NewReadFileTool creates the read_file tool
func NewReadFileTool(liveContext LiveContextManager) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to add to context",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Starting line number (1-based)",
				"minimum":     1,
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Ending line number (1-based)",
				"minimum":     1,
			},
		},
		"required": []string{"path"},
	}

	return models.ToolDefinition{
		Name:        "read_file",
		Description: "Read a file's contents. The file will be automatically included with current data in every request. Use this instead of shell commands like 'cat' to read files.",
		Schema:      schema,
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return readFile(ctx, params, liveContext)
		},
	}
}

// NewStopReadingFileTool creates the stop_reading_file tool
func NewStopReadingFileTool(liveContext LiveContextManager) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to remove from context",
			},
		},
		"required": []string{"path"},
	}

	return models.ToolDefinition{
		Name:        "stop_reading_file",
		Description: "Stop reading a file when you no longer need its contents.",
		Schema:      schema,
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return stopReadingFile(ctx, params, liveContext)
		},
	}
}

// NewReadDirectoryTool creates the read_directory tool
func NewReadDirectoryTool(liveContext LiveContextManager) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the directory to add to context",
			},
			"ignore_gitignore": map[string]interface{}{
				"type":        "boolean",
				"description": "Optional: Whether to ignore .gitignore rules (default: false)",
				"default":     false,
			},
			"ignore_patterns": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Optional: Additional patterns to ignore (glob format)",
			},
		},
		"required": []string{"path"},
	}

	return models.ToolDefinition{
		Name:        "read_directory",
		Description: "Read a directory's nested file structure as a flat list. Use this instead of shell commands like 'ls' or 'find' to explore directories.",
		Schema:      schema,
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return readDirectory(ctx, params, liveContext)
		},
	}
}

// NewStopReadingDirectoryTool creates the stop_reading_directory tool
func NewStopReadingDirectoryTool(liveContext LiveContextManager) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the directory to remove from context",
			},
		},
		"required": []string{"path"},
	}

	return models.ToolDefinition{
		Name:        "stop_reading_directory",
		Description: "Stop reading a directory when you no longer need to see its structure.",
		Schema:      schema,
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return stopReadingDirectory(ctx, params, liveContext)
		},
	}
}

// readFile implements the read file functionality
func readFile(ctx context.Context, params map[string]interface{}, liveContext LiveContextManager) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	var startLine int
	var endLine *int
	if sl, ok := params["start_line"].(float64); ok {
		startLine = int(sl)
	}
	if el, ok := params["end_line"].(float64); ok {
		endLineVal := int(el)
		endLine = &endLineVal
	}

	if err := liveContext.AddFile(path, startLine, endLine); err != nil {
		return "", "", WrapToolError("read_file", err)
	}

	if startLine > 0 || endLine != nil {
		endLineStr := "end"
		if endLine != nil {
			endLineStr = fmt.Sprintf("%d", *endLine)
		}
		return fmt.Sprintf("Reading file %s (lines %d-%s)\n", path, startLine, endLineStr), "Reading", nil
	}
	return fmt.Sprintf("Reading file %s\n", path), "Reading", nil
}

// stopReadingFile implements the stop reading file functionality
func stopReadingFile(ctx context.Context, params map[string]interface{}, liveContext LiveContextManager) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	if err := liveContext.RemoveFile(path); err != nil {
		return "", "", WrapToolError("stop_reading_file", err)
	}

	return fmt.Sprintf("Stopped reading file %s\n", path), "Stopped", nil
}

// readDirectory implements the read directory functionality
func readDirectory(ctx context.Context, params map[string]interface{}, liveContext LiveContextManager) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	ignoreGitignore := false
	if ig, ok := params["ignore_gitignore"].(bool); ok {
		ignoreGitignore = ig
	}

	var ignorePatterns []string
	if patterns, ok := params["ignore_patterns"].([]interface{}); ok {
		for _, pattern := range patterns {
			if str, ok := pattern.(string); ok {
				ignorePatterns = append(ignorePatterns, str)
			}
		}
	}

	if err := liveContext.AddDirectory(path, ignoreGitignore, ignorePatterns...); err != nil {
		return "", "", WrapToolError("read_directory", err)
	}

	return fmt.Sprintf("Reading directory %s\n", path), "Reading", nil
}

// stopReadingDirectory implements the stop reading directory functionality
func stopReadingDirectory(ctx context.Context, params map[string]interface{}, liveContext LiveContextManager) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	if err := liveContext.RemoveDirectory(path); err != nil {
		return "", "", WrapToolError("stop_reading_directory", err)
	}

	return fmt.Sprintf("Stopped reading directory %s\n", path), "Stopped", nil
}

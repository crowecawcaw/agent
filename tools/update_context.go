package tools

import (
	"context"
	"fmt"
	"strings"

	"agent/theme"
)

type UpdateContextTool struct {
	*BaseTool
	liveContext LiveContextManager
}

type LiveContextManager interface {
	AddFile(filePath string, startLine int, endLine *int) error
	RemoveFile(filePath string) error
	ListFiles() []string
	AddDirectory(dirPath string, ignoreGitignore bool, ignorePatterns ...string) error
	RemoveDirectory(dirPath string) error
	ListDirectories() []string
}

func NewUpdateContextTool(liveContext LiveContextManager) *UpdateContextTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"add_file": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to add to live context",
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "Starting line number, 1-based",
						"default":     1,
						"minimum":     1,
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "Ending line number, 1-based, can be negative",
					},
				},
				"required":    []interface{}{"path"},
				"description": "File path to add to live context with optional parameters",
			},
			"remove_file": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to remove from live context",
					},
				},
				"required":    []interface{}{"path"},
				"description": "File path to remove from live context",
			},
			"add_directory": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to add to live context",
					},
					"ignore_gitignore": map[string]interface{}{
						"type":        "boolean",
						"description": "Respect .gitignore files",
						"default":     true,
					},
					"ignore_patterns": map[string]interface{}{
						"type":        "string",
						"description": "Comma-separated git glob patterns to ignore (e.g., '*.tmp,build/*')",
					},
				},
				"required":    []interface{}{"path"},
				"description": "Directory path to add to live context with optional parameters",
			},
			"remove_directory": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to remove from live context",
					},
				},
				"required":    []interface{}{"path"},
				"description": "Directory path to remove from live context",
			},
		},
		"description": "Manage files and directories in the live context. At least one operation must be provided.",
	}

	return &UpdateContextTool{
		BaseTool: NewBaseTool(
			"update_context",
			"", // Empty description since we'll override the Description() method
			schema,
		),
		liveContext: liveContext,
	}
}

// Description returns a dynamic description that includes current live context status
func (t *UpdateContextTool) Description() string {
	var descBuilder strings.Builder
	descBuilder.WriteString("Add or remove files and directories from the live context to read their contents and explore structure. This is the primary way to access file contents and directory structures. Live context contents are AUTOMATICALLY REFRESHED before every AI call - you never need to re-add files to get updated content. Use this instead of separate file reading tools. You can add multiple files and directories at a time. If files may be useful, add them to context quickly. Requesting them one by one is slower. If a file is not relevant or helpful, remove it. Irrelevant files and directories pollute your context and decrease your accuracy.\n\n")

	// Add example using multiline string
	descBuilder.WriteString(`EXAMPLE - Adding and removing multiple items at once:
{
  "add_file": [
    {"path": "/path/to/main.go"},
    {"path": "/path/to/config.json"}
  ],
  "add_directory": [
    {"path": "/path/to/src"},
    {"path": "/path/to/tests"}
  ],
  "remove_file": [
    {"path": "/old/file.txt"}
  ],
  "remove_directory": [
    {"path": "/old/directory"}
  ]
}

`)

	// Add current context status
	currentFiles := t.liveContext.ListFiles()
	currentDirs := t.liveContext.ListDirectories()

	descBuilder.WriteString("CURRENT LIVE CONTEXT:\n")

	descBuilder.WriteString("Files: \n")
	for _, file := range currentFiles {
		descBuilder.WriteString(" - " + file + "\n")
	}
	if len(currentFiles) == 0 {
		descBuilder.WriteString(" (None)\n")
	}

	descBuilder.WriteString("Directories: \n")
	for _, dir := range currentDirs {
		descBuilder.WriteString(" - " + dir + "\n")
	}
	if len(currentDirs) > 0 {
		descBuilder.WriteString(" (None)\n")
	}

	return descBuilder.String()
}

func (t *UpdateContextTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	var addedFiles []string
	var removedFiles []string
	var addedDirs []string
	var removedDirs []string
	var errors []string

	// Helper function to extract file path and attributes from parameter
	extractFileParams := func(param interface{}) (string, int, *int) {
		// Default values
		startLine := 1
		var endLine *int = nil

		switch v := param.(type) {
		case string:
			return v, startLine, endLine
		case map[string]interface{}:
			filePath, _ := v["path"].(string)
			if sl, ok := v["start_line"].(float64); ok {
				startLine = int(sl)
			}
			if el, ok := v["end_line"].(float64); ok {
				elInt := int(el)
				endLine = &elInt
			}
			return filePath, startLine, endLine
		}
		return "", startLine, endLine
	}

	// Helper function to extract directory path and attributes from parameter
	extractDirParams := func(param interface{}) (string, bool, []string) {
		// Default values
		ignoreGitignore := true
		var ignorePatterns []string

		switch v := param.(type) {
		case string:
			return v, ignoreGitignore, ignorePatterns
		case map[string]interface{}:
			dirPath, _ := v["path"].(string)
			if ig, ok := v["ignore_gitignore"].(bool); ok {
				ignoreGitignore = ig
			}
			if ip, ok := v["ignore_patterns"].(string); ok && ip != "" {
				// Split comma-separated patterns and trim whitespace
				patterns := strings.Split(ip, ",")
				for _, pattern := range patterns {
					if trimmed := strings.TrimSpace(pattern); trimmed != "" {
						ignorePatterns = append(ignorePatterns, trimmed)
					}
				}
			}
			return dirPath, ignoreGitignore, ignorePatterns
		}
		return "", ignoreGitignore, ignorePatterns
	}

	// Process add_file parameters
	if addFile, exists := params["add_file"]; exists {
		switch v := addFile.(type) {
		case string, map[string]interface{}:
			filePath, startLine, endLine := extractFileParams(v)
			if filePath == "" {
				errors = append(errors, "Invalid file parameter")
			} else {
				// Validate start_line < end_line if both are specified
				if endLine != nil && startLine >= *endLine {
					errors = append(errors, fmt.Sprintf("start_line (%d) must be less than end_line (%d) for file %s", startLine, *endLine, filePath))
				} else {
					if err := t.liveContext.AddFile(filePath, startLine, endLine); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to add file %s: %v", filePath, err))
					} else {
						addedFiles = append(addedFiles, filePath)
					}
				}
			}
		case []interface{}:
			for _, item := range v {
				filePath, startLine, endLine := extractFileParams(item)
				if filePath == "" {
					errors = append(errors, "Invalid file parameter")
				} else {
					// Validate start_line < end_line if both are specified
					if endLine != nil && startLine >= *endLine {
						errors = append(errors, fmt.Sprintf("start_line (%d) must be less than end_line (%d) for file %s", startLine, *endLine, filePath))
					} else {
						if err := t.liveContext.AddFile(filePath, startLine, endLine); err != nil {
							errors = append(errors, fmt.Sprintf("Failed to add file %s: %v", filePath, err))
						} else {
							addedFiles = append(addedFiles, filePath)
						}
					}
				}
			}
		default:
			errors = append(errors, "Invalid file parameter type")
		}
	}

	// Process add_directory parameters
	if addDir, exists := params["add_directory"]; exists {
		switch v := addDir.(type) {
		case string, map[string]interface{}:
			dirPath, ignoreGitignore, ignorePatterns := extractDirParams(v)
			if dirPath == "" {
				errors = append(errors, "Invalid directory parameter")
			} else {
				if err := t.liveContext.AddDirectory(dirPath, ignoreGitignore, ignorePatterns...); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to add directory %s: %v", dirPath, err))
				} else {
					addedDirs = append(addedDirs, dirPath)
				}
			}
		case []interface{}:
			for _, item := range v {
				dirPath, ignoreGitignore, ignorePatterns := extractDirParams(item)
				if dirPath == "" {
					errors = append(errors, "Invalid directory parameter")
				} else {
					if err := t.liveContext.AddDirectory(dirPath, ignoreGitignore, ignorePatterns...); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to add directory %s: %v", dirPath, err))
					} else {
						addedDirs = append(addedDirs, dirPath)
					}
				}
			}
		default:
			errors = append(errors, "Invalid directory parameter type")
		}
	}

	// Process remove_file parameters
	if removeFile, exists := params["remove_file"]; exists {
		switch v := removeFile.(type) {
		case string:
			if err := t.liveContext.RemoveFile(v); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to remove file %s: %v", v, err))
			} else {
				removedFiles = append(removedFiles, v)
			}
		case map[string]interface{}:
			if path, ok := v["path"].(string); ok {
				if err := t.liveContext.RemoveFile(path); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to remove file %s: %v", path, err))
				} else {
					removedFiles = append(removedFiles, path)
				}
			}
		case []interface{}:
			for _, item := range v {
				switch itemV := item.(type) {
				case string:
					if err := t.liveContext.RemoveFile(itemV); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to remove file %s: %v", itemV, err))
					} else {
						removedFiles = append(removedFiles, itemV)
					}
				case map[string]interface{}:
					if path, ok := itemV["path"].(string); ok {
						if err := t.liveContext.RemoveFile(path); err != nil {
							errors = append(errors, fmt.Sprintf("Failed to remove file %s: %v", path, err))
						} else {
							removedFiles = append(removedFiles, path)
						}
					}
				}
			}
		default:
			errors = append(errors, "Invalid file parameter type")
		}
	}

	// Process remove_directory parameters
	if removeDir, exists := params["remove_directory"]; exists {
		switch v := removeDir.(type) {
		case string:
			if err := t.liveContext.RemoveDirectory(v); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to remove directory %s: %v", v, err))
			} else {
				removedDirs = append(removedDirs, v)
			}
		case map[string]interface{}:
			if path, ok := v["path"].(string); ok {
				if err := t.liveContext.RemoveDirectory(path); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to remove directory %s: %v", path, err))
				} else {
					removedDirs = append(removedDirs, path)
				}
			}
		case []interface{}:
			for _, item := range v {
				switch itemV := item.(type) {
				case string:
					if err := t.liveContext.RemoveDirectory(itemV); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to remove directory %s: %v", itemV, err))
					} else {
						removedDirs = append(removedDirs, itemV)
					}
				case map[string]interface{}:
					if path, ok := itemV["path"].(string); ok {
						if err := t.liveContext.RemoveDirectory(path); err != nil {
							errors = append(errors, fmt.Sprintf("Failed to remove directory %s: %v", path, err))
						} else {
							removedDirs = append(removedDirs, path)
						}
					}
				}
			}
		default:
			errors = append(errors, "Invalid directory parameter type")
		}
	}

	if len(addedFiles) == 0 && len(removedFiles) == 0 && len(addedDirs) == 0 && len(removedDirs) == 0 && len(errors) == 0 {
		return "", fmt.Errorf("no operations specified - provide add_file, remove_file, add_directory, or remove_directory")
	}

	// Generate status update from tool call data
	statusUpdate := GenerateStatusUpdate(addedFiles, removedFiles, addedDirs, removedDirs)

	// Handle errors if any
	var response []string
	if statusUpdate != "No changes to live context." {
		response = append(response, statusUpdate)
	}

	if len(errors) > 0 {
		if len(response) > 0 {
			response = append(response, "")
		}
		response = append(response, "Errors:")
		for _, err := range errors {
			response = append(response, "  "+err)
		}
	}

	statusCh <- statusUpdate

	return "Ok", nil
}

// GenerateStatusUpdate creates a colored diff of files and directories added/removed
func GenerateStatusUpdate(addedFiles, removedFiles, addedDirs, removedDirs []string) string {
	if len(addedFiles) == 0 && len(removedFiles) == 0 &&
		len(addedDirs) == 0 && len(removedDirs) == 0 {
		return "No changes to live context."
	}

	var sb strings.Builder
	sb.WriteString("> Updated context:\n")

	for _, file := range addedFiles {
		sb.WriteString(theme.SuccessText(" + "+file) + "\n")
	}
	for _, file := range removedFiles {
		sb.WriteString(theme.ErrorText(" - "+file) + "\n")
	}
	for _, dir := range addedDirs {
		sb.WriteString(theme.SuccessText(" + "+dir) + "\n")
	}
	for _, dir := range removedDirs {
		sb.WriteString(theme.ErrorText(" - "+dir) + "\n")
	}

	return sb.String()
}

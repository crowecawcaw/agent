package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent/models"
	"agent/theme"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func validateAndResolvePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	return absPath, nil
}

func generateDiff(oldContent, newContent, filePath string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, true)

	var buff strings.Builder

	// Add file header
	buff.WriteString(theme.InfoText(fmt.Sprintf("📄 %s", filePath)) + "\n")
	buff.WriteString("\n───────────────────────────────────────────────────────\n")

	addCount := 0
	delCount := 0

	for diffIndex, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			for i, line := range lines {
				addCount++
				buff.WriteString(theme.SuccessText(line))
				if i < len(lines)-1 {
					buff.WriteString("\n")
				}
			}
		case diffmatchpatch.DiffDelete:
			for i, line := range lines {
				delCount++
				buff.WriteString(theme.ErrorText(line))
				if i < len(lines)-1 {
					buff.WriteString("\n")
				}
			}
		case diffmatchpatch.DiffEqual:
			skipStart := 2
			skipEnd := len(lines) - 3
			if diffIndex == 0 {
				skipStart = 0
			}
			if diffIndex == len(diffs)-1 {
				skipEnd = len(lines)
			}
			if skipStart >= skipEnd {
				_, _ = buff.WriteString(strings.Join(lines, "\n  "))
			} else {
				output := strings.Join(append(append(lines[0:skipStart], "\n"), lines[skipEnd:]...), "\n")
				_, _ = buff.WriteString(output)
			}
		}

		// Add newline between diffs unless it's the last diff
		if diffIndex < len(diffs)-1 {
			buff.WriteString("\n")
		}
	}

	buff.WriteString("\n───────────────────────────────────────────────────────\n")
	buff.WriteString(theme.InfoText(fmt.Sprintf(" +%d -%d lines", addCount, delCount)))

	return buff.String()
}

// NewCreateFileTool creates a create_file tool definition
func NewCreateFileTool() models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to create",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []interface{}{"path", "content"},
	}

	return models.ToolDefinition{
		Name:        "create_file",
		Description: "Create a new file with the specified content. If the file already exists, it will be overwritten.",
		Schema:      schema,
		Func:        createFile,
	}
}

// NewEditFileTool creates an edit_file tool definition
func NewEditFileTool() models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"old_str": map[string]interface{}{
				"type":        "string",
				"description": "The exact string to find and replace. Must match exactly including whitespace and newlines.",
			},
			"new_str": map[string]interface{}{
				"type":        "string",
				"description": "The string to replace old_str with",
			},
		},
		"required": []interface{}{"path", "old_str", "new_str"},
	}

	return models.ToolDefinition{
		Name:        "edit_file",
		Description: "Edit a file by replacing old_str with new_str. The old_str must match exactly including whitespace and newlines. If old_str appears multiple times, only the first occurrence will be replaced.",
		Schema:      schema,
		Func:        editFile,
	}
}

// NewDeleteFileTool creates a delete_file tool definition
func NewDeleteFileTool() models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to delete",
			},
		},
		"required": []interface{}{"path"},
	}

	return models.ToolDefinition{
		Name:        "delete_file",
		Description: "Delete a file from the filesystem",
		Schema:      schema,
		Func:        deleteFile,
	}
}

func createFile(ctx context.Context, params map[string]interface{}) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", "", fmt.Errorf("content must be a string")
	}

	absPath, err := validateAndResolvePath(path)
	if err != nil {
		return "", "", WrapToolError("create_file", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", WrapToolError("create_file", fmt.Errorf("failed to create directory %s: %w", dir, err))
	}

	oldContent := ""
	isUpdate := false
	if existingContent, err := os.ReadFile(absPath); err == nil {
		oldContent = string(existingContent)
		isUpdate = true
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", "", WrapToolError("create_file", fmt.Errorf("failed to write file: %w", err))
	}

	agentMessage := "Created"
	if isUpdate {
		agentMessage = "Updated"
	}

	return generateDiff(oldContent, content, absPath), agentMessage, nil
}

func editFile(ctx context.Context, params map[string]interface{}) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	oldStr, ok := params["old_str"].(string)
	if !ok {
		return "", "", fmt.Errorf("old_str must be a string")
	}

	newStr, ok := params["new_str"].(string)
	if !ok {
		return "", "", fmt.Errorf("new_str must be a string")
	}

	absPath, err := validateAndResolvePath(path)
	if err != nil {
		return "", "", WrapToolError("edit_file", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", "", WrapToolError("edit_file", fmt.Errorf("failed to read file: %w", err))
	}

	oldContent := string(content)

	if !strings.Contains(oldContent, oldStr) {
		return "", "", WrapToolError("edit_file", fmt.Errorf("old_str not found in file"))
	}

	newContent := strings.Replace(oldContent, oldStr, newStr, 1)

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", "", WrapToolError("edit_file", fmt.Errorf("failed to write file: %w", err))
	}

	return generateDiff(oldContent, newContent, absPath), "Updated", nil
}

func deleteFile(ctx context.Context, params map[string]interface{}) (string, string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", "", fmt.Errorf("path must be a string")
	}

	absPath, err := validateAndResolvePath(path)
	if err != nil {
		return "", "", WrapToolError("delete_file", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", "", WrapToolError("delete_file", fmt.Errorf("failed to read file: %w", err))
	}
	oldContent := string(content)

	if err := os.Remove(absPath); err != nil {
		return "", "", WrapToolError("delete_file", fmt.Errorf("failed to delete file: %w", err))
	}

	return generateDiff(oldContent, "", absPath), "Deleted", nil
}

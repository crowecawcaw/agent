package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent/theme"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Shared utilities for file operations

func validateAndResolvePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	return absPath, nil
}

func generateDiff(oldContent, newContent, filePath string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffCleanupSemantic(dmp.DiffMain(oldContent, newContent, true))

	var diff strings.Builder

	diff.WriteString(theme.InfoText(fmt.Sprintf("📄 %s", filePath)) + "\n")
	diff.WriteString(theme.DebugText("───────────────────────────────────────────────────────") + "\n")

	addCount := 0
	delCount := 0

	for diffIndex, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		if len(lines) > 0 && lines[0] == "" {
			lines = lines[1:]
		}
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		switch d.Type {
		case diffmatchpatch.DiffEqual:
			linesToShow := make(map[int]bool)

			if diffIndex > 0 {
				for i := 0; i < 2 && i < len(lines); i++ {
					linesToShow[i] = true
				}
			}

			if diffIndex < len(diffs)-1 {
				for i := len(lines) - 2; i < len(lines); i++ {
					if i >= 0 {
						linesToShow[i] = true
					}
				}
			}

			if len(lines) <= 4 {
				for i := 0; i < len(lines); i++ {
					linesToShow[i] = true
				}
			}

			lastPrinted := -1
			for i := 0; i < len(lines); i++ {
				if linesToShow[i] {
					if lastPrinted >= 0 && i > lastPrinted+1 {
						diff.WriteString("    ...\n")
					}
					diff.WriteString(theme.DebugText("  "+lines[i]) + "\n")
					lastPrinted = i
				}
			}
		case diffmatchpatch.DiffDelete:
			delCount += len(lines)
			for _, line := range lines {
				diff.WriteString(theme.ErrorText("- "+line) + "\n")
			}
		case diffmatchpatch.DiffInsert:
			addCount += len(lines)
			for _, line := range lines {
				diff.WriteString(theme.SuccessText("+ "+line) + "\n")
			}
		}
	}

	diff.WriteString(theme.DebugText("───────────────────────────────────────────────────────") + "\n")
	diff.WriteString(theme.InfoText(fmt.Sprintf("Summary: %d additions, %d deletions", addCount, delCount)) + "\n")

	return diff.String()
}

// CreateFileTool creates new files
type CreateFileTool struct {
	*BaseTool
}

func NewCreateFileTool() *CreateFileTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file to create",
			},
			"contents": map[string]interface{}{
				"type":        "string",
				"description": "Contents of the new file",
			},
		},
		"required": []interface{}{"file_path", "contents"},
	}

	baseTool := NewBaseTool(
		"create_file",
		"Creates a new file with the specified contents. Will fail if the file already exists.",
		schema,
	)

	return &CreateFileTool{
		BaseTool: baseTool,
	}
}

func (t *CreateFileTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	filePath := params["file_path"].(string)
	contents := params["contents"].(string)

	absPath, err := validateAndResolvePath(filePath)
	if err != nil {
		return "", NewToolError(t.Name(), err.Error(), err)
	}

	if _, err := os.Stat(absPath); err == nil {
		return "", NewToolError(t.Name(), fmt.Sprintf("file already exists: %s", absPath), nil)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", NewToolError(t.Name(), fmt.Sprintf("failed to create directory %s", dir), err)
	}

	if err := os.WriteFile(absPath, []byte(contents), 0644); err != nil {
		return "", NewToolError(t.Name(), "failed to create file", err)
	}

	diff := generateDiff("", contents, absPath)
	statusCh <- "\n" + diff

	return "Ok", nil
}

// EditFileTool modifies existing files
type EditFileTool struct {
	*BaseTool
}

func NewEditFileTool() *EditFileTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file to modify",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "Exact text to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The replacement text",
			},
			"expected_replacements": map[string]interface{}{
				"type":        "integer",
				"description": "Number of replacements expected. Defaults to 1 if not specified. Use when you want to replace multiple occurrences.",
				"minimum":     1,
			},
		},
		"required": []interface{}{"file_path", "old_string", "new_string"},
	}

	baseTool := NewBaseTool(
		"edit_file",
		"Modifies an existing file by replacing exact text matches. When changing code, include 3 lines of unchanged code before and after so the tool can locate the correct lines to replace.",
		schema,
	)

	return &EditFileTool{
		BaseTool: baseTool,
	}
}

func (t *EditFileTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	filePathInterface, exists := params["file_path"]
	if !exists || filePathInterface == nil {
		return "", NewToolError(t.Name(), "file_path parameter is required", nil)
	}
	filePath, ok := filePathInterface.(string)
	if !ok {
		return "", NewToolError(t.Name(), "file_path must be a string", nil)
	}

	oldStringInterface, exists := params["old_string"]
	if !exists || oldStringInterface == nil {
		return "", NewToolError(t.Name(), "old_string parameter is required", nil)
	}
	oldString, ok := oldStringInterface.(string)
	if !ok {
		return "", NewToolError(t.Name(), "old_string must be a string", nil)
	}

	newStringInterface, exists := params["new_string"]
	if !exists || newStringInterface == nil {
		return "", NewToolError(t.Name(), "new_string parameter is required", nil)
	}
	newString, ok := newStringInterface.(string)
	if !ok {
		return "", NewToolError(t.Name(), "new_string must be a string", nil)
	}

	expectedReplacements := 1
	if expectedReplInterface, exists := params["expected_replacements"]; exists {
		if expectedReplFloat, ok := expectedReplInterface.(float64); ok {
			expectedReplacements = int(expectedReplFloat)
		} else if expectedReplInt, ok := expectedReplInterface.(int); ok {
			expectedReplacements = expectedReplInt
		}
	}

	absPath, err := validateAndResolvePath(filePath)
	if err != nil {
		return "", NewToolError(t.Name(), err.Error(), err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", NewToolError(t.Name(), fmt.Sprintf("failed to read file %s", absPath), err)
	}

	count := strings.Count(string(content), oldString)
	if count == 0 {
		return "", NewToolError(t.Name(), fmt.Sprintf("could not find text to replace in %s: %q", absPath, oldString), nil)
	}

	if count != expectedReplacements {
		if expectedReplacements == 1 {
			return "", NewToolError(t.Name(), fmt.Sprintf("found %d occurrences of the same text in %s. Add more surrounding context to make the match unique, or set expected_replacements to %d: %q", count, absPath, count, oldString), nil)
		} else {
			return "", NewToolError(t.Name(), fmt.Sprintf("expected %d replacements but found %d occurrences in %s: %q", expectedReplacements, count, absPath, oldString), nil)
		}
	}

	oldContent := string(content)
	var newContent string
	if expectedReplacements == 1 {
		newContent = strings.Replace(oldContent, oldString, newString, 1)
	} else {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", NewToolError(t.Name(), "failed to write file", err)
	}

	diff := generateDiff(oldContent, newContent, absPath)
	statusCh <- "\n" + diff

	return "Ok", nil
}

// DeleteFileTool removes files
type DeleteFileTool struct {
	*BaseTool
}

func NewDeleteFileTool() *DeleteFileTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file to delete",
			},
		},
		"required": []interface{}{"file_path"},
	}

	baseTool := NewBaseTool(
		"delete_file",
		"Deletes an existing file. Will fail if the file does not exist.",
		schema,
	)

	return &DeleteFileTool{
		BaseTool: baseTool,
	}
}

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	filePath := params["file_path"].(string)

	absPath, err := validateAndResolvePath(filePath)
	if err != nil {
		return "", NewToolError(t.Name(), err.Error(), err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", NewToolError(t.Name(), fmt.Sprintf("file does not exist: %s", absPath), err)
	}

	if err := os.Remove(absPath); err != nil {
		return "", NewToolError(t.Name(), "failed to delete file", err)
	}

	diff := generateDiff(string(content), "", absPath)
	statusCh <- "\n" + diff

	return "Ok", nil
}

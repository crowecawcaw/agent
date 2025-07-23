package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create test file
	originalContent := "line 1\nline 2\nline 3"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test parameter validations
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr string
	}{
		{"missing path", map[string]interface{}{"old_str": "old", "new_str": "new"}, "path must be a string"},
		{"invalid path type", map[string]interface{}{"path": 123, "old_str": "old", "new_str": "new"}, "path must be a string"},
		{"missing old_str", map[string]interface{}{"path": testFile, "new_str": "new"}, "old_str must be a string"},
		{"invalid old_str type", map[string]interface{}{"path": testFile, "old_str": 123, "new_str": "new"}, "old_str must be a string"},
		{"missing new_str", map[string]interface{}{"path": testFile, "old_str": "old"}, "new_str must be a string"},
		{"invalid new_str type", map[string]interface{}{"path": testFile, "old_str": "old", "new_str": 123}, "new_str must be a string"},
		{"nonexistent file", map[string]interface{}{"path": "/nonexistent/file.txt", "old_str": "old", "new_str": "new"}, "failed to read file"},
		{"old_str not found", map[string]interface{}{"path": testFile, "old_str": "not found", "new_str": "new"}, "old_str not found in file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := editFile(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}

	// Test successful edit
	params := map[string]interface{}{
		"path":    testFile,
		"old_str": "line 2",
		"new_str": "modified line 2",
	}

	userMsg, agentMsg, err := editFile(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check agent response
	if agentMsg != "Updated" {
		t.Errorf("expected agent message 'Updated', got %q", agentMsg)
	}

	// Check user message contains part of the edit
	if !strings.Contains(userMsg, "modified") || !strings.Contains(userMsg, "line 2") {
		t.Errorf("expected user message to contain edit content, got %q", userMsg)
	}

	// Verify file was actually modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "line 1\nmodified line 2\nline 3"
	if string(content) != expectedContent {
		t.Errorf("expected file content %q, got %q", expectedContent, string(content))
	}
}

func TestCreateFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "new.txt")
	existingFile := filepath.Join(tempDir, "existing.txt")

	// Create existing file for overwrite test
	if err := os.WriteFile(existingFile, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test parameter validations
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr string
	}{
		{"missing path", map[string]interface{}{"content": "test"}, "path must be a string"},
		{"invalid path type", map[string]interface{}{"path": 123, "content": "test"}, "path must be a string"},
		{"missing content", map[string]interface{}{"path": testFile}, "content must be a string"},
		{"invalid content type", map[string]interface{}{"path": testFile, "content": 123}, "content must be a string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := createFile(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}

	// Test successful file creation
	params := map[string]interface{}{
		"path":    testFile,
		"content": "hello world",
	}

	userMsg, agentMsg, err := createFile(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check agent response
	if agentMsg != "Created" {
		t.Errorf("expected agent message 'Created', got %q", agentMsg)
	}

	// Check user message contains creation info
	if !strings.Contains(userMsg, "created") && !strings.Contains(userMsg, "11 characters") && !strings.Contains(userMsg, "new.txt") {
		t.Errorf("expected user message to contain creation info, got %q", userMsg)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected file content %q, got %q", "hello world", string(content))
	}

	// Test file overwrite
	overwriteParams := map[string]interface{}{
		"path":    existingFile,
		"content": "new content",
	}

	userMsg, agentMsg, err = createFile(ctx, overwriteParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check agent response for overwrite
	if agentMsg != "Updated" {
		t.Errorf("expected agent message 'Updated', got %q", agentMsg)
	}

	// Check user message contains diff for overwrite
	if !strings.Contains(userMsg, "new") || !strings.Contains(userMsg, "content") || !strings.Contains(userMsg, "existing.txt") {
		t.Errorf("expected user message to contain diff content, got %q", userMsg)
	}

	// Verify file was overwritten
	content, err = os.ReadFile(existingFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new content" {
		t.Errorf("expected file content %q, got %q", "new content", string(content))
	}
}

func TestDeleteFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "delete_me.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test parameter validations
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr string
	}{
		{"missing path", map[string]interface{}{}, "path must be a string"},
		{"invalid path type", map[string]interface{}{"path": 123}, "path must be a string"},
		{"nonexistent file", map[string]interface{}{"path": "/nonexistent/file.txt"}, "open /nonexistent/file.txt: no such file or directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := deleteFile(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}

	// Test successful file deletion
	params := map[string]interface{}{
		"path": testFile,
	}

	userMsg, agentMsg, err := deleteFile(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check agent response
	if agentMsg != "Deleted" {
		t.Errorf("expected agent message 'Deleted', got %q", agentMsg)
	}

	// Check user message contains deletion info
	if !strings.Contains(userMsg, "deleted") && !strings.Contains(userMsg, "delete_me.txt") {
		t.Errorf("expected user message to contain 'deleted', got %q", userMsg)
	}

	// Verify file was actually deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, but it still exists")
	}
}

func TestGenerateDiff(t *testing.T) {
	// Generate initial content with 50 lines
	initialContentBuilder := strings.Builder{}
	for i := 1; i <= 50; i++ {
		initialContentBuilder.WriteString(fmt.Sprintf("line %d\n", i))
	}
	initialContent := initialContentBuilder.String()

	// Generate new content with line 25 missing
	newContentBuilder := strings.Builder{}
	for i := 1; i <= 50; i++ {
		if i == 25 {
			newContentBuilder.WriteString("bananas\n")
		}
		newContentBuilder.WriteString(fmt.Sprintf("line %d\n", i))
	}
	newContent := newContentBuilder.String()

	tests := []struct {
		name                 string
		oldContent           string
		newContent           string
		expectedDiffParts    []string // Parts expected to be in the diff output
		notExpectedDiffParts []string // Parts not expected to be in the diff output
	}{
		{
			name:       "empty to initial content",
			oldContent: "",
			newContent: initialContent,
			expectedDiffParts: []string{
				"line 1",
				"line 2",
				"line 50",
			},
			notExpectedDiffParts: []string{},
		},
		{
			name:       "initial content to empty",
			oldContent: initialContent,
			newContent: "",
			expectedDiffParts: []string{
				"line 1",
				"line 2",
				"line 50",
			},
			notExpectedDiffParts: []string{},
		},
		{
			name:       "initial content to new content (line 25 missing)",
			oldContent: initialContent,
			newContent: newContent,
			expectedDiffParts: []string{
				"line 24",
				"line 25",
				"bananas",
				"line 26",
			},
			notExpectedDiffParts: []string{
				// Outside the number of lines of unchanged content to be shown
				"line 1",
				"line 50",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := generateDiff(tt.oldContent, tt.newContent, "")

			for _, expected := range tt.expectedDiffParts {
				if !strings.Contains(diff, expected) {
					t.Errorf("expected diff to contain %q, but it did not. Diff:\n%s", expected, diff)
				}
			}
			for _, notExpected := range tt.notExpectedDiffParts {
				if strings.Contains(diff, notExpected) {
					t.Errorf("expected diff to not contain %q, but it did. Diff:\n%s", notExpected, diff)
				}
			}
		})
	}
}

package tools

import (
	"context"
	"strings"
	"testing"
)

func TestShell(t *testing.T) {
	ctx := context.Background()

	// Test parameter validations
	tool := NewShellTool(nil)
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr string
	}{
		{"missing command", map[string]interface{}{}, "command must be a string"},
		{"invalid command type", map[string]interface{}{"command": 123}, "command must be a string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tool.Func(ctx, tt.params)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}

	// Test successful command execution
	params := map[string]interface{}{
		"command": "echo 'hello world'",
	}

	userMsg, agentMsg, err := tool.Func(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// User message should be empty (printed directly via streaming output)
	if userMsg != "" {
		t.Errorf("expected empty user message, got %q", userMsg)
	}

	// Check agent message contains structured info
	if !strings.Contains(agentMsg, "Command: echo 'hello world'") {
		t.Errorf("expected agent message to contain command, got %q", agentMsg)
	}
	if !strings.Contains(agentMsg, "Exit code: 0") {
		t.Errorf("expected agent message to contain exit code 0, got %q", agentMsg)
	}
	if !strings.Contains(agentMsg, "hello world") {
		t.Errorf("expected agent message to contain command output, got %q", agentMsg)
	}
	if !strings.Contains(agentMsg, "Working directory:") {
		t.Errorf("expected agent message to contain working directory, got %q", agentMsg)
	}
	if !strings.Contains(agentMsg, "Duration:") {
		t.Errorf("expected agent message to contain duration, got %q", agentMsg)
	}

	// Test command with non-zero exit code
	failParams := map[string]interface{}{
		"command": "exit 42",
	}

	userMsg, agentMsg, err = tool.Func(ctx, failParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not return error even for failed commands
	if userMsg != "" {
		t.Errorf("expected empty user message, got %q", userMsg)
	}

	// Check agent message contains failure info
	if !strings.Contains(agentMsg, "Command: exit 42") {
		t.Errorf("expected agent message to contain command, got %q", agentMsg)
	}
	if !strings.Contains(agentMsg, "Exit code: 42") {
		t.Errorf("expected agent message to contain exit code 42, got %q", agentMsg)
	}

	// Test command with no output
	noOutputParams := map[string]interface{}{
		"command": "true",
	}

	_, agentMsg, err = tool.Func(ctx, noOutputParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(agentMsg, "Output: (no output)") && !strings.Contains(agentMsg, "Output:\n") {
		t.Errorf("expected agent message to indicate no output, got %q", agentMsg)
	}

	// Test command with stderr output
	stderrParams := map[string]interface{}{
		"command": "echo 'error message' >&2",
	}

	_, agentMsg, err = tool.Func(ctx, stderrParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(agentMsg, "error message") {
		t.Errorf("expected agent message to contain stderr output, got %q", agentMsg)
	}
}

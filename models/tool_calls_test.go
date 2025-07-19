package models

import (
	"testing"
)

func TestMessageWithToolCalls(t *testing.T) {
	// Test that we can create messages with tool calls
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: FunctionCall{
			Name:      "test_tool",
			Arguments: `{"param": "value"}`,
		},
	}

	message := Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: []ToolCall{toolCall},
	}

	if message.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", message.Role)
	}

	if len(message.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(message.ToolCalls))
	}

	if message.ToolCalls[0].ID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", message.ToolCalls[0].ID)
	}

	if message.ToolCalls[0].Function.Name != "test_tool" {
		t.Errorf("Expected function name 'test_tool', got '%s'", message.ToolCalls[0].Function.Name)
	}
}

func TestToolResultMessage(t *testing.T) {
	// Test that we can create tool result messages
	message := Message{
		Role:       "tool",
		Content:    "Tool execution result",
		ToolCallID: "call_123",
	}

	if message.Role != "tool" {
		t.Errorf("Expected role 'tool', got '%s'", message.Role)
	}

	if message.Content != "Tool execution result" {
		t.Errorf("Expected content 'Tool execution result', got '%s'", message.Content)
	}

	if message.ToolCallID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", message.ToolCallID)
	}
}

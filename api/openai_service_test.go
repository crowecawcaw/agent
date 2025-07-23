package api

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go"
)

func TestToolCallMerging(t *testing.T) {
	// Simulate streaming tool call chunks that would cause the original bug
	chunks := []openai.ChatCompletionChunkChoiceDeltaToolCall{
		{
			ID: "call_123",
			Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
				Name:      "update_context",
				Arguments: `{"add_file": {"path"`,
			},
		},
		{
			ID: "call_123",
			Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
				Arguments: `: "/some/file.go"}}`,
			},
		},
	}

	// Simulate the merging logic from the fixed code
	toolCallsMap := make(map[string]*openai.ChatCompletionChunkChoiceDeltaToolCall)

	for _, toolCallChunk := range chunks {
		if toolCallChunk.ID == "" {
			continue
		}

		if existing, exists := toolCallsMap[toolCallChunk.ID]; exists {
			// Merge with existing tool call
			if toolCallChunk.Function.Name != "" {
				existing.Function.Name = toolCallChunk.Function.Name
			}
			if toolCallChunk.Function.Arguments != "" {
				existing.Function.Arguments += toolCallChunk.Function.Arguments
			}
		} else {
			// Create new tool call entry
			toolCallsMap[toolCallChunk.ID] = &openai.ChatCompletionChunkChoiceDeltaToolCall{
				ID: toolCallChunk.ID,
				Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
					Name:      toolCallChunk.Function.Name,
					Arguments: toolCallChunk.Function.Arguments,
				},
			}
		}
	}

	// Verify the merged result
	if len(toolCallsMap) != 1 {
		t.Errorf("Expected 1 merged tool call, got %d", len(toolCallsMap))
	}

	mergedCall := toolCallsMap["call_123"]
	if mergedCall == nil {
		t.Fatal("Expected merged tool call with ID 'call_123'")
	}

	expectedName := "update_context"
	if mergedCall.Function.Name != expectedName {
		t.Errorf("Expected function name %q, got %q", expectedName, mergedCall.Function.Name)
	}

	expectedArgs := `{"add_file": {"path": "/some/file.go"}}`
	if mergedCall.Function.Arguments != expectedArgs {
		t.Errorf("Expected arguments %q, got %q", expectedArgs, mergedCall.Function.Arguments)
	}

	// Test that the merged arguments are valid JSON
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(mergedCall.Function.Arguments), &args); err != nil {
		t.Errorf("Merged arguments should be valid JSON: %v", err)
	}
}

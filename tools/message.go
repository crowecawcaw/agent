package tools

import (
	"context"
	"fmt"
)
type MessageTool struct {
	*BaseTool
	deleteMessageFunc func(role, contentContains string) (bool, error)
}

// ClearHistoryTool provides message history clearing capability
type ClearHistoryTool struct {
	*BaseTool
}

// NewDeleteMessageTool creates a new tool for deleting messages from history
func NewDeleteMessageTool(deleteMessageFunc func(role, contentContains string) (bool, error)) *MessageTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"role": map[string]interface{}{
				"type":        "string",
				"description": "Role of the message to delete",
				"enum":        []interface{}{"user", "assistant", "tool"},
			},
			"content_contains": map[string]interface{}{
				"type":        "string",
				"description": "Unique portion of the message content to match. Provide enough text to uniquely identify the message. If multiple messages match, the first one will be deleted.",
			},
		},
		"required": []interface{}{
			"role", "content_contains",
		},
		"anyOf": []interface{}{
			map[string]interface{}{"required": []interface{}{"content_contains"}},
			map[string]interface{}{"required": []interface{}{"recent_count"}},
		},
	}

	return &MessageTool{
		BaseTool: NewBaseTool(
			"delete_message",
			"Delete a message from the conversation history using role and content matching. Useful for clearing context when messages are no longer useful. Good examples of messages to delete: old build logs, search results that aren't needed, tool calls after their results are applied. Deleting larger messages is more helpful that deleting smaller messages. Bias towards not removing user messages unless they are definitely not needed and large.",
			schema,
		),
		deleteMessageFunc: deleteMessageFunc,
	}
}

// Execute deletes a message from the conversation history
func (t *MessageTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	role, ok := params["role"].(string)
	if !ok {
		return "", fmt.Errorf("role must be one of: user, assistant, tool")
	}

	contentContains, hasContent := params["content_contains"].(string)

	if !hasContent {
		return "", fmt.Errorf("must provide either content_contains")
	}

	if t.deleteMessageFunc == nil {
		return "", fmt.Errorf("message deletion function not set")
	}

	statusCh <- fmt.Sprintf("Searching for %s message containing: %s", role, contentContains)
	deleted, err := t.deleteMessageFunc(role, contentContains)
	if err != nil {
		return "", fmt.Errorf("failed to delete message: %w", err)
	}
	if deleted {
		return fmt.Sprintf("Deleted %s message containing: %s", role, contentContains), nil
	}

	return "Message not found", nil
}

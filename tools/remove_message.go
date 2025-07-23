package tools

import (
	"agent/models"
	"context"
	"fmt"
)

// DeleteMessageFunc is the callback function type for deleting messages
type DeleteMessageFunc func(role, contentContains string) (bool, error)

// NewDeleteMessageTool creates a delete_message tool definition
func NewRemoveMessageTool(deleteMessageFunc DeleteMessageFunc) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"role": map[string]interface{}{
				"type":        "string",
				"description": "Role of the message to delete",
				"enum":        []interface{}{"user", "assistant", "tool"},
			},
			"message_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the message to delete",
			},
		},
		"required": []interface{}{
			"role", "message_id",
		},
		"anyOf": []interface{}{
			map[string]interface{}{"required": []interface{}{"message_id"}},
			map[string]interface{}{"required": []interface{}{"recent_count"}},
		},
	}

	return models.ToolDefinition{
		Name:        "remove_message",
		Description: "Delete a message from the conversation history using role and message ID. Useful for clearing context when messages are no longer useful. Good examples of messages to delete: old build logs, search results that aren't needed, tool calls after their results are applied. Deleting larger messages is more helpful that deleting smaller messages. Bias towards not removing user messages unless they are definitely not needed and large.",
		Schema:      schema,
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return removeMessage(ctx, params, deleteMessageFunc)
		},
	}
}

// removeMessage implements the delete message functionality
func removeMessage(ctx context.Context, params map[string]interface{}, deleteMessageFunc DeleteMessageFunc) (string, string, error) {
	role, ok := params["role"].(string)
	if !ok {
		return "", "", fmt.Errorf("role must be one of: user, assistant, tool")
	}

	messageID, hasMessageID := params["message_id"].(string)

	if !hasMessageID {
		return "", "", fmt.Errorf("must provide message_id")
	}

	if deleteMessageFunc == nil {
		return "", "", fmt.Errorf("message deletion function not set")
	}

	deleted, err := deleteMessageFunc(role, messageID)
	if err != nil {
		return "", "", fmt.Errorf("failed to delete message: %w", err)
	}
	if deleted {
		return fmt.Sprintf("Deleted %s message with ID: %s\n", role, messageID), "Deleted", nil
	}

	return "Message not found\n", "Not found", nil
}

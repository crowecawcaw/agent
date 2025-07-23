package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveMessageTool(t *testing.T) {
	// Mock DeleteMessageFunc
	mockDeleteMessageFunc := func(role, messageID string) (bool, error) {
		// In a real scenario, you might log or store these calls for assertion
		return true, nil
	}

	// Create the tool directly
	tool := NewRemoveMessageTool(mockDeleteMessageFunc)

	// Test case 1: Remove a user message
	params := map[string]interface{}{"message_id": "123", "role": "user"}
	userMsg, agentMsg, err := tool.Func(context.Background(), params)
	assert.NoError(t, err)
	assert.Contains(t, userMsg, "Deleted user message with ID: 123")
	assert.Equal(t, agentMsg, "Deleted")

	// Test case 2: Remove an assistant message
	params = map[string]interface{}{"message_id": "456", "role": "assistant"}
	userMsg, agentMsg, err = tool.Func(context.Background(), params)
	assert.NoError(t, err)
	assert.Contains(t, userMsg, "Deleted assistant message with ID: 456")
	assert.Equal(t, agentMsg, "Deleted")

	// Test case 3: Remove a tool message
	params = map[string]interface{}{"message_id": "789", "role": "tool"}
	userMsg, agentMsg, err = tool.Func(context.Background(), params)
	assert.NoError(t, err)
	assert.Contains(t, userMsg, "Deleted tool message with ID: 789")
	assert.Equal(t, agentMsg, "Deleted")

	// Test case 4: Missing message_id
	params = map[string]interface{}{"role": "user"}
	userMsg, agentMsg, err = tool.Func(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must provide message_id")
	assert.Empty(t, userMsg)
	assert.Empty(t, agentMsg)

	// Test case 5: DeleteMessageFunc returns false (message not found)
	mockDeleteMessageFuncNotFound := func(role, messageID string) (bool, error) {
		return false, nil
	}
	toolNotFound := NewRemoveMessageTool(mockDeleteMessageFuncNotFound)
	params = map[string]interface{}{"message_id": "999", "role": "user"}
	userMsg, agentMsg, err = toolNotFound.Func(context.Background(), params)
	assert.NoError(t, err)
	assert.Contains(t, userMsg, "Message not found")
	assert.Equal(t, agentMsg, "Not found")
}

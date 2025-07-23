package tools

import (
	"agent/models"
	"context"
)

func NewApprovalTool() models.ToolDefinition {
	return models.ToolDefinition{
		Name:        "make_approval_decision",
		Description: "Call this tool to approve or disapprove the command based on the security policy.",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"approved": map[string]interface{}{
					"type":        "boolean",
					"description": "true if the command is approved, false otherwise.",
				},
			},
			"required": []interface{}{"approved"},
		},
		Func: func(ctx context.Context, params map[string]interface{}) (string, string, error) {
			return "", "", nil
		},
	}
}

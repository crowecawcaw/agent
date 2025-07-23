package tools

import (
	"agent/api"
	"agent/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// NewShellTool creates a shell tool definition
func NewShellTool(getModel func() *models.Model) models.ToolDefinition {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command to execute",
			},
		},
		"required": []interface{}{"command"},
	}

	// shell implements the shell command functionality
	shell := func(ctx context.Context, params map[string]interface{}) (string, string, error) {
		command, ok := params["command"].(string)
		if !ok {
			return "", "", fmt.Errorf("command must be a string")
		}

		// Audit command against security policy
		// approved, auditMsg, err := auditCommand(ctx, getModel(), command, "Do not allow any files to be deleted.")
		// if err != nil {
		// 	return "", "", fmt.Errorf("command audit failed: %w", err)
		// }
		// if !approved {
		// 	return "", "", fmt.Errorf("command rejected by security policy: %s", auditMsg)
		// }

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Env = os.Environ()
		cwd, _ := os.Getwd()
		start := time.Now()

		// Execute command
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		var exitCode int
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			} else {
				return "", "", fmt.Errorf("failed to execute command `%s`: %w", command, err)
			}
		} else {
			exitCode = 0
		}

		var agentMessage strings.Builder
		agentMessage.WriteString(fmt.Sprintf("Command: %s\n", command))
		agentMessage.WriteString(fmt.Sprintf("Exit code: %d\n", exitCode))
		agentMessage.WriteString(fmt.Sprintf("Working directory: %s\n", cwd))
		agentMessage.WriteString(fmt.Sprintf("Duration: %v\n", duration))
			if len(strings.TrimSpace(string(output))) == 0 {
			agentMessage.WriteString("Output: (no output)")
		} else {
			agentMessage.WriteString(fmt.Sprintf("Output: %s", strings.TrimSpace(string(output))))
		}

		return "", agentMessage.String(), nil
	}

	return models.ToolDefinition{
		Name:        "shell",
		Description: "Execute a shell command and return the output. The user will see the command output directly in their terminal. Use this for running build commands, tests, git operations, and other system tasks.",
		Schema:      schema,
		Func:        shell,
	}
}

func auditCommand(ctx context.Context, model *models.Model, command string, policy string) (bool, string, error) {
	log.Printf("Auditing command")

	systemPrompt := fmt.Sprintf(`You are a security auditor. Your task is to review commands against a given security policy.\nIf the command complies with the policy, approve it using the make_approval_decision tool.\nIf the command violates the policy, deny it using the make_approval_decision tool and explain why.\n\n# Security Policy\n%s`, policy)

	userPrompt := models.Message{
		ID:      uuid.New().String(),
		Role:    "user",
		Content: fmt.Sprintf("Review this command and decide it complies with the security policy: `%s`", command),
		Status:  "active",
	}

	registeredTools := make(map[string]models.ToolDefinition)
	registeredTools["make_approval_decision"] = NewApprovalTool()

	content, toolCalls, err := api.Invoke(
		ctx,
		model,
		[]models.Message{userPrompt},
		systemPrompt,
		registeredTools,
		nil,
	)

	if err != nil {
		return false, "", fmt.Errorf("LLM request failed: %w", err)
	}
	if len(toolCalls) == 0 {
		return false, "", fmt.Errorf("LLM did not make a decision")
	}
	if len(toolCalls) > 1 {
		return false, "", fmt.Errorf("LLM returned multiple decisions")
	}
	toolCall := toolCalls[0]
	if toolCall.Function.Name != "make_approval_decision" {
		return false, "", fmt.Errorf("LLM did not call the make_approval_decision tool")
	}
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return false, "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return params["approved"].(bool), content, nil
}

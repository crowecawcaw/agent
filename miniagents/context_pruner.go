package miniagents

import (
	"agent/api"
	"agent/models"
	"agent/tools"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

//go:embed context_pruner_prompt.md
var systemPromptTemplate string

// PruneContext runs the context pruning process
func PruneContext(ctx context.Context, model *models.Model, messages *[]models.Message, liveContext tools.LiveContextManager, allTools map[string]models.ToolDefinition) error {

	log.Printf("Starting context pruning")

	prunerTools := make(map[string]models.ToolDefinition)
	prunerTools["remove_message"] = allTools["remove_message"]
	prunerTools["stop_reading_file"] = allTools["stop_reading_file"]
	prunerTools["stop_reading_directory"] = allTools["stop_reading_directory"]

	iteration := 0
	maxIterations := 1

	for iteration < maxIterations {
		iteration++
		log.Printf("Context pruning iteration %d/%d", iteration, maxIterations)

		// Build system prompt with current metrics for this iteration
		systemPrompt := buildSystemPrompt(*messages, liveContext)

		userPrompt := models.Message{
			ID:      uuid.New().String(),
			Role:    "user",
			Content: "Look over the messages and files. Use the tools to reduce the context size.",
			Status:  "active",
		}

		// Make LLM request
		content, toolCalls, err := api.Invoke(
			ctx,
			model,
			[]models.Message{userPrompt},
			systemPrompt,
			prunerTools, // Use tools directly
			nil,         // onReceiveContent - not needed
		)

		if err != nil {
			log.Printf("Context pruning LLM request failed: %v", err)
			return fmt.Errorf("LLM request failed: %w", err)
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			log.Printf("Context pruning completed after %d iterations. Final response: %s", iteration, content)
			break
		}

		// Execute tool calls and update state
		for _, toolCall := range toolCalls {
			tool, exists := prunerTools[toolCall.Function.Name]
			if !exists {
				log.Printf("Tool call skipped: %s not a valid tool", toolCall.Function.Name)
				continue
			}
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
				log.Printf("Tool call failed: %s - %v", toolCall.Function.Name, err)
				continue // Skip to next tool call
			}
			_, agentMessage, err := tool.Func(ctx, params)
			if err != nil {
				log.Printf("Tool call failed: %s - %v", toolCall.Function.Name, err)
				continue // Skip to next tool call
			}
			log.Printf("Tool call succeeded: %s - %s", toolCall.Function.Name, agentMessage)
		}
	}

	if iteration >= maxIterations {
		log.Printf("Context pruning stopped after reaching max iterations (%d)", maxIterations)
	}

	return nil
}

// buildSystemPrompt creates the system prompt with current context metrics
func buildSystemPrompt(messages []models.Message, liveContext tools.LiveContextManager) string {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("- ID: %s, Role: %s, Size: %d chars, Content: %s\n", msg.ID, msg.Role, len(msg.Content), msg.Content))
	}

	prompt := systemPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{MESSAGES}", sb.String())
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_FILE_LIST}", strings.Join(liveContext.ListFiles(), "\n"))
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_DIRECTORY_LIST}", strings.Join(liveContext.ListDirectories(), "\n"))
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_FILES}", liveContext.SerializeFiles())
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_DIRECTORIES}", liveContext.SerializeDirectories())
	return prompt
}

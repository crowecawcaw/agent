package main

import (
	"agent/models"
	"agent/theme"
	"agent/tools"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

//go:embed system_prompt_template.md
var systemPromptTemplate string

// Message extends the model Message with additional fields for internal use
type Message struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	ToolName  string            `json:"tool_name,omitempty"`
	ToolUseID string            `json:"tool_use_id,omitempty"`
	ToolCalls []models.ToolCall `json:"tool_calls,omitempty"` // For assistant messages with tool calls
}

// ConvertToModelMessages converts internal messages to model messages
func ConvertToModelMessages(messages []Message) []models.Message {
	modelMessages := make([]models.Message, len(messages))
	for i, msg := range messages {
		modelMessages[i] = models.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  msg.ToolCalls,
			ToolCallID: msg.ToolUseID, // Map ToolUseID to ToolCallID
		}
	}
	return modelMessages
}

type ToolResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

type DebugInfo struct {
	RecentRequest  interface{}   `json:"recent_request,omitempty"`
	RecentResponse interface{}   `json:"recent_response,omitempty"`
	RecentChunks   []interface{} `json:"recent_chunks,omitempty"`
}

type AgentConfig struct {
	SystemPrompt string
	tools        map[string]tools.Tool
	Messages     []Message
	LiveContext  *LiveContext
	Debug        *DebugInfo
}

type LLMResponse struct {
	TextContent string
	ToolCalls   []models.ToolCall
	HasToolCall bool
}

func NewAgentConfig() *AgentConfig {
	liveContext := NewLiveContext()

	config := &AgentConfig{
		tools:       make(map[string]tools.Tool),
		Messages:    make([]Message, 0),
		LiveContext: liveContext,
		Debug:       &DebugInfo{},
	}

	// Add tools directly
	if liveContext != nil {
		config.tools["update_context"] = tools.NewUpdateContextTool(liveContext)
	}
	config.tools["create_file"] = tools.NewCreateFileTool()
	config.tools["edit_file"] = tools.NewEditFileTool()
	config.tools["delete_file"] = tools.NewDeleteFileTool()
	config.tools["shell"] = tools.NewShellTool()
	config.tools["delete_message"] = tools.NewDeleteMessageTool(config.DeleteMessage)

	// Initialize with default context
	config.InitializeDefaultContext()

	return config
}

func (a *AgentConfig) AddUserMessage(content string) {
	message := Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	}
	a.Messages = append(a.Messages, message)
}

func (a *AgentConfig) AddAgentMessage(content string) {
	message := Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	}
	a.Messages = append(a.Messages, message)
}

func (a *AgentConfig) AddAgentMessageWithToolCalls(content string, toolCalls []models.ToolCall) {
	message := Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}
	a.Messages = append(a.Messages, message)
}

// ConsolidateUpdateContextCalls combines multiple update_context tool calls into a single call
func (a *AgentConfig) ConsolidateUpdateContextCalls(toolCalls []models.ToolCall) []models.ToolCall {
	if len(toolCalls) <= 1 {
		return toolCalls
	}

	var updateContextCalls []models.ToolCall
	var otherCalls []models.ToolCall

	// Separate update_context calls from other tool calls
	for _, call := range toolCalls {
		if call.Function.Name == "update_context" {
			updateContextCalls = append(updateContextCalls, call)
		} else {
			otherCalls = append(otherCalls, call)
		}
	}

	if IsGlobalDebugEnabled() {
		log.Printf("ConsolidateUpdateContextCalls: found %d update_context calls, %d other calls", len(updateContextCalls), len(otherCalls))
	}

	// If we have multiple update_context calls, consolidate them
	if len(updateContextCalls) > 1 {
		if IsGlobalDebugEnabled() {
			log.Printf("Consolidating %d update_context calls into 1 call", len(updateContextCalls))
		}
		consolidatedCall := a.mergeUpdateContextCalls(updateContextCalls)
		// Add the consolidated call plus any other tool calls
		result := []models.ToolCall{consolidatedCall}
		result = append(result, otherCalls...)
		return result
	}

	// No consolidation needed
	return toolCalls
}

// mergeUpdateContextCalls combines multiple update_context calls into a single call
func (a *AgentConfig) mergeUpdateContextCalls(calls []models.ToolCall) models.ToolCall {
	if len(calls) == 0 {
		return models.ToolCall{}
	}

	// Use the ID from the first call
	firstCall := calls[0]

	if IsGlobalDebugEnabled() {
		log.Printf("Merging %d update_context calls:", len(calls))
		for i, call := range calls {
			log.Printf("  Call %d: ID=%s, Args=%s", i+1, call.ID, call.Function.Arguments)
		}
	}

	// Parse arguments from each call and merge them
	var addFiles []interface{}
	var removeFiles []interface{}
	var addDirs []interface{}
	var removeDirs []interface{}

	for _, call := range calls {
		// Parse the JSON arguments
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			if IsGlobalDebugEnabled() {
				log.Printf("Failed to parse arguments for call %s: %v", call.ID, err)
			}
			continue
		}

		// Collect parameters
		if addFile, ok := args["add_file"]; ok {
			addFiles = append(addFiles, addFile)
		}
		if removeFile, ok := args["remove_file"]; ok {
			removeFiles = append(removeFiles, removeFile)
		}
		if addDir, ok := args["add_directory"]; ok {
			addDirs = append(addDirs, addDir)
		}
		if removeDir, ok := args["remove_directory"]; ok {
			removeDirs = append(removeDirs, removeDir)
		}
	}

	// Build merged parameters
	mergedParams := make(map[string]interface{})
	if len(addFiles) > 0 {
		if len(addFiles) == 1 {
			mergedParams["add_file"] = addFiles[0]
		} else {
			mergedParams["add_file"] = addFiles
		}
	}
	if len(removeFiles) > 0 {
		if len(removeFiles) == 1 {
			mergedParams["remove_file"] = removeFiles[0]
		} else {
			mergedParams["remove_file"] = removeFiles
		}
	}
	if len(addDirs) > 0 {
		if len(addDirs) == 1 {
			mergedParams["add_directory"] = addDirs[0]
		} else {
			mergedParams["add_directory"] = addDirs
		}
	}
	if len(removeDirs) > 0 {
		if len(removeDirs) == 1 {
			mergedParams["remove_directory"] = removeDirs[0]
		} else {
			mergedParams["remove_directory"] = removeDirs
		}
	}

	// Convert back to JSON string
	argsBytes, err := json.Marshal(mergedParams)
	if err != nil {
		if IsGlobalDebugEnabled() {
			log.Printf("Failed to marshal merged parameters: %v", err)
		}
		return firstCall // Return original if merge fails
	}

	if IsGlobalDebugEnabled() {
		log.Printf("Merged parameters: %s", string(argsBytes))
	}

	// Create the consolidated call
	return models.ToolCall{
		ID:   firstCall.ID,
		Type: "function",
		Function: models.FunctionCall{
			Name:      "update_context",
			Arguments: string(argsBytes),
		},
	}
}

func (a *AgentConfig) GetHistory() []Message {
	return a.Messages
}

func (a *AgentConfig) DeleteMessage(role, contentContains string) (bool, error) {
	for i, msg := range a.Messages {
		if msg.Role == role && strings.Contains(msg.Content, contentContains) {
			// Delete the message by slicing it out
			a.Messages = append(a.Messages[:i], a.Messages[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (a *AgentConfig) ClearHistory() {
	a.Messages = make([]Message, 0)
}

func (a *AgentConfig) AddToolResultsMessage(toolResults []ToolResult) {
	// Add individual tool messages for each result
	for _, result := range toolResults {
		message := Message{
			Role:      "tool",
			Content:   result.Content,
			Timestamp: time.Now(),
			ToolName:  result.Name,
			ToolUseID: result.ID,
		}
		a.Messages = append(a.Messages, message)
	}
}

func (a *AgentConfig) ProcessLLMResponse(response string, toolCalls []models.ToolCall) (*LLMResponse, error) {
	result := &LLMResponse{
		TextContent: response,
		ToolCalls:   toolCalls,
		HasToolCall: len(toolCalls) > 0,
	}

	result.TextContent = strings.TrimSpace(result.TextContent)

	return result, nil
}

func (a *AgentConfig) BuildSystemPrompt() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	// Build tool descriptions
	var toolDescriptions strings.Builder
	for _, tool := range a.tools {
		toolDescriptions.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", tool.Name(), tool.Description()))
	}

	// Get context usage
	currentSize, maxSize, usagePercent := a.LiveContext.GetContextUsage()
	contextUsage := fmt.Sprintf("Context Usage: %d/%d bytes (%.1f%%)", currentSize, maxSize, usagePercent)

	// Replace template placeholders
	prompt := strings.ReplaceAll(systemPromptTemplate, "{ENV_OS}", runtime.GOOS)
	prompt = strings.ReplaceAll(prompt, "{ENV_CWD}", cwd)
	prompt = strings.ReplaceAll(prompt, "{TOOL_DESCRIPTIONS}", toolDescriptions.String())
	prompt = strings.ReplaceAll(prompt, "{CONTEXT_USAGE}", contextUsage)
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_FILES}", a.LiveContext.SerializeFiles())
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_DIRECTORIES}", a.LiveContext.SerializeDirectories())

	return prompt
}

func (a *AgentConfig) ExecuteToolCall(ctx context.Context, toolCall models.ToolCall) (string, error) {
	tool, exists := a.tools[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", toolCall.Function.Name)
	}

	// Parse the JSON arguments
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// Create a buffered channel for status updates
	statusCh := make(chan string, 10)

	// Start a goroutine to handle status updates
	done := make(chan struct{})
	go func() {
		defer close(done)
		for status := range statusCh {
			fmt.Printf("%s", status)
		}
	}()

	// Execute the tool
	result, err := tool.Execute(ctx, params, statusCh)

	// Close the status channel and wait for the goroutine to finish
	close(statusCh)
	<-done

	return result, err
}

// ProcessWithDirectService handles the complete conversation flow with tool calling
func (a *AgentConfig) ProcessWithDirectService(ctx context.Context, model *models.Model, userInput string) error {
	// Add user message
	a.AddUserMessage(userInput)

	maxIterations := -1
	maxConsecutiveFailures := 3
	consecutiveFailures := 0

	for iteration := 0; maxIterations == -1 || iteration < maxIterations; iteration++ {
		// Get system prompt with current live context
		systemPrompt := a.BuildSystemPrompt()

		// Convert messages to model format
		modelMessages := ConvertToModelMessages(a.GetHistory())

		// Create callback for streaming content
		renderer := theme.NewMarkdownRenderer()
		onReceiveContent := func(token string) {
			_, _ = renderer.Write([]byte(token))
		}

		// Create callback for capturing chunks for debug
		onChunkReceived := func(chunk interface{}) {
			if a.Debug != nil {
				a.Debug.RecentChunks = append(a.Debug.RecentChunks, chunk)
			}
		}

		// Store request info for debug
		if a.Debug != nil {
			a.Debug.RecentRequest = map[string]interface{}{
				"model":         model.Name,
				"provider":      model.Provider.Name,
				"messages":      modelMessages,
				"system_prompt": systemPrompt,
				"tools":         getToolNames(a.GetTools()),
				"timestamp":     time.Now(),
			}
			// Reset chunks for new request
			a.Debug.RecentChunks = make([]interface{}, 0)
		}

		renderer.Flush()

		// Use the package-level models.Request function
		content, toolCalls, err := models.Request(
			ctx,
			model,
			modelMessages,
			systemPrompt,
			a.GetTools(),
			onReceiveContent,
			nil, // onReceiveToolCall - not needed here
			onChunkReceived,
		)

		// Store response info for debug
		if a.Debug != nil {
			a.Debug.RecentResponse = map[string]interface{}{
				"content":    content,
				"tool_calls": toolCalls,
				"error":      err,
				"timestamp":  time.Now(),
			}
		}

		if err != nil {
			if err == context.Canceled {
				fmt.Println("Cancelled by user")
				return nil
			}

			return fmt.Errorf("AI response error: %w", err)
		}

		// If there are tool calls, handle them
		if len(toolCalls) > 0 {
			// Consolidate update_context calls if needed
			consolidatedToolCalls := a.ConsolidateUpdateContextCalls(toolCalls)

			// Add assistant message with tool calls
			a.AddAgentMessageWithToolCalls(content, consolidatedToolCalls)

			// Execute tool calls
			var toolResults []ToolResult
			allToolsSucceeded := true

			for _, toolCall := range consolidatedToolCalls {
				result, err := a.ExecuteToolCall(ctx, toolCall)
				if err != nil {
					// Track consecutive failures
					consecutiveFailures++

					toolResults = append(toolResults, ToolResult{
						ID:      toolCall.ID,
						Name:    toolCall.Function.Name,
						Content: fmt.Sprintf("Tool execution failed: %v", err),
						IsError: true,
					})
					allToolsSucceeded = false

					// If we've hit max consecutive failures, we'll still add the tool results
					// but then return an error after adding them to avoid orphaned tool calls
					if consecutiveFailures >= maxConsecutiveFailures {
						// Add tool results first to satisfy API requirements
						a.AddToolResultsMessage(toolResults)
						return fmt.Errorf("tool execution failed after %d consecutive attempts: %w", maxConsecutiveFailures, err)
					}
				} else {
					// Reset consecutive failures counter on success
					consecutiveFailures = 0
					toolResults = append(toolResults, ToolResult{
						ID:      toolCall.ID,
						Name:    toolCall.Function.Name,
						Content: result,
						IsError: false,
					})
				}
			}

			// Add tool results as a user message
			a.AddToolResultsMessage(toolResults)

			// If any tool failed, add error feedback and continue
			if !allToolsSucceeded {
				a.AddUserMessage("Please fix the error and try again.")
				continue
			}

			// Continue the loop for next iteration
			continue
		} else {
			// No tool calls - add assistant message and we're done
			a.AddAgentMessage(content)
			fmt.Println() // Add newline after response
			return nil
		}
	}

	// If we reach here, we hit max iterations
	finalMsg := fmt.Sprintf("Reached maximum tool call iterations (%d). Processing stopped.", maxIterations)
	a.AddAgentMessage(finalMsg)
	return fmt.Errorf("reached maximum iterations")
}

func (a *AgentConfig) GetTools() map[string]tools.Tool {
	return a.tools
}

// getToolNames returns a slice of tool names for debug purposes
func getToolNames(tools map[string]tools.Tool) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	return names
}

// InitializeDefaultContext sets up the default context with current directory and README.md
func (a *AgentConfig) InitializeDefaultContext() {
	if a.LiveContext == nil {
		return
	}

	// Add current directory
	_ = a.LiveContext.AddDirectory(".", true)

	// Check if README.md exists and add it if it does
	if _, err := os.Stat("README.md"); err == nil {
		_ = a.LiveContext.AddFile("README.md", 1, nil)
	}
}

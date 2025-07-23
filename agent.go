package main

import (
	"agent/api"
	"agent/models"
	"agent/theme"
	"agent/tools"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

//go:embed system_prompt_template.md
var systemPromptTemplate string

type Agent struct {
	mu          sync.RWMutex
	tools       map[string]models.ToolDefinition
	Messages    []models.Message
	LiveContext *LiveContext

	commands        map[string]Command
	config          *Config
	currentModel    *models.Model
	cancelFunc      context.CancelFunc
	inProgress      bool
	inProgressMutex sync.Mutex
	sessionLogger   *SessionLogger
}

func NewAgent() *Agent {
	agent := &Agent{
		Messages:      make([]models.Message, 0),
		LiveContext:   NewLiveContext(),
		sessionLogger: NewSessionLogger(),

		config: LoadConfig(),
	}

	if agent.config.Model != nil {
		err := agent.switchProvider(agent.config.Model.Provider, agent.config.Model.Model)
		if err != nil {
			panic(err)
		}
	}
	agent.registerBuiltinCommands()
	agent.registerTools()
	agent.InitializeDefaultContext()

	return agent
}

func (a *Agent) registerTools() {
	getModel := func() *models.Model {
		return a.currentModel
	}

	a.tools = make(map[string]models.ToolDefinition)
	a.tools["create_file"] = tools.NewCreateFileTool()
	a.tools["edit_file"] = tools.NewEditFileTool()
	a.tools["delete_file"] = tools.NewDeleteFileTool()
	a.tools["shell"] = tools.NewShellTool(getModel)
	a.tools["read_file"] = tools.NewReadFileTool(a.LiveContext)
	a.tools["stop_reading_file"] = tools.NewStopReadingFileTool(a.LiveContext)
	a.tools["read_directory"] = tools.NewReadDirectoryTool(a.LiveContext)
	a.tools["stop_reading_directory"] = tools.NewStopReadingDirectoryTool(a.LiveContext)
	a.tools["remove_message"] = tools.NewRemoveMessageTool(a.DeleteMessage)

}

func (a *Agent) ProcessMessage(input string) {
	// Set in-progress flag
	a.inProgressMutex.Lock()
	a.inProgress = true
	// Create a new cancellable context
	ctx, cancelFunc := context.WithCancel(context.Background())
	a.cancelFunc = cancelFunc
	a.inProgressMutex.Unlock()

	// Ensure we clear the in-progress flag when done
	defer func() {
		a.inProgressMutex.Lock()
		a.inProgress = false
		a.inProgressMutex.Unlock()
	}()

	// Use the simplified agent processing
	err := a.ProcesssMessageWithCancellation(ctx, a.currentModel, input)
	if err != nil {
		fmt.Println("")
		if errors.Is(err, context.Canceled) {
			fmt.Println(theme.WarningText("Cancelled request"))
		} else {
			fmt.Println(theme.WarningText(fmt.Sprintf("Operation failed: %v", err)))
		}
	}
}

func (a *Agent) Close() error {
	return a.sessionLogger.Close()
}

func (a *Agent) switchProvider(providerId string, modelId string) error {
	var model *models.Model
	for _, Provider := range a.config.Providers {
		for _, Model := range Provider.Models {
			if providerId == Provider.ID && modelId == Model.ID {
				model = Model
				model.Provider = Provider
				if strings.HasPrefix(model.Provider.APIKey, "env:") {
					envVar := strings.TrimPrefix(model.Provider.APIKey, "env:")
					model.Provider.APIKey = os.Getenv(envVar)
				}
			}
		}
	}

	if model == nil {
		return fmt.Errorf("model %s not found in registry", modelId)
	}

	// Update chatbot state
	a.currentModel = model

	// Update persistent configuration
	a.config.Model = &SelectedModel{
		Provider: providerId,
		Model:    modelId,
	}

	// Save the updated configuration
	if err := SaveConfig(a.config); err != nil {
		log.Printf("Failed to save config after model switch: %v", err)
	}

	return nil
}

func (a *Agent) AddUserMessage(content string) {
	message := models.Message{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
		Status:    "active",
	}

	a.mu.Lock()
	a.Messages = append(a.Messages, message)
	a.mu.Unlock()

	a.sessionLogger.LogMessage(message)
}

func (a *Agent) AddAgentMessage(content string) {
	message := models.Message{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
		Status:    "active",
	}

	a.mu.Lock()
	a.Messages = append(a.Messages, message)
	a.mu.Unlock()

	a.sessionLogger.LogMessage(message)
}

func (a *Agent) AddAgentMessageWithToolCalls(content string, toolCalls []models.ToolCall) {
	message := models.Message{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
		Status:    "active",
	}

	a.mu.Lock()
	a.Messages = append(a.Messages, message)
	a.mu.Unlock()

	a.sessionLogger.LogMessage(message)
}

func (a *Agent) GetHistory() []models.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]models.Message, len(a.Messages))
	copy(history, a.Messages)
	return history
}

func (a *Agent) DeleteMessage(role, contentContains string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, msg := range a.Messages {
		if msg.Role == role && strings.Contains(msg.Content, contentContains) && msg.Status == "active" {

			deletedMsg := msg
			deletedMsg.ID = uuid.New().String()
			deletedMsg.Timestamp = time.Now()
			deletedMsg.Status = "deleted"

			a.sessionLogger.LogMessage(deletedMsg)

			a.Messages[i].Status = "deleted"
			return true, nil
		}
	}
	return false, nil
}

func (a *Agent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Messages = make([]models.Message, 0)
}

func (a *Agent) AddToolResultsMessage(toolResults []models.ToolResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, result := range toolResults {
		message := models.Message{
			ID:         uuid.New().String(),
			Role:       "tool",
			Content:    result.Content,
			Timestamp:  time.Now(),
			ToolName:   result.Name,
			ToolCallID: result.ID,
			Status:     "active",
		}
		a.Messages = append(a.Messages, message)
		a.sessionLogger.LogMessage(message)
	}
}

func (a *Agent) BuildSystemPrompt() string {

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	currentSize, maxSize, usagePercent := a.LiveContext.GetContextUsage()
	contextUsage := fmt.Sprintf("Context Usage: %d/%d bytes (%.1f%%)", currentSize, maxSize, usagePercent)

	prompt := strings.ReplaceAll(systemPromptTemplate, "{ENV_OS}", runtime.GOOS)
	prompt = strings.ReplaceAll(prompt, "{ENV_CWD}", cwd)
	prompt = strings.ReplaceAll(prompt, "{CONTEXT_USAGE}", contextUsage)
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_FILES}", a.LiveContext.SerializeFiles())
	prompt = strings.ReplaceAll(prompt, "{LIVE_CONTEXT_DIRECTORIES}", a.LiveContext.SerializeDirectories())

	return prompt
}

func (a *Agent) ExecuteToolCall(ctx context.Context, toolCall models.ToolCall) (string, error) {
	tool, exists := a.tools[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", toolCall.Function.Name)
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	userMessage, agentMessage, err := tool.Func(ctx, params)

	if userMessage != "" {
		fmt.Print(lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("2")). // Green
			PaddingLeft(2))
	}

	return agentMessage, err
}

// ProcesssMessageWithCancellation handles the complete conversation flow with tool calling
func (a *Agent) ProcesssMessageWithCancellation(ctx context.Context, model *models.Model, userInput string) error {
	a.AddUserMessage(userInput)

	maxIterations := -1
	maxConsecutiveFailures := 3
	consecutiveFailures := 0

	for iteration := 0; maxIterations == -1 || iteration < maxIterations; iteration++ {
		systemPrompt := a.BuildSystemPrompt()

		modelMessages := (a.GetHistory())

		renderer := theme.NewMarkdownRenderer()
		onReceiveContent := func(token string) {
			renderer.Write([]byte(token))
		}

		fmt.Print("🦜 ")
		renderer.Flush()

		content, toolCalls, err := api.Invoke(
			ctx,
			model,
			modelMessages,
			systemPrompt,
			a.GetTools(),
			onReceiveContent,
		)

		if err != nil {
			if err == context.Canceled {
				fmt.Println("Cancelled by user")
				return nil
			}

			return fmt.Errorf("AI response error: %w", err)
		}

		if len(toolCalls) > 0 {
			a.AddAgentMessageWithToolCalls(content, toolCalls)

			var toolResults []models.ToolResult

			for _, toolCall := range toolCalls {
				result, err := a.ExecuteToolCall(ctx, toolCall)
				if err != nil {
					consecutiveFailures++

					toolResults = append(toolResults, models.ToolResult{
						ID:      toolCall.ID,
						Name:    toolCall.Function.Name,
						Content: fmt.Sprintf("Tool execution failed: %v", err),
						IsError: true,
					})

					if consecutiveFailures >= maxConsecutiveFailures {
						a.AddToolResultsMessage(toolResults)
						return fmt.Errorf("tool execution failed after %d consecutive attempts: %w", maxConsecutiveFailures, err)
					}
				} else {
					consecutiveFailures = 0
					toolResults = append(toolResults, models.ToolResult{
						ID:      toolCall.ID,
						Name:    toolCall.Function.Name,
						Content: result,
						IsError: false,
					})
				}
			}

			a.AddToolResultsMessage(toolResults)
			continue
		} else {
			a.AddAgentMessage(content)
			fmt.Println()
			return nil
		}
	}

	finalMsg := fmt.Sprintf("Reached maximum tool call iterations (%d). Processing stopped.", maxIterations)
	a.AddAgentMessage(finalMsg)
	return fmt.Errorf("reached maximum iterations")
}

func (a *Agent) GetTools() map[string]models.ToolDefinition {
	return a.tools
}

// GetContextCharacterCount calculates the total character count of the context
func (a *Agent) GetContextCharacterCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	totalChars := 0

	for _, msg := range a.Messages {
		if msg.Status == "active" {
			totalChars += len(msg.Content)
		}
	}

	if a.LiveContext != nil {
		totalChars += len(a.LiveContext.SerializeFiles())
		totalChars += len(a.LiveContext.SerializeDirectories())
	}

	return totalChars
}

// InitializeDefaultContext sets up the default context with current directory and README.md
func (a *Agent) InitializeDefaultContext() {
	if a.LiveContext == nil {
		return
	}

	_ = a.LiveContext.AddDirectory(".", true)

	if _, err := os.Stat("README.md"); err == nil {
		_ = a.LiveContext.AddFile("README.md", 1, nil)
	}
}

// SessionLogger logs messages to a session-specific JSONL file.
type SessionLogger struct {
	logFile *os.File
	encoder *json.Encoder
}

// NewSessionLogger creates a new SessionLogger for a given session.
// It creates a new log file named with a timestamp in ~/.agent/sessions/.
func NewSessionLogger() *SessionLogger {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get user home directory: %v", err)
	}

	sessionDir := filepath.Join(homeDir, ".agent", "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		log.Fatalf("failed to create session directory: %v", err)
	}

	timestamp := time.Now().Format("20060102150405")
	logFileName := filepath.Join(sessionDir, fmt.Sprintf("%s.jsonl", timestamp))

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	return &SessionLogger{
		logFile: logFile,
		encoder: json.NewEncoder(logFile),
	}
}

// LogMessage logs a single message to the session log file.
func (sl *SessionLogger) LogMessage(message models.Message) {
	if err := sl.encoder.Encode(message); err != nil {
		fmt.Printf("Error encoding message to log file: %v\n", err)
	}
}

// Close closes the session log file.
func (sl *SessionLogger) Close() error {
	return sl.logFile.Close()
}

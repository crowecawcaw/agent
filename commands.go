package main

import (
	"agent/theme"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExecuteCommand processes a command input and executes the corresponding handler
func (c *Chatbot) ExecuteCommand(input string) {
	parts := strings.Fields(strings.TrimPrefix(input, "/"))
	if len(parts) == 0 {
		return
	}

	commandName := parts[0]
	args := parts[1:]

	handler, exists := c.commands[commandName]
	if !exists {
		fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Unknown command: /%s", commandName)))
		return
	}

	handler(c, args)
}

// GetAvailableCommands returns a string listing all available commands
func (c *Chatbot) GetAvailableCommands() string {
	var commandNames []string
	for name := range c.commands {
		commandNames = append(commandNames, "/"+name)
	}

	return "Commands: " + strings.Join(commandNames, ", ")
}

type Command struct {
	Handler func(*Chatbot, []string)
	Desc    string
}

var builtinCommands = map[string]Command{
	"help":     {handleHelp, "Show available commands and their descriptions"},
	"history":  {handleHistory, "Show message history"},
	"model":    {handleModel, "Show or change the AI model and provider"},
	"provider": {handleProvider, "Show current AI provider"},
	"context":  {handleContext, "Show live context summary (use 'full' to see complete content)"},
	"debug":    {handleDebug, "Toggle debug mode or create debug directory (use 'on', 'off', or no args for directory)"},
	"config":   {handleConfig, "Show current configuration"},
	"clear":    {handleClear, "Clear conversation context"},
	"quit":     {handleQuit, "Exit the chatbot"},
}

// registerBuiltinCommands sets up all the built-in commands
func (c *Chatbot) registerBuiltinCommands() {
	for name, cmd := range builtinCommands {
		c.commands[name] = cmd.Handler
		c.commandDescs[name] = cmd.Desc
	}
}

// Command handlers
func handleHelp(c *Chatbot, args []string) {
	fmt.Println(theme.IndentedInfoText("Available Commands:"))

	for name, desc := range c.commandDescs {
		fmt.Printf("%s - %s\n",
			theme.IndentedSuccessText(fmt.Sprintf("/%s", name)),
			theme.IndentText(desc))
	}

	fmt.Println()
	fmt.Println(theme.IndentedInfoText("Usage examples:"))
	fmt.Println(theme.IndentText("/model anthropic.claude-3-haiku-20240307-v1:0"))
	fmt.Println(theme.IndentText("/context full"))
	fmt.Println(theme.IndentText("/debug on    # Enable debug mode"))
	fmt.Println(theme.IndentText("/debug off   # Disable debug mode"))
	fmt.Println(theme.IndentText("/debug       # Create debug directory"))
	fmt.Println()
	fmt.Println(theme.IndentedInfoText("Note: Press Ctrl+C to cancel ongoing requests or quit at prompt"))
}

func handleProvider(c *Chatbot, args []string) {
	fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Current provider: %s", c.currentModel.Provider.Name)))
	fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Current model: %s", c.currentModel.Name)))
	fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Max tokens: %d", c.currentModel.Config.MaxTokens)))
	fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Temperature: %.1f", c.currentModel.Config.Temperature)))

	fmt.Println()
	fmt.Printf("%s\n", theme.IndentedInfoText("To switch providers, use the /model command:"))
	fmt.Printf("%s\n", theme.IndentText("/model openrouter:anthropic/claude-3.5-sonnet"))
	fmt.Printf("%s\n", theme.IndentText("/model openai:gpt-4o"))
	fmt.Println()
	fmt.Printf("%s\n", theme.IndentedInfoText("Note: OpenRouter requires OPENROUTER_API_KEY environment variable"))
}

func handleQuit(c *Chatbot, args []string) {
	os.Exit(0)
}

func handleHistory(c *Chatbot, args []string) {
	if len(c.agentConfig.Messages) == 0 {
		fmt.Println(theme.IndentText("No conversation history."))
		return
	}

	var result strings.Builder
	result.WriteString("Conversation History:\n")
	result.WriteString("====================\n\n")

	for _, msg := range c.agentConfig.Messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		switch msg.Role {
		case "user":
			result.WriteString(fmt.Sprintf("[%s] User: %s\n", timestamp, msg.Content))
		case "assistant":
			result.WriteString(fmt.Sprintf("[%s] Agent: %s\n", timestamp, msg.Content))
		case "tool":
			result.WriteString(fmt.Sprintf("[%s] Tool (%s): %s\n", timestamp, msg.ToolName, msg.Content))
		}
	}

	fmt.Println(theme.IndentText(result.String()))
}

func handleModel(c *Chatbot, args []string) {
	if len(args) == 0 {
		// Show current model and provider info
		fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Current provider: %s", c.currentModel.Provider.Name)))
		fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Current model: %s", c.currentModel.Name)))
		fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Temperature: %.2f", c.currentModel.Config.Temperature)))
		fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Max tokens: %d", c.currentModel.Config.MaxTokens)))
		fmt.Println()

		// Show available models (registry should never be nil)
		if c.registry == nil {
			panic("registry should never be nil")
		}
		fmt.Printf("%s\n", theme.IndentedInfoText("Available models:"))
		for _, provider := range c.registry.Providers {
			fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("%s:", provider.Name)))
			for _, model := range provider.Models {
				fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("  %s:%s - %s", provider.ID, model.ID, model.Name)))
			}
		}
		fmt.Println()

		fmt.Printf("%s\n", theme.IndentedInfoText("Usage:"))
		fmt.Printf("%s\n", theme.IndentText("/model <provider>:<model-id>        - Switch provider and model"))
		fmt.Println()
		fmt.Printf("%s\n", theme.IndentedInfoText("Example:"))
		fmt.Printf("%s\n", theme.IndentText("/model openrouter:moonshotai/kimi-k2"))
		return
	}

	if len(args) == 1 {
		// Parse provider:model format
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 {
			fmt.Printf("%s\n", theme.IndentedErrorText("Invalid format. Use provider:model (e.g., openrouter:anthropic/claude-3.5-sonnet)"))
			return
		}

		provider := parts[0]
		modelID := parts[1]

		if err := c.switchProvider(provider, modelID); err != nil {
			fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to switch provider: %v", err)))
			if provider == "openrouter" {
				fmt.Println()
				fmt.Printf("%s\n", theme.IndentedInfoText("To use OpenRouter:"))
				fmt.Printf("%s\n", theme.IndentText("1. Get API key from https://openrouter.ai/"))
				fmt.Printf("%s\n", theme.IndentText("2. Set: export OPENROUTER_API_KEY=\"your-key\""))
			}
		} else {
			fmt.Printf("%s\n", theme.IndentedSuccessText(fmt.Sprintf("Switched to %s:%s", provider, modelID)))
		}
		return
	}

	// Too many arguments
	fmt.Printf("%s\n", theme.IndentedErrorText("Invalid arguments. Use /model for usage information."))
}

func handleClear(c *Chatbot, args []string) {
	c.agentConfig.ClearHistory()
	// Re-initialize default context after clearing
	c.agentConfig.InitializeDefaultContext()
	fmt.Println(theme.IndentedSuccessText("Conversation context and history cleared"))
}

func handleDebug(c *Chatbot, args []string) {
	// Handle debug mode toggling
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "on", "true", "enable":
			c.config.Debug = true
			if err := SaveConfig(c.config); err != nil {
				fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to save config: %v", err)))
				return
			}
			fmt.Println(theme.IndentedSuccessText("Debug mode enabled"))
			return
		case "off", "false", "disable":
			c.config.Debug = false
			if err := SaveConfig(c.config); err != nil {
				fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to save config: %v", err)))
				return
			}
			fmt.Println(theme.IndentedSuccessText("Debug mode disabled"))
			return
		case "status":
			status := "disabled"
			if c.config.Debug {
				status = "enabled"
			}
			fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Debug mode is %s", status)))
			return
		}
	}

	// Create timestamped debug directory
	timestamp := time.Now().Format("2006-01-02-150405")
	debugDir := fmt.Sprintf("debug-%s", timestamp)

	err := os.MkdirAll(debugDir, 0755)
	if err != nil {
		fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to create debug directory: %v", err)))
		return
	}

	var filesCreated []string

	// Write LLM request as JSON - request.json
	if c.agentConfig.Debug != nil && c.agentConfig.Debug.RecentRequest != nil {
		requestPath := filepath.Join(debugDir, "request.json")
		if err := writeJSONFile(requestPath, c.agentConfig.Debug.RecentRequest); err != nil {
			fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to write request.json: %v", err)))
		} else {
			filesCreated = append(filesCreated, "- request.json")
		}
	}

	// Write LLM response as JSON - response.json
	if c.agentConfig.Debug != nil && c.agentConfig.Debug.RecentResponse != nil {
		responsePath := filepath.Join(debugDir, "response.json")
		if err := writeJSONFile(responsePath, c.agentConfig.Debug.RecentResponse); err != nil {
			fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to write response.json: %v", err)))
		} else {
			filesCreated = append(filesCreated, "- response.json")
		}
	}

	// Write LLM chunks as an array in a JSON file - chunks.json
	if c.agentConfig.Debug != nil && len(c.agentConfig.Debug.RecentChunks) > 0 {
		chunksPath := filepath.Join(debugDir, "chunks.json")
		if err := writeJSONFile(chunksPath, c.agentConfig.Debug.RecentChunks); err != nil {
			fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to write chunks.json: %v", err)))
		} else {
			filesCreated = append(filesCreated, "- chunks.json")
		}
	}

	// Write chat history
	historyPath := filepath.Join(debugDir, "chat_history.txt")
	if err := writeChatHistory(c, historyPath); err != nil {
		fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to write chat history: %v", err)))
		return
	}
	filesCreated = append(filesCreated, "- chat_history.txt")

	// Write terminal output
	terminalPath := filepath.Join(debugDir, "terminal_output.txt")
	if err := writeTerminalOutput(c, terminalPath); err != nil {
		fmt.Printf("%s\n", theme.IndentedErrorText(fmt.Sprintf("Failed to write terminal output: %v", err)))
		return
	}
	filesCreated = append(filesCreated, "- terminal_output.txt")

	fmt.Printf("%s\n", theme.IndentedSuccessText(fmt.Sprintf("Debug files written to directory: %s", debugDir)))
	fmt.Printf("%s\n", theme.IndentText("Files created:"))
	for _, file := range filesCreated {
		fmt.Printf("%s\n", theme.IndentText(file))
	}

}

// writeJSONFile writes data as formatted JSON to a file
func writeJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(path, jsonData, 0644)
}

// writeChatHistory writes the chat history to a text file
func writeChatHistory(c *Chatbot, path string) error {
	var content strings.Builder

	content.WriteString("=== CHAT HISTORY ===\n")
	content.WriteString(fmt.Sprintf("Timestamp: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if c.agentConfig != nil {
		history := c.agentConfig.GetHistory()
		if len(history) == 0 {
			content.WriteString("No chat history available\n")
		} else {
			for _, msg := range history {
				// Format: [role timestamp] content
				timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
				content.WriteString(fmt.Sprintf("[%s %s] %s\n", msg.Role, timestamp, msg.Content))

				// Add tool call details if present
				if len(msg.ToolCalls) > 0 {
					for _, tool := range msg.ToolCalls {
						content.WriteString(fmt.Sprintf("  └─ Tool Call: %s (ID: %s)\n", tool.Function.Name, tool.ID))
						if tool.Function.Arguments != "" {
							content.WriteString(fmt.Sprintf("     Args: %s\n", tool.Function.Arguments))
						}
					}
				}

				// Add tool metadata if present
				if msg.ToolName != "" {
					content.WriteString(fmt.Sprintf("  └─ Tool Response: %s (ID: %s)\n", msg.ToolName, msg.ToolUseID))
				}
			}
		}
	} else {
		content.WriteString("Agent configuration not available\n")
	}

	return os.WriteFile(path, []byte(content.String()), 0644)
}

// writeTerminalOutput writes captured terminal output to a text file
func writeTerminalOutput(c *Chatbot, path string) error {
	var content strings.Builder

	content.WriteString("=== TERMINAL OUTPUT ===\n")
	content.WriteString(fmt.Sprintf("Timestamp: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if c.stdoutCapture != nil {
		capturedOutput := c.stdoutCapture.GetContent()
		if capturedOutput != "" {
			content.WriteString(capturedOutput)
		} else {
			content.WriteString("No terminal output captured\n")
		}
	} else {
		content.WriteString("Terminal capture not available\n")
	}

	return os.WriteFile(path, []byte(content.String()), 0644)
}

func handleContext(c *Chatbot, args []string) {
	liveContext := c.agentConfig.LiveContext
	showFull := len(args) > 0 && args[0] == "full"

	// Show context usage
	currentSize, maxSize, usagePercent := liveContext.GetContextUsage()
	fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Context Usage: %d/%d bytes (%.1f%%)", currentSize, maxSize, usagePercent)))
	fmt.Println()

	if showFull {
		fmt.Println(theme.IndentedInfoText("=== LIVE CONTEXT (FULL) ==="))
		fmt.Print(theme.IndentText(liveContext.SerializeFiles()))
		fmt.Print(theme.IndentText(liveContext.SerializeDirectories()))
		fmt.Print(theme.IndentText("\n"))
	} else {
		files := liveContext.ListFiles()
		dirs := liveContext.ListDirectories()

		fmt.Println(theme.IndentedInfoText("=== LIVE CONTEXT SUMMARY ==="))

		if len(files) > 0 {
			fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Files (%d):", len(files))))
			for _, file := range files {
				fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("- %s", file)))
			}
		}

		if len(dirs) > 0 {
			fmt.Printf("%s\n", theme.IndentedInfoText(fmt.Sprintf("Directories (%d):", len(dirs))))
			for _, dir := range dirs {
				fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("- %s", dir)))
			}
		}

		fmt.Println(theme.IndentText(""))
		fmt.Println(theme.IndentedInfoText("Use '/context full' to see complete content"))
	}
}

func handleConfig(c *Chatbot, args []string) {
	fmt.Println(theme.IndentedInfoText("=== CURRENT CONFIGURATION ==="))

	// Debug mode
	debugStatus := "disabled"
	if c.config.Debug {
		debugStatus = "enabled"
	}
	fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("Debug mode: %s", debugStatus)))

	// Current model
	if c.config.Model != nil {
		fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("Provider: %s", c.config.Model.Provider)))
		fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("Model: %s", c.config.Model.Model)))
	} else {
		fmt.Printf("%s\n", theme.IndentText("Model: not configured"))
	}

	// Configuration file location
	configPath, _ := getConfigPath()
	fmt.Printf("%s\n", theme.IndentText(fmt.Sprintf("Config file: %s", configPath)))

	fmt.Println()
}

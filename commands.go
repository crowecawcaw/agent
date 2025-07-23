package main

import (
	"agent/miniagents"
	"agent/theme"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ExecuteCommand processes a command input and executes the corresponding handler
func (a *Agent) ExecuteCommand(input string) {
	parts := strings.Fields(strings.TrimPrefix(input, "/"))
	if len(parts) == 0 {
		return
	}

	commandName := parts[0]
	args := parts[1:]

	cmd, exists := a.commands[commandName]
	if !exists {
		fmt.Printf("%s\n", theme.CommandText(fmt.Sprintf("Unknown command: /%s", commandName)))
		return
	}

	output := cmd.Handler(a, args)
	fmt.Println(theme.CommandText(output))
}

// GetAvailableCommands returns a string listing all available commands
func (a *Agent) GetAvailableCommands() string {
	var commandNames []string
	for name := range a.commands {
		commandNames = append(commandNames, "/"+name)
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Commands: " + strings.Join(commandNames, ", "))
}

type Command struct {
	Handler func(*Agent, []string) string
	Desc    string
}

var builtinCommands = map[string]Command{
	"help":    {handleHelp, "Show available commands and their descriptions"},
	"model":   {handleModel, "Show or change the AI model and provider"},
	"context": {handleContext, "Show live context summary (use 'full' to see complete content)"},
	"prune":   {handlePrune, "Prune context to reduce size (usage: /prune [target_reduction_chars])"},
	"clear":   {handleClear, "Clear conversation history"},
	"quit":    {handleQuit, "Quit to the terminal"},
}

// registerBuiltinCommands sets up all the built-in commands
func (a *Agent) registerBuiltinCommands() {
	a.commands = make(map[string]Command)
	for name, cmd := range builtinCommands {
		a.commands[name] = cmd
	}
}

func handleHelp(a *Agent, args []string) string {
	var result strings.Builder
	result.WriteString(theme.InfoText("Available Commands:") + "\n\n")

	for name, cmd := range a.commands {
		result.WriteString(fmt.Sprintf("%s - %s\n",
			theme.SuccessText(fmt.Sprintf("/%s", name)),
			theme.InfoText(cmd.Desc)))
	}

	result.WriteString("\n")
	result.WriteString(theme.InfoText("Usage examples:") + "\n")
	result.WriteString(theme.InfoText("/model anthropic.claude-3-haiku-20240307-v1:0") + "\n")
	result.WriteString(theme.InfoText("/context full") + "\n\n")

	result.WriteString(theme.InfoText("Note: Press Ctrl+C to cancel ongoing requests or quit at prompt") + "\n")

	return result.String()
}

func handleQuit(a *Agent, args []string) string {
	os.Exit(0)
	return ""
}

func handleModel(a *Agent, args []string) string {
	var result strings.Builder

	if len(args) == 0 {
		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Current model: %s:%s", a.currentModel.Provider.Name, a.currentModel.Name))))
		result.WriteString("\n")

		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("Available models:")))
		for _, provider := range a.config.Providers {
			result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("%s:", provider.Name))))
			for _, model := range provider.Models {
				result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("  %s:%s - %s", provider.ID, model.ID, model.Name))))
			}
		}
		result.WriteString("\n")

		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("Usage:")))
		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("/model <provider>:<model-id>        - Switch provider and model")))
		result.WriteString("\n")
		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("Example:")))
		result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("/model openrouter:moonshotai/kimi-k2")))
		return result.String()
	}

	if len(args) == 1 {
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 {
			return theme.ErrorText("Invalid format. Use provider:model (e.g., openrouter:anthropic/claude-3.5-sonnet)")
		}

		provider := parts[0]
		modelID := parts[1]

		if err := a.switchProvider(provider, modelID); err != nil {
			var errorMsg strings.Builder
			errorMsg.WriteString(theme.ErrorText(fmt.Sprintf("Failed to switch provider: %v", err)))
			if provider == "openrouter" {
				errorMsg.WriteString("\n")
				errorMsg.WriteString(theme.InfoText("To use OpenRouter:") + "\n")
				errorMsg.WriteString(theme.InfoText("1. Get API key from https://openrouter.ai/") + "\n")
				errorMsg.WriteString(theme.InfoText("2. Set: export OPENROUTER_API_KEY=\"your-key\"") + "\n")
			}
			return errorMsg.String()
		} else {
			return theme.SuccessText(fmt.Sprintf("Switched to %s:%s", provider, modelID))
		}
	}

	return theme.ErrorText("Invalid arguments. Use /model for usage information.")
}

func handleClear(a *Agent, args []string) string {
	a.ClearHistory()
	a.InitializeDefaultContext()
	return theme.SuccessText("Conversation context and history cleared")
}

func handleContext(a *Agent, args []string) string {
	liveContext := a.LiveContext
	showFull := len(args) > 0 && args[0] == "full"

	var result strings.Builder

	currentSize, maxSize, usagePercent := liveContext.GetContextUsage()
	result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Context Usage: %d/%d bytes (%.1f%%)", currentSize, maxSize, usagePercent))))
	result.WriteString("\n")

	if showFull {
		result.WriteString(theme.InfoText("=== LIVE CONTEXT (FULL) ===") + "\n")
		result.WriteString(theme.InfoText(liveContext.SerializeFiles()))
		result.WriteString(theme.InfoText(liveContext.SerializeDirectories()))
		result.WriteString(theme.InfoText("\n"))
	} else {
		files := liveContext.ListFiles()
		dirs := liveContext.ListDirectories()

		result.WriteString(theme.InfoText("=== LIVE CONTEXT SUMMARY ===") + "\n")

		if len(files) > 0 {
			result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Files (%d):", len(files)))))
			for _, file := range files {
				result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("- %s", file))))
			}
		}

		if len(dirs) > 0 {
			result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Directories (%d):", len(dirs)))))
			for _, dir := range dirs {
				result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("- %s", dir))))
			}
		}

		result.WriteString(theme.InfoText("") + "\n")
		result.WriteString(theme.InfoText("Use '/context full' to see complete content") + "\n")
	}

	return result.String()
}

func handlePrune(a *Agent, args []string) string {
	currentSize := a.GetContextCharacterCount()

	targetReduction := currentSize / 4

	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 {
			targetReduction = parsed
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("Starting context pruning...")))
	result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Current context size: %d characters", currentSize))))
	result.WriteString(fmt.Sprintf("%s\n", theme.InfoText(fmt.Sprintf("Target reduction: %d characters", targetReduction))))

	if a.currentModel == nil {
		return theme.ErrorText("No model configured. Use /model to set one.")
	}

	messages := a.GetHistory()

	go func() {
		ctx := context.Background()
		if err := miniagents.PruneContext(ctx, a.currentModel, &messages, a.LiveContext, a.tools); err != nil {
			fmt.Printf("%s\n", theme.ErrorText(fmt.Sprintf("Context pruning failed: %v", err)))
		} else {
			newSize := a.GetContextCharacterCount()
			actualReduction := currentSize - newSize
			fmt.Printf("%s\n", theme.SuccessText("Context pruning completed!"))
			fmt.Printf("%s\n", theme.InfoText(fmt.Sprintf("New context size: %d characters", newSize)))
			fmt.Printf("%s\n", theme.InfoText(fmt.Sprintf("Actual reduction: %d characters", actualReduction)))
		}
	}()

	result.WriteString(fmt.Sprintf("%s\n", theme.InfoText("Context pruning started in background...")))
	return result.String()
}

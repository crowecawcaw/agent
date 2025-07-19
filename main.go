package main

import (
	"agent/models"
	"agent/theme"
	"agent/tools"
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
)

// Global debug state
var globalDebugEnabled bool

// SetGlobalDebug sets the global debug state
func SetGlobalDebug(enabled bool) {
	globalDebugEnabled = enabled
}

// IsGlobalDebugEnabled returns the global debug state
func IsGlobalDebugEnabled() bool {
	return globalDebugEnabled
}

// StdoutCapture captures stdout output to memory
type StdoutCapture struct {
	buffer *bytes.Buffer
	mutex  sync.Mutex
}

func NewStdoutCapture() *StdoutCapture {
	return &StdoutCapture{
		buffer: &bytes.Buffer{},
	}
}

func (sc *StdoutCapture) Write(p []byte) (n int, err error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.buffer.Write(p)
}

func (sc *StdoutCapture) GetContent() string {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.buffer.String()
}

type Chatbot struct {
	commands        map[string]func(*Chatbot, []string)
	commandDescs    map[string]string
	config          *Config
	registry        *models.Registry
	currentModel    *models.Model
	agentConfig     *AgentConfig
	ctx             context.Context
	cancelFunc      context.CancelFunc
	inProgress      bool
	inProgressMutex sync.Mutex
	stdoutCapture   *StdoutCapture
}

func (c *Chatbot) setupStdoutCapture() {
	// Create a MultiWriter that writes to both stdout and our capture buffer
	multiWriter := io.MultiWriter(os.Stdout, c.stdoutCapture)

	// Replace os.Stdout with a custom writer that uses our MultiWriter
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("Failed to create pipe for stdout capture: %v", err)
		return
	}

	// Replace os.Stdout with our pipe writer
	os.Stdout = w

	// Start a goroutine to copy from the pipe reader to our MultiWriter
	go func() {
		defer r.Close()
		defer w.Close()
		_, _ = io.Copy(multiWriter, r)
	}()
}

func (c *Chatbot) IsDebugEnabled() bool {
	return c.config.Debug
}

func (c *Chatbot) handleAIError(err error, context string) string {
	return HandleSystemError(context, err)
}

func NewChatbot() *Chatbot {
	// Load persistent configuration
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
		config, err = createDefaultConfig()
		if err != nil {
			log.Fatalf("Failed to create default config: %v", err)
		}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	chatbot := &Chatbot{
		commands:      make(map[string]func(*Chatbot, []string)),
		commandDescs:  make(map[string]string),
		config:        config,
		agentConfig:   NewAgentConfig(),
		ctx:           ctx,
		cancelFunc:    cancelFunc,
		inProgress:    false,
		stdoutCapture: NewStdoutCapture(),
	}

	// Set up stdout capture
	chatbot.setupStdoutCapture()

	// Update global debug state and error handlers
	SetGlobalDebug(config.Debug)
	InitializeErrorHandler(config.Debug)
	tools.InitializeToolErrorHandler(config.Debug)

	// Load model registry from config
	chatbot.registry = models.LoadRegistry(config.Providers)

	// Get the configured model from persistent config
	var model *models.Model
	if config.Model != nil {
		model = chatbot.registry.GetModelByProviderAndID(config.Model.Provider, config.Model.Model)
	}

	if model == nil {
		var modelDesc string
		if config.Model != nil {
			modelDesc = fmt.Sprintf("%s/%s", config.Model.Provider, config.Model.Model)
		} else {
			modelDesc = "no model configured"
		}
		panic(fmt.Sprintf("Model %s not found in registry", modelDesc))
	}

	chatbot.currentModel = model

	// Initialize the agent configuration
	chatbot.agentConfig = NewAgentConfig()

	chatbot.registerBuiltinCommands()

	return chatbot
}

func (c *Chatbot) switchProvider(provider string, modelID string) error {
	if c.registry == nil {
		panic("registry should never be nil")
	}

	// Find the model in the registry
	model := c.registry.GetModel(modelID)
	if model == nil {
		return fmt.Errorf("model %s not found in registry", modelID)
	}

	// Verify the model belongs to the specified provider
	if model.Provider.ID != provider {
		return fmt.Errorf("model %s belongs to provider %s, not %s", modelID, model.Provider.ID, provider)
	}

	// Update chatbot state
	c.currentModel = model

	// Update persistent configuration
	c.config.Model = &SelectedModel{
		Provider: provider,
		Model:    modelID,
	}

	// Save the updated configuration
	if err := SaveConfig(c.config); err != nil {
		log.Printf("Failed to save config after model switch: %v", err)
	}

	return nil
}

func (c *Chatbot) ProcessAndPrintMessage(input string) {
	c.processWithToolLoop(input) // Handles adding user message, printing, and history
	fmt.Println()
}

func (c *Chatbot) processWithToolLoop(input string) {
	// Set in-progress flag
	c.inProgressMutex.Lock()
	c.inProgress = true
	// Create a new cancellable context
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFunc = cancelFunc
	c.inProgressMutex.Unlock()

	// Ensure we clear the in-progress flag when done
	defer func() {
		c.inProgressMutex.Lock()
		c.inProgress = false
		c.inProgressMutex.Unlock()
	}()

	// Use the simplified agent processing
	err := c.agentConfig.ProcessWithDirectService(ctx, c.currentModel, input)
	if err != nil {
		errorMsg := c.handleAIError(err, "AI generation")
		fmt.Println("")
		fmt.Println(theme.IndentedWarningText(errorMsg))
	}
}

// applyModelOverride parses and applies a model override from command line
func applyModelOverride(chatbot *Chatbot, modelSpec string) error {
	parts := strings.SplitN(modelSpec, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("model must be in format provider:model (e.g., openrouter:anthropic/claude-3.5-sonnet)")
	}

	provider := parts[0]
	model := parts[1]

	// Update the config
	chatbot.config.Model = &SelectedModel{
		Provider: provider,
		Model:    model,
	}

	// Save the updated config
	if err := SaveConfig(chatbot.config); err != nil {
		return fmt.Errorf("failed to save config with model override: %w", err)
	}

	// Reinitialize with the new model
	if err := chatbot.reinitializeOpenAIService(); err != nil {
		return fmt.Errorf("failed to initialize model service: %w", err)
	}

	return nil
}

func (c *Chatbot) reinitializeOpenAIService() error {
	if c.registry == nil {
		panic("registry should never be nil")
	}

	// Get the configured model from persistent config
	var model *models.Model
	if c.config.Model != nil {
		model = c.registry.GetModelByProviderAndID(c.config.Model.Provider, c.config.Model.Model)
	}

	if model == nil {
		return fmt.Errorf("model not found in registry")
	}

	c.currentModel = model

	return nil
}

func main() {
	// Parse command line flags
	var modelFlag = flag.String("model", "", "Set model in format provider:model (e.g., openrouter:anthropic/claude-3.5-sonnet)")
	flag.Parse()

	// Initialize theme first
	theme.InitializeTheme()

	// Initialize error handler
	InitializeErrorHandler(false) // Will be updated after chatbot is created

	chatbot := NewChatbot()

	// Update global debug state and error handlers
	SetGlobalDebug(chatbot.config.Debug)
	InitializeErrorHandler(chatbot.config.Debug)
	tools.InitializeToolErrorHandler(chatbot.config.Debug)

	// Apply model override from command line if provided
	if *modelFlag != "" {
		if err := applyModelOverride(chatbot, *modelFlag); err != nil {
			log.Fatalf("Invalid model specification: %v", err)
		}
	}

	// Set up signal handling for request cancellation on Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Handle signals in a goroutine
	go func() {
		for {
			<-sigChan
			chatbot.inProgressMutex.Lock()
			if chatbot.inProgress && chatbot.cancelFunc != nil {
				chatbot.cancelFunc()
				chatbot.inProgressMutex.Unlock()
			} else {
				chatbot.inProgressMutex.Unlock()
				fmt.Printf("\n%s\n", theme.IndentedInfoText("Exiting..."))
				os.Exit(0)
			}
		}
	}()

	// Print welcome messages
	fmt.Println(theme.IndentedInfoText("Welcome!"))
	fmt.Println(theme.IndentedCommandText(chatbot.GetAvailableCommands()))
	fmt.Println()

	// Create scanner for input
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(theme.PromptText("> "))

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Printf("Error reading input: %v\n", err)
			}
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if input == "/quit" {
				break
			}
			chatbot.ExecuteCommand(input)
			continue
		}

		// Process the message
		chatbot.ProcessAndPrintMessage(input)
		fmt.Println() // Add spacing after response
	}
}

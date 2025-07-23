package main

import (
	"agent/theme"
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
)

func main() {
	theme.InitializeTheme()
	agent := NewAgent()

	// Set up signal handling for request cancellation on Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Handle signals in a goroutine
	go func() {
		for {
			<-sigChan
			agent.inProgressMutex.Lock()
			if agent.inProgress && agent.cancelFunc != nil {
				agent.cancelFunc()
				agent.inProgressMutex.Unlock()
			} else {
				agent.inProgressMutex.Unlock()
				fmt.Printf("\n%s\n", theme.InfoText("Exiting..."))
				os.Exit(0)
			}
		}
	}()

	fmt.Println(theme.AgentText("🦜 welcome, friend\n   " + agent.GetAvailableCommands()))
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
		fmt.Printf("\033[1A\033[K") // Moves cursor up one line and clears the line
		fmt.Println(theme.UserText("👤 " + input))
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if input == "/quit" {
				break
			}
			agent.ExecuteCommand(input)
			continue
		}

		// Process the message
		agent.ProcessMessage(input) // Handles adding user message, printing, and history
		fmt.Println()
		fmt.Println()
	}

	if err := agent.Close(); err != nil {
		log.Fatalf("Failed to close chatbot: %v", err)
	}
}

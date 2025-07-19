package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"agent/theme"
)

type ShellTool struct {
	*BaseTool
	workingDir     string
	deniedCommands []string
	timeout        time.Duration
}

func NewShellTool() *ShellTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command to execute using bash",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Brief description of what the command does (optional)",
			},
			"working_directory": map[string]interface{}{
				"type":        "string",
				"description": "Directory to run command in, relative to current directory (optional)",
			},
		},
		"required": []interface{}{"command"},
	}

	tool := &ShellTool{
		BaseTool: NewBaseTool(
			"shell",
			"Execute shell commands using bash. Supports most common Unix commands and operations and supports any software the user has installed.",
			schema,
		),
		workingDir: ".",
		deniedCommands: []string{
			"sudo", "su", "rm", "rmdir", "del", "deltree",
			"format", "fdisk", "mkfs", "dd", "shutdown", "reboot",
			"halt", "poweroff", "init", "systemctl", "service",
			"chown", "chmod 777", "chmod -R 777", "killall",
			"pkill", "kill -9", "kill -KILL", "> /dev/", "curl -X DELETE",
			"wget --post-data", "nc -l", "netcat -l", "python -m http.server",
			"python3 -m http.server", "php -S", "ruby -run-httpd",
		},
		timeout: 30 * time.Second,
	}

	return tool
}

func (s *ShellTool) Execute(ctx context.Context, params map[string]interface{}, statusCh chan<- string) (string, error) {
	command, workDir, err := s.extractParams(params)
	if err != nil {
		// Return actual error instead of formatted string
		return "", NewToolError(s.Name(), "parameter validation failed", err)
	}

	// Send status update with command info
	if len(command) > 50 {
		statusCh <- fmt.Sprintf("Executing shell command: %s...\n", theme.CodeText(command[:47]))
	} else {
		statusCh <- fmt.Sprintf("Executing shell command: %s\n", theme.CodeText(command))
	}

	// Execute command
	result, err := s.executeCommand(ctx, command, workDir)

	// Send completion status
	if err != nil {
		statusCh <- fmt.Sprintf("Error running command: %s\n", err)
		return result, WrapToolError(s.Name(), err)
	}

	return result, nil
}

// extractParams consolidates parameter extraction and validation
func (s *ShellTool) extractParams(params map[string]interface{}) (string, string, error) {
	// Extract command
	command, ok := params["command"].(string)
	if !ok {
		return "", "", NewToolError(s.Name(), "command parameter must be a string", nil)
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return "", "", NewToolError(s.Name(), "command cannot be empty", nil)
	}

	// Validate command safety
	if err := s.validateCommandSafety(command); err != nil {
		return "", "", WrapToolError(s.Name(), err)
	}

	// Extract and validate working directory
	workDir := s.workingDir
	if wd, exists := params["working_directory"]; exists && wd != nil {
		wdStr, ok := wd.(string)
		if ok && wdStr != "" {
			if filepath.IsAbs(wdStr) {
				return "", "", NewToolError(s.Name(), "working_directory must be relative", nil)
			}

			absPath, err := filepath.Abs(filepath.Join(s.workingDir, wdStr))
			if err != nil {
				return "", "", NewToolError(s.Name(), fmt.Sprintf("invalid working directory: %v", err), err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return "", "", NewToolError(s.Name(), fmt.Sprintf("working directory does not exist: %s", wdStr), err)
			}

			workDir = absPath
		}
	}

	return command, workDir, nil
}

func (s *ShellTool) validateCommandSafety(command string) error {
	lowerCommand := strings.ToLower(command)

	for _, denied := range s.deniedCommands {
		deniedLower := strings.ToLower(denied)
		if strings.HasPrefix(lowerCommand, deniedLower) {
			// Ensure it's a word boundary
			if len(lowerCommand) == len(deniedLower) ||
				lowerCommand[len(deniedLower)] == ' ' ||
				lowerCommand[len(deniedLower)] == '\t' {
				return fmt.Errorf("command '%s' is not allowed for security reasons", denied)
			}
		}

		// Check after separators
		separators := []string{"|", ";", "&&", "||"}
		for _, sep := range separators {
			parts := strings.Split(lowerCommand, sep)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, deniedLower) {
					if len(part) == len(deniedLower) ||
						part[len(deniedLower)] == ' ' ||
						part[len(deniedLower)] == '\t' {
						return fmt.Errorf("command '%s' is not allowed for security reasons", denied)
					}
				}
			}
		}
	}

	return nil
}

func (s *ShellTool) executeCommand(ctx context.Context, command, workDir string) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := s.createCommand(execCtx, command, workDir)

	// Set up output capture
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %v", err)
	}

	// Stream output and capture result
	output := s.streamAndCaptureOutput(stdout, stderr)

	// Wait for completion
	err = cmd.Wait()

	// Handle different error types
	if err != nil {
		if ctx.Err() == context.Canceled {
			return s.formatResult(output, fmt.Errorf("command cancelled")), nil
		}
		if execCtx.Err() == context.DeadlineExceeded {
			return s.formatResult(output, fmt.Errorf("command timed out after %v", s.timeout)), nil
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			return s.formatResult(output, fmt.Errorf("command failed with exit code %d", exitError.ExitCode())), nil
		}
		return s.formatResult(output, fmt.Errorf("command failed: %v", err)), nil
	}

	return output, nil
}

func (s *ShellTool) createCommand(ctx context.Context, command, workDir string) *exec.Cmd {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Try bash first, fallback to cmd
		if _, err := exec.LookPath("bash"); err == nil {
			cmd = exec.CommandContext(ctx, "bash", "-c", command)
		} else {
			cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		}
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}

	cmd.Dir = workDir
	cmd.Env = os.Environ()
	return cmd
}

func (s *ShellTool) streamAndCaptureOutput(stdout, stderr io.ReadCloser) string {
	var output strings.Builder
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)

	// Stream stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("│   %s\n", theme.DebugText(line))
			mu.Lock()
			output.WriteString(line + "\n")
			mu.Unlock()
		}
	}()

	// Stream stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "│   %s\n", theme.DebugText(line))
			mu.Lock()
			output.WriteString(line + "\n")
			mu.Unlock()
		}
	}()

	wg.Wait()
	return output.String()
}

func (s *ShellTool) formatResult(output string, err error) string {
	if err == nil {
		return output
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Command failed: %v\n", err))

	if len(output) > 0 {
		result.WriteString("\nOutput:\n")
		result.WriteString(output)
	}

	return result.String()
}

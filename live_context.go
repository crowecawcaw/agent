package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxContextSize is the maximum allowed context size in bytes
const MaxContextSize = 100 * 1024 // 100kB

// FileInfo holds information about a file in live context
type FileInfo struct {
	Path      string
	StartLine int
	EndLine   *int // nil means read to end
}

// DirectoryInfo holds information about a directory in live context
type DirectoryInfo struct {
	Path            string
	IgnoreGitignore bool
	IgnorePatterns  []string
}

// LiveContext manages files and directories for the agent
type LiveContext struct {
	files       map[string]FileInfo
	directories map[string]DirectoryInfo
}

// NewLiveContext creates a new LiveContext instance
func NewLiveContext() *LiveContext {
	return &LiveContext{
		files:       make(map[string]FileInfo),
		directories: make(map[string]DirectoryInfo),
	}
}

// AddFile adds a file with optional parameters
func (lc *LiveContext) AddFile(filePath string, startLine int, endLine *int) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if startLine <= 0 {
		startLine = 1
	}

	lc.files[filePath] = FileInfo{
		Path:      filePath,
		StartLine: startLine,
		EndLine:   endLine,
	}
	return nil
}

// RemoveFile removes a file from live context
func (lc *LiveContext) RemoveFile(filePath string) error {
	if _, exists := lc.files[filePath]; !exists {
		return fmt.Errorf("file %s not found in live context", filePath)
	}
	delete(lc.files, filePath)
	return nil
}

// ListFiles returns all files in live context
func (lc *LiveContext) ListFiles() []string {
	files := make([]string, 0, len(lc.files))
	for filePath := range lc.files {
		files = append(files, filePath)
	}
	return files
}

// AddDirectory adds a directory with optional parameters
func (lc *LiveContext) AddDirectory(dirPath string, ignoreGitignore bool, ignorePatterns ...string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	lc.directories[dirPath] = DirectoryInfo{
		Path:            dirPath,
		IgnoreGitignore: ignoreGitignore,
		IgnorePatterns:  ignorePatterns,
	}
	return nil
}

// RemoveDirectory removes a directory from live context
func (lc *LiveContext) RemoveDirectory(dirPath string) error {
	if _, exists := lc.directories[dirPath]; !exists {
		return fmt.Errorf("directory %s not found in live context", dirPath)
	}
	delete(lc.directories, dirPath)
	return nil
}

// ListDirectories returns all directories in live context
func (lc *LiveContext) ListDirectories() []string {
	dirs := make([]string, 0, len(lc.directories))
	for dirPath := range lc.directories {
		dirs = append(dirs, dirPath)
	}
	return dirs
}

// SerializeFiles generates the files section of live context
func (lc *LiveContext) SerializeFiles() string {
	var sections []string

	sections = append(sections, "\n--- FILES ---")
	for filePath, fileInfo := range lc.files {
		endLineString := "end"
		if fileInfo.EndLine != nil {
			endLineString = fmt.Sprintf("%d", *fileInfo.EndLine)
		}
		sections = append(sections, fmt.Sprintf("\n--- FILE: %s [Lines %d:%s]---", filePath, fileInfo.StartLine, endLineString))

		content, err := lc.readFileWithOptions(fileInfo)
		if err != nil {
			sections = append(sections, fmt.Sprintf("Error reading file: %v", err))
		} else {
			sections = append(sections, content)
		}
	}

	if len(lc.files) == 0 {
		sections = append(sections, "No files in live context")
	}

	return strings.Join(sections, "\n")
}

// SerializeDirectories generates the directories section of live context
func (lc *LiveContext) SerializeDirectories() string {
	var sections []string

	sections = append(sections, "\n--- DIRECTORY STRUCTURES ---")
	for dirPath, dirInfo := range lc.directories {
		sections = append(sections, fmt.Sprintf("\n--- DIRECTORY: %s ---", dirPath))

		structure, err := generateDirectoryTree(
			dirInfo.Path,
			dirInfo.IgnoreGitignore,
			dirInfo.IgnorePatterns,
		)
		if err != nil {
			sections = append(sections, fmt.Sprintf("Error reading directory: %v", err))
			// TODO how to handle warnings LogWarning("live_context", "directory_read", err)
		} else {
			sections = append(sections, structure)
		}
	}

	if len(lc.directories) == 0 {
		sections = append(sections, "No directories in live context")
	}

	return strings.Join(sections, "\n")
}

// readFileWithOptions reads a file with the specified options
func (lc *LiveContext) readFileWithOptions(fileInfo FileInfo) (string, error) {
	content, err := os.ReadFile(fileInfo.Path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// Handle start and end line bounds
	startLine := fileInfo.StartLine
	if startLine < 1 {
		startLine = 1
	}
	if startLine > totalLines {
		return "", fmt.Errorf("start line %d exceeds file length %d", startLine, totalLines)
	}

	endLine := totalLines
	if fileInfo.EndLine != nil {
		if *fileInfo.EndLine < 0 {
			// Negative end line means count from end
			endLine = totalLines + *fileInfo.EndLine + 1
		} else {
			endLine = *fileInfo.EndLine
		}
	}

	if endLine > totalLines {
		endLine = totalLines
	}
	if endLine < startLine {
		return "", fmt.Errorf("end line %d is before start line %d", endLine, startLine)
	}

	// Extract the specified range (convert to 0-based indexing)
	selectedLines := lines[startLine-1 : endLine]

	// Apply line length limits and max lines
	var processedLines []string
	for i, line := range selectedLines {
		if len(processedLines) > 2000 {
			processedLines = append(processedLines, fmt.Sprintf("... (truncated after %d lines)", len(processedLines)))
			break
		}

		if len(line) > 2000 {
			line = line[:2000] + "..."
		}
		processedLines = append(processedLines, line)

		// Add line numbers if we're showing a subset
		if fileInfo.StartLine > 1 || fileInfo.EndLine != nil {
			actualLineNum := startLine + i
			processedLines[len(processedLines)-1] = fmt.Sprintf("%d: %s", actualLineNum, line)
		}
	}

	return strings.Join(processedLines, "\n"), nil
}

// generateDirectoryTree creates a flat list representation of a directory using breadth-first traversal
func generateDirectoryTree(dirPath string, ignoreGitignore bool, ignorePatterns []string) (string, error) {
	const maxItems = 100
	const maxDepth = 10 // Fixed reasonable depth limit

	// Set up exclusions
	defaultIgnores := []string{".git", "node_modules", ".vscode", ".idea", ".DS_Store"}
	ignoreMap := make(map[string]bool)
	for _, pattern := range append(defaultIgnores, ignorePatterns...) {
		ignoreMap[pattern] = true
	}

	// Breadth-first traversal
	type queueItem struct {
		path  string
		depth int
	}

	queue := []queueItem{{path: dirPath, depth: 0}}
	var results []string
	truncatedDirs := make(map[string]bool)

	for len(queue) > 0 && len(results) < maxItems {
		current := queue[0]
		queue = queue[1:]

		// Skip if we've exceeded max depth
		if current.depth > maxDepth {
			continue
		}

		entries, err := os.ReadDir(current.path)
		if err != nil {
			continue
		}

		var dirEntries []os.DirEntry
		var fileEntries []os.DirEntry

		// Separate directories and files, apply filters
		for _, entry := range entries {
			name := entry.Name()

			// Skip ignored patterns
			if ignoreMap[name] || strings.HasPrefix(name, ".") {
				continue
			}

			// Skip .log files
			if strings.HasSuffix(name, ".log") {
				continue
			}

			if entry.IsDir() {
				dirEntries = append(dirEntries, entry)
			} else {
				fileEntries = append(fileEntries, entry)
			}
		}

		// Add directories first, then files
		allEntries := append(dirEntries, fileEntries...)

		itemsAdded := 0
		for _, entry := range allEntries {
			if len(results) >= maxItems {
				// Mark this directory as truncated
				truncatedDirs[current.path] = true
				break
			}

			fullPath := filepath.Join(current.path, entry.Name())
			relPath, err := filepath.Rel(dirPath, fullPath)
			if err != nil {
				continue
			}

			displayPath := "./" + relPath
			if entry.IsDir() {
				displayPath += "/"
				// Add to queue for next level
				queue = append(queue, queueItem{path: fullPath, depth: current.depth + 1})
			} else {
				// Always include file sizes for better LLM context
				if info, err := entry.Info(); err == nil {
					size := info.Size()
					if size < 1024 {
						displayPath += fmt.Sprintf(" (%d B)", size)
					} else if size < 1024*1024 {
						displayPath += fmt.Sprintf(" (%.1f KB)", float64(size)/1024)
					} else {
						displayPath += fmt.Sprintf(" (%.1f MB)", float64(size)/(1024*1024))
					}
				}
			}

			results = append(results, displayPath)
			itemsAdded++
		}

		// If we couldn't add all items from this directory, mark as truncated
		if itemsAdded < len(allEntries) {
			truncatedDirs[current.path] = true
		}
	}

	// Add truncation indicators for directories that weren't fully explored
	if len(truncatedDirs) > 0 {
		for dirPath := range truncatedDirs {
			relPath, err := filepath.Rel(dirPath, dirPath)
			if err == nil && relPath != "." {
				displayPath := "./" + relPath + "/..."
				results = append(results, displayPath)
			} else {
				results = append(results, "./...")
			}
		}
	}

	return strings.Join(results, "\n"), nil
}

// GetContextUsage returns current size, max size, and usage percentage
func (lc *LiveContext) GetContextUsage() (int, int, float64) {
	// Calculate current context size
	filesContent := lc.SerializeFiles()
	dirsContent := lc.SerializeDirectories()
	currentSize := len(filesContent) + len(dirsContent)

	usagePercent := float64(currentSize) / float64(MaxContextSize) * 100
	return currentSize, MaxContextSize, usagePercent
}

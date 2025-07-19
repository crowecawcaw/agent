package theme

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StyleType int

const (
	StylePrompt StyleType = iota
	StyleSuccess
	StyleError
	StyleWarning
	StyleInfo
	StyleThinking
	StyleTool
	StyleCommand
	StyleDebug
	StyleAgent
	StyleCode
	StyleCodeBlock
	StyleTreeBranch
	StyleTreeFile
	StyleTreeDir
	StyleTreeSize
)

type Theme struct {
	styles map[StyleType]lipgloss.Style
}

var theme *Theme

func InitializeTheme() {
	theme = &Theme{
		styles: map[StyleType]lipgloss.Style{
			StylePrompt:     lipgloss.NewStyle().Foreground(lipgloss.Color("13")),                                 // Pink
			StyleSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")),                                  // Green
			StyleError:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")),                                  // Red
			StyleWarning:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),                                  // Yellow
			StyleInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),                                  // Cyan
			StyleThinking:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")),                                  // Cyan
			StyleTool:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")),                                 // Bright Blue
			StyleCommand:    lipgloss.NewStyle().Foreground(lipgloss.Color("218")),                                // Pink
			StyleDebug:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),                                  // Grey
			StyleAgent:      lipgloss.NewStyle(),                                                                  // No color (default terminal color)
			StyleCode:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("0")), // Green on black
			StyleCodeBlock:  lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("8")), // Green on grey
			StyleTreeBranch: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),                                  // Grey
			StyleTreeFile:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),                                 // White
			StyleTreeDir:    lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),                       // Bold blue
			StyleTreeSize:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")),                                  // Grey
		},
	}
}

// Core styling functions
func StyledText(text string, styleType StyleType) string {
	if theme == nil {
		return text
	}
	return theme.styles[styleType].Render(text)
}

func IndentedText(text string, styleType StyleType) string {
	return StyledText(IndentText(text), styleType)
}

// Indentation functions
func IndentText(text string) string {
	return lipgloss.NewStyle().MarginLeft(2).BorderLeft(true).BorderForeground(lipgloss.Color("13")).Render(text)
}

// Convenience functions for common styles (backward compatibility)
func PromptText(text string) string    { return StyledText(text, StylePrompt) }
func SuccessText(text string) string   { return StyledText(text, StyleSuccess) }
func ErrorText(text string) string     { return StyledText(text, StyleError) }
func WarningText(text string) string   { return StyledText(text, StyleWarning) }
func InfoText(text string) string      { return StyledText(text, StyleInfo) }
func ThinkingText(text string) string  { return StyledText(text, StyleThinking) }
func ToolText(text string) string      { return StyledText(text, StyleTool) }
func CommandText(text string) string   { return StyledText(text, StyleCommand) }
func DebugText(text string) string     { return StyledText(text, StyleDebug) }
func AgentText(text string) string     { return StyledText(text, StyleAgent) }
func CodeText(text string) string      { return StyledText(text, StyleCode) }
func CodeBlockText(text string) string { return StyledText(text, StyleCodeBlock) }

// Indented convenience functions
func IndentedSuccessText(text string) string  { return IndentedText(text, StyleSuccess) }
func IndentedErrorText(text string) string    { return IndentedText(text, StyleError) }
func IndentedWarningText(text string) string  { return IndentedText(text, StyleWarning) }
func IndentedInfoText(text string) string     { return IndentedText(text, StyleInfo) }
func IndentedThinkingText(text string) string { return IndentedText(text, StyleThinking) }
func IndentedToolText(text string) string     { return IndentedText(text, StyleTool) }
func IndentedCommandText(text string) string  { return IndentedText(text, StyleCommand) }
func IndentedDebugText(text string) string    { return IndentedText(text, StyleDebug) }
func IndentedAgentText(text string) string    { return IndentedText(text, StyleAgent) }

// Tree rendering functions
func RenderTreeBranch(text string) string { return StyledText(text, StyleTreeBranch) }
func RenderTreeFile(text string) string   { return StyledText(text, StyleTreeFile) }
func RenderTreeDir(text string) string    { return StyledText(text, StyleTreeDir) }
func RenderTreeSize(text string) string   { return StyledText(text, StyleTreeSize) }



// Print functions with formatting and indentation
func PrintStyled(styleType StyleType, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	fmt.Print(IndentedText(text, styleType))
}

func PrintSuccess(format string, args ...interface{})  { PrintStyled(StyleSuccess, format, args...) }
func PrintError(format string, args ...interface{})    { PrintStyled(StyleError, format, args...) }
func PrintWarning(format string, args ...interface{})  { PrintStyled(StyleWarning, format, args...) }
func PrintInfo(format string, args ...interface{})     { PrintStyled(StyleInfo, format, args...) }
func PrintThinking(format string, args ...interface{}) { PrintStyled(StyleThinking, format, args...) }
func PrintTool(format string, args ...interface{})     { PrintStyled(StyleTool, format, args...) }
func PrintCommand(format string, args ...interface{})  { PrintStyled(StyleCommand, format, args...) }
func PrintDebug(format string, args ...interface{})    { PrintStyled(StyleDebug, format, args...) }
func PrintAgent(format string, args ...interface{})    { PrintStyled(StyleAgent, format, args...) }
func PrintLog(format string, args ...interface{})      { PrintStyled(StyleDebug, format, args...) }

// Utility function for tool output
func IndentAndStyleToolOutput(text string) string {
	return IndentedText(text, StyleDebug)
}

// MarkdownState represents the current parsing state
type MarkdownState int

const (
	StateNormal MarkdownState = iota
	StateHeader
	StateBold
	StateItalic
	StateCodeBlock
	StateInlineCode
)

// MarkdownRenderer handles streaming markdown rendering with basic styling
type MarkdownRenderer struct {
	state            MarkdownState
	lineStart        bool
	codeBlock        bool
	// indenter         *StreamingIndenter // Removed as StreamingIndenter is dead code
	headerBuffer     strings.Builder
	boldBuffer       strings.Builder
	italicBuffer     strings.Builder
	codeBuffer       strings.Builder
	inlineCodeBuffer strings.Builder
	pendingStars     int
}

// NewMarkdownRenderer creates a new streaming markdown renderer
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		lineStart: true,
		// indenter:  NewStreamingIndenter(),
	}
}

// Write processes incoming markdown tokens and renders them with styling
func (mr *MarkdownRenderer) Write(data []byte) (int, error) {
	text := string(data)

	for _, char := range text {
		mr.processChar(char)
	}

	return len(data), nil
}

// processChar handles a single character in the markdown stream
func (mr *MarkdownRenderer) processChar(char rune) {
	switch mr.state {
	case StateNormal:
		mr.processNormalChar(char)
	case StateHeader:
		mr.processHeaderChar(char)
	case StateBold:
		mr.processBoldChar(char)
	case StateItalic:
		mr.processItalicChar(char)
	case StateCodeBlock:
		mr.processCodeBlockChar(char)
	case StateInlineCode:
		mr.processInlineCodeChar(char)
	}
}

// processNormalChar handles characters in normal text state
func (mr *MarkdownRenderer) processNormalChar(char rune) {
	switch char {
	case '#':
		if mr.lineStart {
			mr.state = StateHeader
			mr.headerBuffer.Reset()
			mr.headerBuffer.WriteRune(char)
			return
		}
	case '*':
		mr.pendingStars++
		if mr.pendingStars == 1 {
			// Wait to see if we get a second star
			return
		} else if mr.pendingStars == 2 {
			mr.state = StateBold
			mr.boldBuffer.Reset()
			mr.pendingStars = 0
			return
		}
	case '`':
		if mr.checkForCodeBlock() {
			return
		}
		mr.state = StateInlineCode
		mr.inlineCodeBuffer.Reset()
		return
	default:
		// Handle pending stars
		if mr.pendingStars == 1 {
			// Single star followed by non-star - start italic
			if char != ' ' && char != '\t' && char != '\n' {
				mr.state = StateItalic
				mr.italicBuffer.Reset()
				mr.italicBuffer.WriteRune(char)
				mr.pendingStars = 0
				return
			} else {
				// Single star followed by whitespace - just output the star
				mr.outputChar('*')
				mr.pendingStars = 0
			}
		}
		mr.outputChar(char)
	}
}

// processHeaderChar handles characters while parsing a header
func (mr *MarkdownRenderer) processHeaderChar(char rune) {
	if char == '\n' {
		// End of header line - apply styling and output
		headerText := mr.headerBuffer.String()
		styledHeader := mr.styleHeader(headerText)
		mr.outputText(styledHeader)
		mr.outputChar('\n')
		mr.state = StateNormal
		mr.lineStart = true
	} else {
		mr.headerBuffer.WriteRune(char)
		mr.lineStart = false
	}
}

// processBoldChar handles characters while parsing bold text
func (mr *MarkdownRenderer) processBoldChar(char rune) {
	if char == '*' {
		mr.pendingStars++
		if mr.pendingStars == 2 {
			// End of bold text - apply cyan bold styling and output
			boldText := mr.boldBuffer.String()
			boldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true) // Cyan and bold
			styledBold := boldStyle.Render(boldText)
			mr.outputText(styledBold)
			mr.state = StateNormal
			mr.pendingStars = 0
		}
	} else {
		// Output any pending stars as part of bold content
		for i := 0; i < mr.pendingStars; i++ {
			mr.boldBuffer.WriteRune('*')
		}
		mr.pendingStars = 0
		mr.boldBuffer.WriteRune(char)
	}
}

// processItalicChar handles characters while parsing italic text
func (mr *MarkdownRenderer) processItalicChar(char rune) {
	if char == '*' {
		// End of italic text - apply italic styling and output
		italicText := mr.italicBuffer.String()
		italicStyle := lipgloss.NewStyle().Italic(true)
		styledItalic := italicStyle.Render(italicText)
		mr.outputText(styledItalic)
		mr.state = StateNormal
	} else {
		mr.italicBuffer.WriteRune(char)
	}
}

// processCodeBlockChar handles characters while parsing a code block
func (mr *MarkdownRenderer) processCodeBlockChar(char rune) {
	mr.codeBuffer.WriteRune(char)

	// Check for end of code block (``` on its own line)
	if char == '\n' {
		content := mr.codeBuffer.String()
		lines := strings.Split(content, "\n")
		if len(lines) >= 2 && strings.TrimSpace(lines[len(lines)-2]) == "```" {
			// End of code block - apply styling and output
			codeContent := strings.Join(lines[:len(lines)-2], "\n")
			if codeContent != "" {
				styledCode := mr.styleCodeBlock(codeContent)
				mr.outputText(styledCode)
			}
			mr.outputText("```\n") // Output closing marker
			mr.state = StateNormal
			mr.codeBlock = false
			mr.lineStart = true
		}
	} else {
		mr.lineStart = false
	}
}

// processInlineCodeChar handles characters while parsing inline code
func (mr *MarkdownRenderer) processInlineCodeChar(char rune) {
	if char == '`' {
		// End of inline code - apply styling and output
		codeText := mr.inlineCodeBuffer.String()
		styledCode := StyledText(codeText, StyleCode)
		mr.outputText(styledCode)
		mr.state = StateNormal
	} else {
		mr.inlineCodeBuffer.WriteRune(char)
	}
}

// checkForCodeBlock checks if we're starting a code block (```)
func (mr *MarkdownRenderer) checkForCodeBlock() bool {
	// This is a simplified check - in a real implementation you'd need to
	// buffer and check for three backticks
	if mr.lineStart {
		mr.state = StateCodeBlock
		mr.codeBuffer.Reset()
		mr.codeBuffer.WriteString("```")
		mr.codeBlock = true
		return true
	}
	return false
}

// styleHeader applies header styling based on the number of # characters
func (mr *MarkdownRenderer) styleHeader(headerText string) string {
	// Count leading # characters
	level := 0
	for _, char := range headerText {
		if char == '#' {
			level++
		} else {
			break
		}
	}

	// Extract the actual header text (remove # and spaces)
	text := strings.TrimSpace(headerText[level:])

	// Apply bold styling to headers
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true) // Cyan and bold

	switch level {
	case 1:
		return style.Render("# " + text)
	case 2:
		return style.Render("## " + text)
	case 3:
		return style.Render("### " + text)
	default:
		return style.Render(headerText)
	}
}

// styleCodeBlock applies code block styling
func (mr *MarkdownRenderer) styleCodeBlock(codeText string) string {
	return StyledText(codeText, StyleCodeBlock)
}

// outputChar outputs a single character with proper indentation
func (mr *MarkdownRenderer) outputChar(char rune) {
	fmt.Print(string(char))

	if char == '\n' {
		mr.lineStart = true
	} else if char != ' ' && char != '	' {
		mr.lineStart = false
	}
}

// outputText outputs text with proper indentation
func (mr *MarkdownRenderer) outputText(text string) {
	fmt.Print(text)

	// Update lineStart based on the last character
	if strings.HasSuffix(text, "\n") {
		mr.lineStart = true
	} else if strings.TrimSpace(text) != "" {
		mr.lineStart = false
	}
}

// Flush outputs any remaining buffered content
func (mr *MarkdownRenderer) Flush() {
	// Output any remaining content in buffers
	switch mr.state {
	case StateHeader:
		headerText := mr.headerBuffer.String()
		if headerText != "" {
			styledHeader := mr.styleHeader(headerText)
			mr.outputText(styledHeader)
		}
	case StateBold:
		// Output ** and the buffered content
		mr.outputText("**" + mr.boldBuffer.String())
	case StateItalic:
		// Output * and the buffered content
		mr.outputText("*" + mr.italicBuffer.String())
	case StateCodeBlock:
		// Output the code block content
		codeContent := mr.codeBuffer.String()
		if codeContent != "" {
			styledCode := mr.styleCodeBlock(codeContent)
			mr.outputText(styledCode)
		}
	case StateInlineCode:
		// Output ` and the buffered content
		mr.outputText("`" + mr.inlineCodeBuffer.String())
	}

	// Output any pending stars
	for i := 0; i < mr.pendingStars; i++ {
		mr.outputChar('*')
	}

	mr.state = StateNormal
	mr.pendingStars = 0
}

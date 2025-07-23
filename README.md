# Agent

This project is a CLI agent where I can experiment with some ideas about managing context.

The distinctive feature of this agent is **Live Context**. Most agents use tool calls to read content into the chat logs as static entries, but the context fills up over time, contains irrelevant details, and retains outdated information. With Live Context, this agent adds and removes files and other information into its context which is kept up to date. File contents are always fresh, the file structure is always accurate, and the conversation history doesn't accumulate old file contents over time.

Supported live context tools:
- add/remove file (with optional line range and length limits)
- add/remove directory structure (with optional depth, file sizes, gitignore support, and custom ignore patterns)

Some other features I'd like to add in the future:
- adding a secondary agent to run an out-of-band context pruning process to remove messages and files when they are no longer needed
- support a human-readable security policy for shell commands which is enforced by a secondary agent

## Getting Started

```bash
git clone https://github.com/crowecawcaw/agent
cd agent
make build
./bin/agent
```

### Prerequisites
- Go 1.19 or later
- AI service credentials (choose one):
  - OpenAI API key
  - OpenRouter API key

### Configuration
The agent uses a persistent JSON configuration file (`~/.agent/config.json`) to store settings between sessions.

### Environment Variables
- `OPENAI_API_KEY` - OpenAI API key
- `OPENROUTER_API_KEY` - OpenRouter API key

### Build Commands
```bash
make build              # Build the application
make run                # Build and run the application
make test               # Run all tests
make lint               # Run golangci-lint
make check              # Run lint and tests
make dev                # Full development workflow (clean, lint, test, build)
make deps               # Install dependencies and tools
```

---

## Development

### Code Patterns

#### Error Handling
- Use Go's error wrapping (`fmt.Errorf("%w", ...)`) and unwrapping (`errors.Is`, `errors.As`) to consolidate repetitive error patterns and manage context.
- Keep error messages user-friendly and actionable, leveraging error unwrapping to show simple messages to users while logging technical details.

#### Code Organization
- Prefer direct data structures over wrapper abstractions
- Let individual tools handle their own parameter type conversion
- Use generic naming that doesn't tie code to specific AI models
- Keep tool call processing simple with single-pass regex patterns

### Agent Instructions

When working with this codebase, follow these guidelines:

- NEVER add obvious comments when code is simple to understand. Prefer good function and variable names and clear code to comments when possible.
- Only add unit tests when prompted. Unit tests should be human readable and test high level functionality. Prefer one larger test to many small tests. cover important functionality but do not aim for 100% coverage. 
- Use lipgloss for styling. Never use ANSI color codes directly
- avoid extra abstractions and unnecessary helpers. when refactoring, try to eliminate functions when possible. I prefer code with larger functions and fewer helpers
- After making changes, run these commands in order:
  1. `make build` - Build the package and fix any compilation errors
  2. `make lint` - Run the linter and fix any linting errors  
  3. `echo "/quit" | ./bin/agent` - Start the agent and immediately quit to verify it starts successfully
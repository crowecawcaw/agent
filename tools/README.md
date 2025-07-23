# Tools Package

Simple function-based tool system for extending agent capabilities.

## Tool Description Guidelines

Tool descriptions should clarify what the user will see when the tool runs:
- **Shell commands** show output directly to the user in their terminal
- **Context tools** only affect the agent's available information (user doesn't see the file contents)
- **Other tools** should specify if they produce user-visible output or just return data to the agent

For example, when a user asks to see file contents, the agent should use shell commands knowing the user will see the result.

## Structure

Tools are `ToolDefinition` structs with a `ToolFunc` (see `tool.go`). Functions must be thread-safe and handle context cancellation.

**ToolFunc signature**: `func(ctx, params) (userMessage, agentMessage, error)`
- **userMessage**: Rich formatted message for humans (empty if tool prints directly)
- **agentMessage**: Minimal status for the agent  
- **error**: Any error that occurred

## Output Patterns

**Simple Tools** (file operations, context tools): Return rich userMessage and minimal agentMessage. Agent automatically prints userMessage with consistent formatting.

## Context Tools

**Primary file access method** - Use these instead of shell commands:
- `read_file` - Read file contents (replaces `cat`, `head`, `tail`)
- `stop_reading_file` - Stop reading file contents
- `read_directory` - Read nested directory structure as flat list (replaces `ls`, `find`)
- `stop_reading_directory` - Stop reading directory structure

Files/directories being read are automatically included with current contents in every request.

## Error Handling

Always use `ToolError` (see `tool.go`) for consistent, user-friendly error reporting with technical details preserved.

## Adding Tools

1. Create `ToolFunc` following signature in `tool.go`
2. Add to `registry.go`
3. See existing tools for patterns

package tools

import "agent/models"

// NewToolRegistry creates a map of all available tools
func NewToolRegistry(liveContext LiveContextManager, deleteMessageFunc DeleteMessageFunc, getModel func() *models.Model) map[string]models.ToolDefinition {
	tools := make(map[string]models.ToolDefinition)

	// File tools
	tools["create_file"] = NewCreateFileTool()
	tools["edit_file"] = NewEditFileTool()
	tools["delete_file"] = NewDeleteFileTool()

	// Shell tool
	tools["shell"] = NewShellTool(getModel)

	// Context tools (only add if dependencies are provided)
	if liveContext != nil {
		tools["read_file"] = NewReadFileTool(liveContext)
		tools["stop_reading_file"] = NewStopReadingFileTool(liveContext)
		tools["read_directory"] = NewReadDirectoryTool(liveContext)
		tools["stop_reading_directory"] = NewStopReadingDirectoryTool(liveContext)
		tools["remove_message"] = NewRemoveMessageTool(deleteMessageFunc)

	}

	return tools
}

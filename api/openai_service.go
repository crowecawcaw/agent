package api

import (
	"agent/models"
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Streaming request to the OpenAI-compatible API
func Invoke(
	ctx context.Context,
	model *models.Model,
	messages []models.Message,
	systemPrompt string,
	availableTools map[string]models.ToolDefinition,
	onReceiveContent func(string),
) (string, []models.ToolCall, error) {
	client := openai.NewClient(
		option.WithAPIKey(model.Provider.APIKey),
		option.WithBaseURL(model.Provider.BaseURL),
	)

	// Create request parameters
	request := openai.ChatCompletionNewParams{
		Model:       model.ID,
		Messages:    convertMessages(messages, systemPrompt),
		MaxTokens:   openai.Int(int64(model.Config.MaxTokens)),
		Temperature: openai.Float(model.Config.Temperature),
		TopP:        openai.Float(model.Config.TopP),
		Tools:       convertTools(availableTools),
	}

	// Create streaming request
	chatStream := client.Chat.Completions.NewStreaming(ctx, request)
	defer chatStream.Close()

	// Use OpenAI's accumulator to properly handle streaming tool calls
	acc := openai.ChatCompletionAccumulator{}
	var content string
	var toolCalls []models.ToolCall

	// Process streaming response
	for chatStream.Next() {
		chunk := chatStream.Current()

		// Add chunk to accumulator
		acc.AddChunk(chunk)

		// Handle content tokens
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			token := chunk.Choices[0].Delta.Content
			content += token
			if onReceiveContent != nil {
				onReceiveContent(token)
			}
		}

		// Check for completed tool calls
		if tool, ok := acc.JustFinishedToolCall(); ok {
			toolCall := models.ToolCall{
				ID:   tool.ID,
				Type: "function",
				Function: models.FunctionCall{
					Name:      tool.Name,
					Arguments: tool.Arguments,
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	if err := chatStream.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return "", nil, fmt.Errorf("request cancelled: %w", err)
		}
		return "", nil, fmt.Errorf("%s stream error: %w", model.Provider.Name, err)
	}

	return content, toolCalls, nil
}

// Helper methods

func convertMessages(messages []models.Message, systemPrompt string) []openai.ChatCompletionMessageParamUnion {
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	openaiMessages = append(openaiMessages, openai.SystemMessage(systemPrompt))

	// Convert messages
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				var toolCalls []openai.ChatCompletionMessageToolCallParam
				for _, tc := range msg.ToolCalls {
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID:   tc.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}

				assistantParam := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCalls,
				}
				if msg.Content != "" {
					assistantParam.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					}
				}

				openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantParam,
				})
			} else {
				openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
			}
		case "tool":
			openaiMessages = append(openaiMessages, openai.ToolMessage(msg.Content, msg.ToolCallID))
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		}
	}

	return openaiMessages
}

func convertTools(availableTools map[string]models.ToolDefinition) []openai.ChatCompletionToolParam {
	var openaiTools []openai.ChatCompletionToolParam

	for _, tool := range availableTools {
		schema := tool.Schema

		openaiTool := openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  schema,
			},
		}

		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools
}

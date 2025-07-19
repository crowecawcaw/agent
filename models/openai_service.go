package models

import (
	"agent/tools"
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Request makes a streaming request to the OpenAI-compatible API
func Request(
	ctx context.Context,
	model *Model,
	messages []Message,
	systemPrompt string,
	availableTools map[string]tools.Tool,
	onReceiveContent func(string),
	onReceiveToolCall func(ToolCall),
	onChunkReceived func(interface{}),
) (string, []ToolCall, error) {
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
	var toolCalls []ToolCall

	// Process streaming response
	for chatStream.Next() {
		chunk := chatStream.Current()

		// Send chunk for debugging if callback provided
		if onChunkReceived != nil {
			onChunkReceived(chunk)
		}

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
			toolCall := ToolCall{
				ID:   tool.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      tool.Name,
					Arguments: tool.Arguments,
				},
			}
			toolCalls = append(toolCalls, toolCall)

			if onReceiveToolCall != nil {
				onReceiveToolCall(toolCall)
			}
		}
	}

	if err := chatStream.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return "", nil, fmt.Errorf("request cancelled: %w", err)
		}
		return "", nil, fmt.Errorf("%s stream error: %w", model.Provider.Name, err)
	}

	// Log token usage if available
	if acc.Usage.PromptTokens > 0 {
		fmt.Printf("Token usage - Prompt: %d, Completion: %d, Total: %d\n", 
			acc.Usage.PromptTokens, acc.Usage.CompletionTokens, acc.Usage.TotalTokens)
	}

	return content, toolCalls, nil
}

// Helper methods

func convertMessages(messages []Message, systemPrompt string) []openai.ChatCompletionMessageParamUnion {
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	// Add system prompt if provided
	if systemPrompt != "" {
		openaiMessages = append(openaiMessages, openai.SystemMessage(systemPrompt))
	}

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

func convertTools(availableTools map[string]tools.Tool) []openai.ChatCompletionToolParam {
	var openaiTools []openai.ChatCompletionToolParam

	for _, tool := range availableTools {
		schema := tool.InputSchema()

		openaiTool := openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name(),
				Description: openai.String(tool.Description()),
				Parameters:  schema,
			},
		}

		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools
}

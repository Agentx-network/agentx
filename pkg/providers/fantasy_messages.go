package providers

import (
	"encoding/json"

	"charm.land/fantasy"
)

// AgentXToFantasyMessages converts AgentX session messages to Fantasy format.
func AgentXToFantasyMessages(messages []Message) []fantasy.Message {
	result := make([]fantasy.Message, 0, len(messages))

	for _, msg := range messages {
		fMsg := convertOneMessage(msg)
		result = append(result, fMsg)
	}

	return result
}

func convertOneMessage(msg Message) fantasy.Message {
	switch msg.Role {
	case "system":
		// System message: use SystemParts if available for cache-aware support
		if len(msg.SystemParts) > 0 {
			parts := make([]fantasy.MessagePart, 0, len(msg.SystemParts))
			for _, block := range msg.SystemParts {
				parts = append(parts, fantasy.TextPart{Text: block.Text})
			}
			return fantasy.Message{
				Role:    fantasy.MessageRoleSystem,
				Content: parts,
			}
		}
		return fantasy.NewSystemMessage(msg.Content)

	case "user":
		return fantasy.NewUserMessage(msg.Content)

	case "assistant":
		parts := make([]fantasy.MessagePart, 0)
		if msg.Content != "" {
			parts = append(parts, fantasy.TextPart{Text: msg.Content})
		}
		if msg.ReasoningContent != "" {
			parts = append(parts, fantasy.ReasoningPart{Text: msg.ReasoningContent})
		}
		// Convert tool calls to Fantasy ToolCallParts
		for _, tc := range msg.ToolCalls {
			argsStr := ""
			if tc.Function != nil {
				argsStr = tc.Function.Arguments
			} else if tc.Arguments != nil {
				raw, _ := json.Marshal(tc.Arguments)
				argsStr = string(raw)
			}
			parts = append(parts, fantasy.ToolCallPart{
				ToolCallID: tc.ID,
				ToolName:   toolCallName(tc),
				Input:      argsStr,
			})
		}
		return fantasy.Message{
			Role:    fantasy.MessageRoleAssistant,
			Content: parts,
		}

	case "tool":
		return fantasy.Message{
			Role: fantasy.MessageRoleTool,
			Content: []fantasy.MessagePart{
				fantasy.ToolResultPart{
					ToolCallID: msg.ToolCallID,
					Output:     fantasy.ToolResultOutputContentText{Text: msg.Content},
				},
			},
		}

	default:
		// Fallback: treat as user message
		return fantasy.NewUserMessage(msg.Content)
	}
}

// toolCallName extracts the tool name from a ToolCall.
func toolCallName(tc ToolCall) string {
	if tc.Name != "" {
		return tc.Name
	}
	if tc.Function != nil {
		return tc.Function.Name
	}
	return ""
}

// FantasyStepToAgentXMessages converts a Fantasy StepResult to AgentX messages for session storage.
func FantasyStepToAgentXMessages(step fantasy.StepResult) []Message {
	var result []Message

	// Build assistant message from step content
	assistantMsg := Message{
		Role: "assistant",
	}

	var toolResultMsgs []Message

	for _, content := range step.Content {
		switch content.GetType() {
		case fantasy.ContentTypeText:
			if tc, ok := fantasy.AsContentType[fantasy.TextContent](content); ok {
				assistantMsg.Content = tc.Text
			}
		case fantasy.ContentTypeReasoning:
			if rc, ok := fantasy.AsContentType[fantasy.ReasoningContent](content); ok {
				assistantMsg.ReasoningContent = rc.Text
			}
		case fantasy.ContentTypeToolCall:
			if tc, ok := fantasy.AsContentType[fantasy.ToolCallContent](content); ok {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, ToolCall{
					ID:   tc.ToolCallID,
					Type: "function",
					Name: tc.ToolName,
					Function: &FunctionCall{
						Name:      tc.ToolName,
						Arguments: tc.Input,
					},
				})
			}
		case fantasy.ContentTypeToolResult:
			if tr, ok := fantasy.AsContentType[fantasy.ToolResultContent](content); ok {
				toolContent := ""
				if tr.Result != nil {
					if textResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](tr.Result); ok {
						toolContent = textResult.Text
					} else if errResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](tr.Result); ok {
						if errResult.Error != nil {
							toolContent = errResult.Error.Error()
						}
					}
				}
				toolResultMsgs = append(toolResultMsgs, Message{
					Role:       "tool",
					Content:    toolContent,
					ToolCallID: tr.ToolCallID,
				})
			}
		}
	}

	// Only add assistant message if it has content or tool calls
	if assistantMsg.Content != "" || len(assistantMsg.ToolCalls) > 0 {
		result = append(result, assistantMsg)
	}

	// Add tool result messages after assistant
	result = append(result, toolResultMsgs...)

	return result
}

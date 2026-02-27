package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"charm.land/fantasy"

	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/providers"
	"github.com/Agentx-network/agentx/pkg/tools"
	"github.com/Agentx-network/agentx/pkg/utils"
)

// runFantasyIteration runs the LLM + tool call loop using Fantasy SDK.
// Returns the final text content, step count, and any error.
func (al *AgentLoop) runFantasyIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
) (string, int, error) {
	model := agent.FantasyModel
	if model == nil {
		return "", 0, fmt.Errorf("fantasy model not configured for agent %s", agent.ID)
	}

	// Extract system prompt from first message
	systemPrompt := ""
	startIdx := 0
	if len(messages) > 0 && messages[0].Role == "system" {
		systemPrompt = messages[0].Content
		startIdx = 1
	}

	// Fantasy SDK requires Prompt (string) = current user message,
	// Messages = prior conversation history.
	// Extract the last user message as the prompt, rest as history.
	prompt := ""
	var historyMessages []providers.Message
	for i := len(messages) - 1; i >= startIdx; i-- {
		if messages[i].Role == "user" {
			prompt = messages[i].Content
			historyMessages = messages[startIdx:i]
			break
		}
	}

	if prompt == "" {
		// Fallback: use the last message content as prompt regardless of role
		if len(messages) > startIdx {
			prompt = messages[len(messages)-1].Content
			historyMessages = messages[startIdx : len(messages)-1]
		}
	}

	fantasyMessages := providers.AgentXToFantasyMessages(historyMessages)

	// Wrap tools
	forUserSink := func(content string) {
		if opts.SendResponse && content != "" {
			al.bus.PublishOutbound(bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: content,
			})
		}
	}

	fantasyTools := tools.AdaptToolsForFantasy(agent.Tools, forUserSink)

	// Create agent with options
	maxTokens := int64(agent.MaxTokens)
	temperature := agent.Temperature

	fantasyAgent := fantasy.NewAgent(model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(fantasyTools...),
		fantasy.WithMaxOutputTokens(maxTokens),
		fantasy.WithTemperature(temperature),
		fantasy.WithStopConditions(fantasy.StepCountIs(agent.MaxIterations)),
	)

	// Set up tool context in the context
	ctx = tools.WithToolContext(ctx, tools.ToolContext{
		Channel: opts.Channel,
		ChatID:  opts.ChatID,
	})

	// Track text content and steps
	var textBuf strings.Builder
	var stepCount int
	var mu sync.Mutex

	// Run with streaming
	result, err := fantasyAgent.Stream(ctx, fantasy.AgentStreamCall{
		Prompt:   prompt,
		Messages: fantasyMessages,

		OnTextDelta: func(id, text string) error {
			mu.Lock()
			textBuf.WriteString(text)
			mu.Unlock()

			// Publish stream delta
			al.bus.PublishStreamDelta(bus.StreamDelta{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Delta:   text,
			})
			return nil
		},

		OnToolCall: func(tc fantasy.ToolCallContent) error {
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s", tc.ToolName),
				map[string]any{
					"agent_id":     agent.ID,
					"tool":         tc.ToolName,
					"tool_call_id": tc.ToolCallID,
				})
			return nil
		},

		OnToolResult: func(tr fantasy.ToolResultContent) error {
			logger.InfoCF("agent", "Tool result received",
				map[string]any{
					"agent_id":     agent.ID,
					"tool":         tr.ToolName,
					"tool_call_id": tr.ToolCallID,
				})
			return nil
		},

		OnStepFinish: func(step fantasy.StepResult) error {
			mu.Lock()
			stepCount++
			currentStep := stepCount
			mu.Unlock()

			// Save step messages to session
			stepMessages := providers.FantasyStepToAgentXMessages(step)
			for _, msg := range stepMessages {
				agent.Sessions.AddFullMessage(opts.SessionKey, msg)
			}

			logger.DebugCF("agent", "Fantasy step finished",
				map[string]any{
					"agent_id": agent.ID,
					"step":     currentStep,
				})
			return nil
		},

		OnTextEnd: func(id string) error {
			// Signal stream done
			al.bus.PublishStreamDelta(bus.StreamDelta{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Done:    true,
			})
			return nil
		},
	})

	if err != nil {
		// Extract detailed error info from ProviderError
		var providerErr *fantasy.ProviderError
		if errors.As(err, &providerErr) {
			logger.ErrorCF("agent", "Fantasy provider error", map[string]any{
				"agent_id":    agent.ID,
				"status_code": providerErr.StatusCode,
				"title":       providerErr.Title,
				"message":     providerErr.Message,
				"response":    string(providerErr.ResponseBody),
			})
		}

		// Check for context/token errors and attempt compression
		errMsg := strings.ToLower(err.Error())
		isContextError := strings.Contains(errMsg, "token") ||
			strings.Contains(errMsg, "context") ||
			strings.Contains(errMsg, "length")

		if isContextError || (providerErr != nil && providerErr.IsContextTooLarge()) {
			logger.WarnCF("agent", "Context window error, attempting compression",
				map[string]any{"error": err.Error()})

			al.forceCompression(agent, opts.SessionKey)
			newHistory := agent.Sessions.GetHistory(opts.SessionKey)
			newSummary := agent.Sessions.GetSummary(opts.SessionKey)
			newMessages := agent.ContextBuilder.BuildMessages(
				newHistory, newSummary, "",
				nil, opts.Channel, opts.ChatID,
			)

			// Retry once after compression
			return al.runFantasyIteration(ctx, agent, newMessages, opts)
		}

		return "", stepCount, fmt.Errorf("fantasy agent failed: %w", err)
	}

	// Extract final text from result
	finalContent := ""
	if result != nil {
		finalContent = result.Response.Content.Text()
	}

	// If streaming buffer has content but result doesn't, use buffer
	mu.Lock()
	if finalContent == "" && textBuf.Len() > 0 {
		finalContent = textBuf.String()
	}
	mu.Unlock()

	if finalContent != "" {
		logger.InfoCF("agent", fmt.Sprintf("Response: %s", utils.Truncate(finalContent, 120)),
			map[string]any{
				"agent_id": agent.ID,
				"steps":    stepCount,
			})
	}

	return finalContent, stepCount, nil
}

// summarizeWithFantasy uses the Fantasy model directly for summarization.
func (al *AgentLoop) summarizeWithFantasy(
	ctx context.Context,
	model fantasy.LanguageModel,
	prompt string,
) (string, error) {
	fantasyMessages := []fantasy.Message{
		fantasy.NewUserMessage(prompt),
	}

	maxTokens := int64(1024)
	temperature := 0.3

	resp, err := model.Generate(ctx, fantasy.Call{
		Prompt:          fantasyMessages,
		MaxOutputTokens: &maxTokens,
		Temperature:     &temperature,
	})
	if err != nil {
		return "", err
	}

	return resp.Content.Text(), nil
}

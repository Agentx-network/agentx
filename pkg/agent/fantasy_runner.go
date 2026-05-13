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

// toolResultStoredLimit caps how many bytes of a tool's output are kept in
// session history. Large outputs (file reads, search results) are still seen
// in full by the model on the turn that produced them; this limit prevents
// them from bloating EVERY subsequent turn's prompt.
const toolResultStoredLimit = 2000

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
			fields := map[string]any{
				"agent_id":     agent.ID,
				"tool":         tr.ToolName,
				"tool_call_id": tr.ToolCallID,
			}
			// Surface the tool's actual output so users can see WHY the agent
			// did or didn't proceed. Truncated to keep logs readable.
			if tr.Result != nil {
				if textResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](tr.Result); ok {
					fields["result"] = utils.Truncate(textResult.Text, 300)
				} else if errResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](tr.Result); ok {
					if errResult.Error != nil {
						fields["error"] = errResult.Error.Error()
					}
				}
			}
			logger.InfoCF("agent", "Tool result received", fields)
			return nil
		},

		OnStepFinish: func(step fantasy.StepResult) error {
			mu.Lock()
			stepCount++
			currentStep := stepCount
			mu.Unlock()

			// Save step messages to session, with tool result content truncated.
			// Verbatim tool outputs (file contents, search results, etc.) can
			// be tens of KB; replaying them on every subsequent turn balloons
			// the prompt and burns tokens. Truncate aggressively here — the
			// agent already saw the full output in the turn that produced it.
			stepMessages := providers.FantasyStepToAgentXMessages(step)
			for _, msg := range stepMessages {
				if msg.Role == "tool" && len(msg.Content) > toolResultStoredLimit {
					msg.Content = msg.Content[:toolResultStoredLimit] +
						fmt.Sprintf("\n…[tool output truncated: %d bytes total]", len(msg.Content))
				}
				// Skip exact-duplicate consecutive messages. Some models (notably
				// gpt-oss-120b on Groq) emit the same assistant text across two
				// adjacent steps, which would otherwise be persisted twice and
				// re-sent on every future turn — burning tokens for nothing.
				history := agent.Sessions.GetHistory(opts.SessionKey)
				if n := len(history); n > 0 {
					last := history[n-1]
					if last.Role == msg.Role && last.Content == msg.Content &&
						len(msg.Content) > 0 {
						continue
					}
				}
				agent.Sessions.AddFullMessage(opts.SessionKey, msg)
			}

			logger.DebugCF("agent", "Fantasy step finished",
				map[string]any{
					"agent_id":      agent.ID,
					"step":          currentStep,
					"input_tokens":  step.Usage.InputTokens,
					"output_tokens": step.Usage.OutputTokens,
					"total_tokens":  step.Usage.TotalTokens,
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

		// Decide whether this is a genuine context-window overflow.
		// We trust the SDK's own IsContextTooLarge() check — substring matches
		// on "token"/"context"/"length" false-positive on rate-limit errors
		// like "tokens per minute (TPM)", which previously caused the agent
		// to compress (destroying session history) on every TPM hit.
		isContextOverflow := providerErr != nil && providerErr.IsContextTooLarge()

		// If the centralized classifier identifies this as anything other than
		// a context overflow (rate limit, auth, billing, timeout, etc.), bubble
		// the error up instead of attempting compression.
		if !isContextOverflow {
			if classified := providers.ClassifyError(err, agent.ID, agent.Model); classified != nil {
				logger.WarnCF("agent", "Provider error classified, no compression",
					map[string]any{
						"reason":   string(classified.Reason),
						"agent_id": agent.ID,
					})
				return "", stepCount, fmt.Errorf("fantasy agent failed: %w", err)
			}
		}

		if isContextOverflow {
			// Bound the retry: one compression attempt, then give up.
			if opts.CompressionRetried {
				return "", stepCount, fmt.Errorf("fantasy agent failed after compression retry: %w", err)
			}

			logger.WarnCF("agent", "Context window error, attempting compression",
				map[string]any{"error": err.Error()})

			al.forceCompression(agent, opts.SessionKey)
			newHistory := agent.Sessions.GetHistory(opts.SessionKey)
			newSummary := agent.Sessions.GetSummary(opts.SessionKey)
			newMessages := agent.ContextBuilder.BuildMessages(
				newHistory, newSummary, "",
				nil, opts.Channel, opts.ChatID,
			)

			opts.CompressionRetried = true
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

	// Some models (notably gpt-oss-120b on Groq) emit the final answer twice
	// in succession, producing "<reply><reply>" output. Strip an exact-half
	// duplicate when we see one — conservative heuristic, only triggers on
	// even-length strings where both halves match byte-for-byte.
	finalContent = stripExactDuplicate(finalContent)

	// Emit a dedicated Usage log line (the Response line is logged once in
	// the outer loop). Keeps token data visible without duplicating the text.
	if result != nil {
		fields := map[string]any{
			"agent_id":      agent.ID,
			"steps":         stepCount,
			"input_tokens":  result.TotalUsage.InputTokens,
			"output_tokens": result.TotalUsage.OutputTokens,
			"total_tokens":  result.TotalUsage.TotalTokens,
		}
		if result.TotalUsage.CacheReadTokens > 0 {
			fields["cache_read_tokens"] = result.TotalUsage.CacheReadTokens
		}
		logger.InfoCF("agent", "Usage", fields)
	}

	return finalContent, stepCount, nil
}

// stripExactDuplicate removes a trailing exact-duplicate half if the string
// is "XY" where X == Y. Guards against models that emit their reply twice.
func stripExactDuplicate(s string) string {
	n := len(s)
	if n < 20 || n%2 != 0 {
		return s
	}
	half := n / 2
	if s[:half] == s[half:] {
		return s[:half]
	}
	return s
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

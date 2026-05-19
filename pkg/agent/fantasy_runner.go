package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/providers"
	"github.com/Agentx-network/agentx/pkg/tools"
	"github.com/Agentx-network/agentx/pkg/utils"
)

var toolCallJSONLine = regexp.MustCompile(
	`(?m)^[ \t]*\{[ \t]*"name"[ \t]*:[ \t]*"[^"]+"[^\n]*"arguments"[^\n]*$`,
)

var blankRunCollapse = regexp.MustCompile(`\n{3,}`)

func stripToolCallJSON(s string) string {
	cleaned := toolCallJSONLine.ReplaceAllString(s, "")
	cleaned = blankRunCollapse.ReplaceAllString(cleaned, "\n\n")
	return strings.TrimSpace(cleaned)
}

// toolResultStoredLimit caps how many bytes of a tool's output are kept in
// session history. Large outputs (file reads, search results) are still seen
// in full by the model on the turn that produced them; this limit prevents
// them from bloating EVERY subsequent turn's prompt.
const toolResultStoredLimit = 2000

func looksLikeToolCallJSONStream(buf string) bool {
	s := strings.TrimLeft(buf, " \t\n\r")
	if s == "" {
		return false
	}
	return s[0] == '{' || s[0] == '['
}

// retryAfterPattern picks the wait duration out of a provider rate-limit
// error message. Matches both Gemini's wording ("Please retry in 22.5s")
// and Groq's variant ("Retry-After: 11", "retry-after: 11"). Returns 0 if
// no number can be extracted — caller falls back to a default.
var retryAfterPattern = regexp.MustCompile(
	`(?i)(?:retry[ -]?after[:\s]*|retry in[ \t]*)(\d+(?:\.\d+)?)\s*s?`,
)

// parseRetryAfter reads an error message for the provider's suggested
// retry delay and returns it bounded to [2s, 30s]. Long waits during a chat
// would feel broken, so we cap and let the user retry manually if even 30s
// isn't enough. Defaults to 8s when nothing parseable is found — short
// enough to feel like a one-time glitch, long enough that Gemini's per-
// minute window has usually rolled over.
func parseRetryAfter(errMsg string) time.Duration {
	const (
		fallback = 8 * time.Second
		minWait  = 2 * time.Second
		maxWait  = 30 * time.Second
	)
	m := retryAfterPattern.FindStringSubmatch(errMsg)
	if len(m) < 2 {
		return fallback
	}
	secs, err := strconv.ParseFloat(m[1], 64)
	if err != nil || secs <= 0 {
		return fallback
	}
	d := time.Duration(secs * float64(time.Second))
	if d < minWait {
		return minWait
	}
	if d > maxWait {
		return maxWait
	}
	return d
}

// looksCompleteToolCall returns true when the buffer parses as a full
// tool-call payload (parseTextToolCall succeeds). Used to distinguish
// "model gave us a real but unrecognized tool name" from "model cut off
// mid-emit" — the user-facing error message differs between those cases.
func looksCompleteToolCall(buf string) bool {
	_, _, ok := parseTextToolCall(buf)
	return ok
}

// isCerebrasContextOverflow returns true when the error message looks like
// Cerebras's "context_length_exceeded" wire format. We match on the
// machine-stable code (or its message phrasing) rather than generic words
// like "length" or "tokens", which would trip on TPM rate-limit errors and
// destructively trigger compression for unrelated failures.
//
// Patterns observed in field reports (May 2026):
//
//	"code":"context_length_exceeded"
//	"Please reduce the length of the messages or completion"
//
// Both come back as 400 / invalid_request_error from /chat/completions.
func isCerebrasContextOverflow(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "Please reduce the length of the messages")
}

// parseTextToolCall tries to interpret the model's text output as a tool-call
// payload that should have been emitted through the structured tool_calls
// channel. Some hosted Llama deployments (Cerebras, occasionally Groq) print
// the call as JSON text instead, which the Fantasy SDK ignores. Without
// recovery the user sees a "tried to use a tool but..." defensive message
// even though the agent's intent was unambiguous.
//
// Accepts either form Cerebras emits in practice:
//
//	{"name":"exec","arguments":{"command":"agentx wallet balance"}}      ← object
//	{"name":"exec","arguments":"{\"command\":\"agentx wallet balance\"}"} ← stringified
//
// Returns ok=false when the text isn't a single complete tool-call object —
// we deliberately don't try to extract from prose or multi-call payloads.
func parseTextToolCall(s string) (name string, args map[string]any, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return "", nil, false
	}
	if !strings.Contains(s, `"name"`) || !strings.Contains(s, `"arguments"`) {
		return "", nil, false
	}

	var parsed struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return "", nil, false
	}
	if parsed.Name == "" {
		return "", nil, false
	}

	args = map[string]any{}
	if len(parsed.Arguments) > 0 {
		// Try as object first.
		if err := json.Unmarshal(parsed.Arguments, &args); err != nil {
			// Some models double-encode arguments as a JSON string.
			var argStr string
			if err2 := json.Unmarshal(parsed.Arguments, &argStr); err2 == nil {
				_ = json.Unmarshal([]byte(argStr), &args)
			}
		}
	}
	return parsed.Name, args, true
}

// formatToolResultAsReply turns the raw output of a recovery-path tool call
// into a user-friendly final reply. The model would normally do this framing
// in a follow-up turn, but on recovery we have no second turn — we just show
// the data. Special-cases the wallet balance shape because that's the most
// common demo path; falls back to a JSON code block for everything else.
func formatToolResultAsReply(toolName, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Wallet-balance shape: array of {symbol, balance, ...}. The exec tool is
	// how `agentx wallet balance` runs, so this is the wire format users hit.
	if strings.HasPrefix(raw, "[") && strings.Contains(raw, `"symbol"`) && strings.Contains(raw, `"balance"`) {
		var balances []struct {
			Symbol  string `json:"symbol"`
			Balance string `json:"balance"`
		}
		if err := json.Unmarshal([]byte(raw), &balances); err == nil && len(balances) > 0 {
			var sb strings.Builder
			sb.WriteString("Here are your wallet balances:\n")
			for _, b := range balances {
				if b.Symbol == "" {
					continue
				}
				fmt.Fprintf(&sb, "- **%s**: %s\n", b.Symbol, b.Balance)
			}
			return strings.TrimRight(sb.String(), "\n")
		}
	}

	// Wallet-info shape: {address, chain, createdAt}
	if strings.HasPrefix(raw, "{") && strings.Contains(raw, `"address"`) && strings.Contains(raw, `"chain"`) {
		var info struct {
			Address string `json:"address"`
			Chain   string `json:"chain"`
		}
		if err := json.Unmarshal([]byte(raw), &info); err == nil && info.Address != "" {
			return fmt.Sprintf("Your wallet:\n- **Address**: `%s`\n- **Chain**: %s", info.Address, info.Chain)
		}
	}

	// Generic structured output — wrap in a fenced block.
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		return "Here's the result:\n```json\n" + raw + "\n```"
	}

	// Plain text — just pass through.
	return raw
}

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

	fantasyAgent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithTools(fantasyTools...),
		fantasy.WithMaxOutputTokens(maxTokens),
		fantasy.WithTemperature(temperature),
		fantasy.WithStopConditions(fantasy.StepCountIs(agent.MaxIterations)),
	)

	// Find the most recent assistant message in history. Side-effecting tools
	// use this to recognize the "user said yes, install what we just offered"
	// pattern as legitimate consent.
	var lastAssistantMsg string
	if hist := agent.Sessions.GetHistory(opts.SessionKey); len(hist) > 0 {
		for i := len(hist) - 1; i >= 0; i-- {
			if hist[i].Role == "assistant" && hist[i].Content != "" {
				lastAssistantMsg = hist[i].Content
				break
			}
		}
	}

	// Set up tool context in the context. UserMessage = the verbatim text of
	// this turn's user prompt, used by side-effecting tools (install_skill in
	// particular) to verify the user actually named the resource being acted
	// on. Without it, weak LLMs auto-install skills the user never approved.
	ctx = tools.WithToolContext(ctx, tools.ToolContext{
		Channel:              opts.Channel,
		ChatID:               opts.ChatID,
		UserMessage:          opts.UserMessage,
		LastAssistantMessage: lastAssistantMsg,
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
			accumulated := textBuf.String()
			mu.Unlock()

			// Hide tool-call JSON from the streamed view. Weak models that emit
			// the call as text (Cerebras Llama, some Groq deployments) otherwise
			// flash raw {"name":"exec","arguments":{...}} in the chat bubble for
			// a moment before our recovery replaces it with a formatted result.
			// We detect the prefix once enough characters have arrived to be
			// confident — short replies that legitimately start with `{` (rare
			// in chat) get a brief delay rather than being hidden permanently,
			// since the suppression only lasts until the model finishes and the
			// final formatted content is published as a normal message.
			if looksLikeToolCallJSONStream(accumulated) {
				return nil
			}

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
		//
		// Fallback: the SDK doesn't recognize every provider's wire format.
		// Cerebras returns {"code":"context_length_exceeded","message":"Please
		// reduce the length of the messages or completion..."} which slips
		// past IsContextTooLarge and surfaces as a 400 the user can't recover
		// from. We add a narrow substring check that matches only this
		// specific shape — distinct enough from TPM rate-limit text to be
		// safe from the false-positive we were defending against above.
		isContextOverflow := providerErr != nil && providerErr.IsContextTooLarge()
		if !isContextOverflow && isCerebrasContextOverflow(err) {
			isContextOverflow = true
			logger.InfoCF("agent", "Detected Cerebras-style context overflow via fallback pattern",
				map[string]any{"agent_id": agent.ID})
		}

		// If the centralized classifier identifies this as anything other than
		// a context overflow (rate limit, auth, billing, timeout, etc.), bubble
		// the error up instead of attempting compression.
		if !isContextOverflow {
			if classified := providers.ClassifyError(err, agent.ID, agent.Model); classified != nil {
				// Rate-limit auto-retry. Gemini free tier (20 req/min) and
				// other providers' free quotas reset within seconds, and the
				// error body often carries the retry-after delay verbatim
				// (e.g. "Please retry in 22.546079432s"). Wait that long once
				// instead of bouncing back to the user — demos read awful when
				// every other message says "rate limited, try again."
				if classified.Reason == providers.FailoverRateLimit && !opts.RateLimitRetried {
					wait := parseRetryAfter(err.Error())
					logger.WarnCF("agent", "Rate-limited, auto-retrying after wait",
						map[string]any{
							"agent_id": agent.ID,
							"wait":     wait.String(),
						})
					select {
					case <-time.After(wait):
					case <-ctx.Done():
						return "", stepCount, ctx.Err()
					}
					opts.RateLimitRetried = true
					return al.runFantasyIteration(ctx, agent, messages, opts)
				}

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

	// Remove raw tool-call JSON that some models echo as text content. The
	// actual tool execution still happens via the SDK's structured tool_call
	// channel; this just hides the duplicate JSON echo from the chat UI.
	rawBeforeStrip := finalContent
	finalContent = stripToolCallJSON(finalContent)

	// Catch incomplete tool-call JSON that the regex above missed. Models
	// sometimes stop emitting mid-payload — e.g. `{"name": "` and nothing
	// further — when they hit a token limit or just bail. The regex needs
	// both `"name"` and `"arguments"` to match, so partial calls slip past
	// and surface as raw JSON in the chat bubble. Detect that shape here
	// and clear finalContent so the recovery / defensive-message branch
	// below handles it cleanly.
	if looksLikeToolCallJSONStream(finalContent) {
		logger.WarnCF("agent", "Stripping incomplete tool-call JSON from final content",
			map[string]any{"agent_id": agent.ID, "remnant": utils.Truncate(finalContent, 120)})
		finalContent = ""
	}

	// If stripping erased the entire reply, the model produced ONLY a raw
	// tool-call JSON line (no structured tool_call and no surrounding text).
	// The Fantasy SDK ignored that text — but the model's intent is clear, so
	// recover by parsing the JSON and executing the tool ourselves. This makes
	// Cerebras-hosted Llama 3.x and similarly-misbehaving deployments usable
	// for tool queries (wallet, web search, file ops) instead of bouncing the
	// user with a "switch your model" message.
	if finalContent == "" && strings.TrimSpace(rawBeforeStrip) != "" {
		if name, args, parseOK := parseTextToolCall(rawBeforeStrip); parseOK && agent.Tools != nil {
			if _, exists := agent.Tools.Get(name); exists {
				logger.InfoCF("agent", "Recovering text-form tool call",
					map[string]any{"agent_id": agent.ID, "tool": name})

				toolRes := agent.Tools.Execute(ctx, name, args)
				if toolRes != nil {
					switch {
					case toolRes.IsError:
						// Prefer the tool's user-facing message when set.
						// Tools that use BlockedResult provide a clean line
						// meant for direct display (e.g. "Which skill would
						// you like me to install?"). Falling back to ForLLM
						// would surface internal model-guidance text like
						// "BLOCKED: User MUST NOT install ..." into the chat,
						// which is what triggered the field reports of
						// "internal instructions leaking to users."
						if toolRes.ForUser != "" {
							finalContent = toolRes.ForUser
						} else {
							errMsg := strings.TrimSpace(toolRes.ForLLM)
							if errMsg == "" && toolRes.Err != nil {
								errMsg = toolRes.Err.Error()
							}
							finalContent = fmt.Sprintf("I tried to run `%s` but it failed: %s", name, errMsg)
						}
					default:
						// Always run through the formatter so known shapes (wallet
						// balance, wallet info) get pretty-printed. The exec tool
						// sets ForUser=ForLLM=raw stdout, so reading ForUser first
						// — as we used to — short-circuited the formatter and
						// dumped raw JSON into the chat. ForLLM is preferred;
						// ForUser only wins when the tool didn't set ForLLM.
						raw := toolRes.ForLLM
						if raw == "" {
							raw = toolRes.ForUser
						}
						if raw != "" {
							finalContent = formatToolResultAsReply(name, raw)
						}
					}
				}
			} else {
				logger.WarnCF("agent", "Text-form tool call references unknown tool",
					map[string]any{"agent_id": agent.ID, "tool": name})
			}
		}

		// If recovery didn't yield anything, surface a clear explanation so
		// users understand what went wrong instead of seeing silence (or raw
		// JSON fragments). Distinguish "incomplete payload" (model cut off
		// mid-call) from "tool unsupported" (model never emitted a valid
		// call) — the former is recoverable by retrying, the latter requires
		// switching models.
		if finalContent == "" {
			isIncomplete := looksLikeToolCallJSONStream(rawBeforeStrip) && !looksCompleteToolCall(rawBeforeStrip)
			logger.WarnCF("agent", "Tool-call recovery failed",
				map[string]any{
					"agent_id":   agent.ID,
					"raw_len":    len(rawBeforeStrip),
					"incomplete": isIncomplete,
				})
			if isIncomplete {
				finalContent = "The model started using a tool but stopped before finishing — its reply got cut off. " +
					"Please try asking again. If this keeps happening, switch to a more capable model in Settings → Config (e.g. claude-sonnet-4, gemini-2.5-flash, or llama-3.3-70b)."
			} else {
				finalContent = "I tried to use a tool but the response came back in a format the system can't execute. " +
					"This usually means the selected model doesn't reliably support function calling. " +
					"Try a different model in Settings → Config (e.g. claude-sonnet-4, llama-3.3-70b, or gpt-5.2)."
			}
		}
	}

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

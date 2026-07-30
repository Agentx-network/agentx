package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/Agentx-network/agentx/pkg/attach"
	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/providers"
	"github.com/Agentx-network/agentx/pkg/tools"
	"github.com/Agentx-network/agentx/pkg/utils"
)

// Matches a line that's a text-form tool call in either shape:
//
//	{"name":"x","arguments":...}  or  {"type":"function","name":"x","parameters":...}
var toolCallJSONLine = regexp.MustCompile(
	`(?m)^[ \t]*\{[^\n]*"name"[ \t]*:[ \t]*"[^"]+"[^\n]*"(?:arguments|parameters)"[^\n]*$`,
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

// retryAfterPattern matches "Please retry in Xs" (Gemini) and "Retry-After: N"
// (Groq) wordings in rate-limit error messages.
var retryAfterPattern = regexp.MustCompile(
	`(?i)(?:retry[ -]?after[:\s]*|retry in[ \t]*)(\d+(?:\.\d+)?)\s*s?`,
)

// parseRetryAfter returns the suggested retry delay from a rate-limit error,
// bounded to [2s, 30s]. Defaults to 8s when nothing parseable is found.
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

// looksCompleteToolCall reports whether buf parses as a full tool-call payload.
// Used to distinguish "unknown tool" from "model cut off mid-emit".
func looksCompleteToolCall(buf string) bool {
	_, _, ok := parseTextToolCall(buf)
	return ok
}

// resolveToolName maps a model-mangled tool name to a registered one. Exact
// match wins; otherwise it compares with separators/case stripped, so
// "webfetch"/"WebFetch" resolve to "web_fetch". Returns the input unchanged
// when no match is found (caller then reports "unknown tool").
func resolveToolName(reg *tools.ToolRegistry, name string) string {
	if reg == nil || name == "" {
		return name
	}
	if _, ok := reg.Get(name); ok {
		return name
	}
	norm := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, "-", "")
		return strings.ReplaceAll(s, " ", "")
	}
	want := norm(name)
	for _, registered := range reg.List() {
		if norm(registered) == want {
			return registered
		}
	}
	return name
}

// textRecoverableTools is the allowlist of tools the text-form recovery path
// may auto-execute (H4, audit). A tool call emitted as plain text is a weak
// signal of intent — a model may simply be echoing user-pasted content — so we
// only run low-risk, mostly read-only tools here. Anything that runs commands or
// installs code (exec, install_skill, spawn, cron, hardware) must arrive as a
// genuine structured tool call from the provider, never via text recovery.
var textRecoverableTools = map[string]bool{
	"web_search":               true,
	"web_fetch":                true,
	"find_skills":              true,
	"image_generate":           true,
	"configure_image_provider": true,
	"message":                  true,
}

func textRecoverableTool(name string) bool {
	return textRecoverableTools[name]
}

// isEchoedToolCall reports whether the raw text-form tool call appears verbatim
// in a recent user message — i.e. the model is parroting attacker- or
// user-pasted JSON rather than deciding to call a tool. Whitespace is
// normalized so formatting differences don't defeat the check. (H4, audit.)
func isEchoedToolCall(messages []providers.Message, raw string) bool {
	// Strip ALL whitespace so reformatting (extra spaces, line breaks) can't
	// defeat the comparison; both sides are stripped equally.
	norm := func(s string) string { return strings.Join(strings.Fields(s), "") }
	needle := norm(raw)
	if needle == "" {
		return false
	}
	for _, m := range messages {
		if m.Role == "user" && strings.Contains(norm(m.Content), needle) {
			return true
		}
	}
	return false
}

// isCerebrasContextOverflow matches Cerebras's "context_length_exceeded" code
// (or its message phrasing). Narrower than generic "length"/"tokens" matching,
// which would false-positive on TPM rate-limit errors.
func isCerebrasContextOverflow(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "Please reduce the length of the messages")
}

// parseTextToolCall parses a tool-call payload a model printed as text instead
// of using the SDK's tool_calls channel. Handles both shapes seen in the wild:
//
//	{"name":"web_search","arguments":{...}}                    ← Cerebras Llama
//	{"type":"function","name":"web_search","parameters":{...}} ← OpenAI-wrapper form
//
// Args may be an object or a stringified JSON. Returns ok=false on prose.
func parseTextToolCall(s string) (name string, args map[string]any, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return "", nil, false
	}
	if !strings.Contains(s, `"name"`) ||
		(!strings.Contains(s, `"arguments"`) && !strings.Contains(s, `"parameters"`)) {
		return "", nil, false
	}

	var parsed struct {
		Name       string          `json:"name"`
		Arguments  json.RawMessage `json:"arguments"`
		Parameters json.RawMessage `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return "", nil, false
	}
	if parsed.Name == "" {
		return "", nil, false
	}

	rawArgs := parsed.Arguments
	if len(rawArgs) == 0 {
		rawArgs = parsed.Parameters // OpenAI-wrapper form
	}
	args = map[string]any{}
	if len(rawArgs) > 0 {
		// Try as object first; some models double-encode as a JSON string.
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			var argStr string
			if err2 := json.Unmarshal(rawArgs, &argStr); err2 == nil {
				_ = json.Unmarshal([]byte(argStr), &args)
			}
		}
	}
	return parsed.Name, args, true
}

// formatToolResultAsReply formats a tool result for direct display when the
// recovery path has no second model turn to do the framing. Special-cases
// wallet balance + info; falls back to a JSON code block otherwise.
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

	// Strip the machine-readable attachment display block — it's persisted in
	// history for the UI to re-render, but the model should see clean text
	// (images/audio arrive as FileParts below; docs via the read_file note).
	prompt = attach.StripDisplayBlock(prompt)

	fantasyMessages := providers.AgentXToFantasyMessages(historyMessages)

	// Build image/audio attachments for THIS turn as Fantasy FileParts. The
	// capability guard skips modalities the resolved model can't accept and
	// returns a note we fold into the prompt so the model tells the user
	// (accept-and-warn, enforced in code rather than prompt-only).
	fileParts, attachNote := al.buildAttachmentFileParts(model, opts.MediaFiles)
	if attachNote != "" {
		prompt += attachNote
	}

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

	// Last assistant message + current user message are surfaced via ToolContext
	// so side-effecting tools (install_skill) can verify consent.
	var lastAssistantMsg string
	if hist := agent.Sessions.GetHistory(opts.SessionKey); len(hist) > 0 {
		for i := len(hist) - 1; i >= 0; i-- {
			if hist[i].Role == "assistant" && hist[i].Content != "" {
				lastAssistantMsg = hist[i].Content
				break
			}
		}
	}
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
		Files:    fileParts,
		Messages: fantasyMessages,

		OnTextDelta: func(id, text string) error {
			mu.Lock()
			textBuf.WriteString(text)
			accumulated := textBuf.String()
			mu.Unlock()

			// Suppress streaming when the buffer looks like a raw tool-call JSON
			// prefix — recovery replaces it with formatted text in the final SSE
			// `done` event. Stops raw {"name":"exec",...} from flashing in chat.
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

		// Trust the SDK's IsContextTooLarge; fall back to a Cerebras-specific
		// pattern since the SDK doesn't recognize that wire format.
		isContextOverflow := providerErr != nil && providerErr.IsContextTooLarge()
		if !isContextOverflow && isCerebrasContextOverflow(err) {
			isContextOverflow = true
			logger.InfoCF("agent", "Detected Cerebras-style context overflow via fallback pattern",
				map[string]any{"agent_id": agent.ID})
		}

		if !isContextOverflow {
			if classified := providers.ClassifyError(err, agent.ID, agent.Model); classified != nil {
				// Rate-limit auto-retry: wait the provider's suggested delay once
				// before bubbling. Gemini free tier's per-minute window usually
				// resets within ~25s.
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

	// Strip duplicated halves (gpt-oss-120b quirk) and any raw tool-call JSON
	// some models echo as text alongside the structured tool_call.
	finalContent = stripExactDuplicate(finalContent)
	rawBeforeStrip := finalContent
	finalContent = stripToolCallJSON(finalContent)

	// Catch incomplete tool-call JSON the regex missed (model cut off mid-emit).
	if looksLikeToolCallJSONStream(finalContent) {
		logger.WarnCF("agent", "Stripping incomplete tool-call JSON from final content",
			map[string]any{"agent_id": agent.ID, "remnant": utils.Truncate(finalContent, 120)})
		finalContent = ""
	}

	// Recovery: stripping erased the reply, so the model emitted ONLY a text-form
	// tool call. Parse + execute it ourselves so Cerebras-hosted Llamas work for
	// tool queries instead of bouncing the user with "switch your model".
	if finalContent == "" && strings.TrimSpace(rawBeforeStrip) != "" {
		if name, args, parseOK := parseTextToolCall(rawBeforeStrip); parseOK && agent.Tools != nil {
			// Weak models also mangle tool names ("webfetch" → "web_fetch").
			// Resolve to a registered name before lookup.
			name = resolveToolName(agent.Tools, name)
			_, exists := agent.Tools.Get(name)

			// H4 (audit): hard-gate the recovery path before executing anything.
			// Refuse tools not on the low-risk allowlist (never exec/install via
			// text), and refuse calls echoed from the user's message (paste-back /
			// prompt injection). These can't be auto-run on a text signal alone.
			switch {
			case exists && !textRecoverableTool(name):
				logger.WarnCF("agent", "Refusing text-form recovery for non-allowlisted tool",
					map[string]any{"agent_id": agent.ID, "tool": name})
				exists = false
			case exists && isEchoedToolCall(messages, rawBeforeStrip):
				logger.WarnCF("agent", "Refusing text-form recovery: tool call echoed from user message",
					map[string]any{"agent_id": agent.ID, "tool": name})
				exists = false
			}

			if exists {
				logger.InfoCF("agent", "Recovering text-form tool call",
					map[string]any{"agent_id": agent.ID, "tool": name})

				toolRes := agent.Tools.Execute(ctx, name, args)
				if toolRes != nil {
					switch {
					case toolRes.IsError:
						// Prefer ForUser (BlockedResult) over ForLLM, which would
						// leak internal model-guidance text into chat.
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
						raw := toolRes.ForLLM
						if raw == "" {
							raw = toolRes.ForUser
						}
						switch {
						case raw == "":
							// nothing to show
						case !opts.ToolResultRetried:
							// Re-prompt the model with the result so it synthesizes a
							// real answer (e.g. read web_search snippets → "The current
							// CM is X") instead of us dumping raw tool output. Bounded
							// to one re-prompt; also set WebSearchRetried so the model
							// can't trigger another auto-search on this turn.
							logger.InfoCF("agent", "Re-prompting model with recovered tool result",
								map[string]any{"agent_id": agent.ID, "tool": name})
							augmented := append(messages, providers.Message{
								Role: "user",
								Content: fmt.Sprintf(
									"[SYSTEM: The %s tool was run for the question above. Result:]\n\n%s\n\n"+
										"Answer the user's original question directly using this result. Be concise and "+
										"conclusive — synthesize the key facts into a clear answer. Do NOT paste the raw "+
										"result or list of links, and do NOT call any more tools.",
									name, raw,
								),
							})
							opts.ToolResultRetried = true
							opts.WebSearchRetried = true
							return al.runFantasyIteration(ctx, agent, augmented, opts)
						default:
							// Already re-prompted once; show formatted raw as fallback.
							finalContent = formatToolResultAsReply(name, raw)
						}
					}
				}
			} else if _, registered := agent.Tools.Get(name); !registered {
				// Truly unknown tool (gated tools already logged their refusal above).
				logger.WarnCF("agent", "Text-form tool call references unknown tool",
					map[string]any{"agent_id": agent.ID, "tool": name})
			}
		}

		// Recovery exhausted: distinguish "model cut off mid-call" from "model
		// never emitted a valid call" so the user gets an actionable message.
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

	// Auto-web-search fallback. Weak models (Cerebras Llama) often NARRATE a
	// search ("I will use web_search... please wait") or deflect with "my
	// knowledge cutoff" instead of actually emitting the tool call. Detect that
	// and run the search ourselves, then re-prompt the model with the results
	// so it answers from real current data. Bounded to one retry.
	if !opts.WebSearchRetried && agent.Tools != nil && shouldAutoWebSearch(finalContent) {
		if _, ok := agent.Tools.Get("web_search"); ok && strings.TrimSpace(opts.UserMessage) != "" {
			logger.InfoCF("agent", "Auto web-search fallback triggered",
				map[string]any{"agent_id": agent.ID, "query": utils.Truncate(opts.UserMessage, 120)})

			searchRes := agent.Tools.Execute(ctx, "web_search", map[string]any{"query": opts.UserMessage})
			if searchRes != nil && !searchRes.IsError && strings.TrimSpace(searchRes.ForLLM) != "" {
				augmented := append(messages, providers.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"[SYSTEM: A web_search was already run for the question above. Results below.]\n\n%s\n\n"+
							"Answer the user's original question directly using ONLY these results. "+
							"State the current information plainly. Do NOT say you will search — the search is already done. "+
							"If the results don't contain the answer, say so honestly.",
						searchRes.ForLLM,
					),
				})
				opts.WebSearchRetried = true
				return al.runFantasyIteration(ctx, agent, augmented, opts)
			}
			logger.WarnCF("agent", "Auto web-search returned no usable results",
				map[string]any{"agent_id": agent.ID})
		}
	}

	// Auto-skill-search fallback. When the user asks for a capability the agent
	// doesn't have built-in, models frequently just deflect ("I can't do that, my
	// tools don't have that capability") and stop — instead of checking whether an
	// installable skill would add it. Detect that deflection, run find_skills
	// ourselves, and re-prompt the model with the results so it OFFERS to install a
	// matching skill rather than dead-ending the user. Bounded to one retry.
	if !opts.SkillSearchRetried && agent.Tools != nil && shouldAutoSkillSearch(finalContent) {
		if _, ok := agent.Tools.Get("find_skills"); ok && strings.TrimSpace(opts.UserMessage) != "" {
			logger.InfoCF("agent", "Auto skill-search fallback triggered",
				map[string]any{"agent_id": agent.ID, "query": utils.Truncate(opts.UserMessage, 120)})

			searchRes := agent.Tools.Execute(ctx, "find_skills", map[string]any{"query": opts.UserMessage})
			// Only re-prompt when the search actually surfaced candidate skills.
			// A blocked (vague query) or empty result would just make the model
			// apologize twice, so leave the original honest reply in place.
			if searchRes != nil && !searchRes.IsError &&
				strings.TrimSpace(searchRes.ForLLM) != "" &&
				!strings.Contains(searchRes.ForLLM, "No skills found") {
				augmented := append(messages, providers.Message{
					Role: "user",
					Content: fmt.Sprintf(
						"[SYSTEM: The user asked for a capability you said you can't do. A find_skills "+
							"search was already run to look for an installable skill that adds it. Results below.]\n\n%s\n\n"+
							"If one of these skills matches what the user needs, DON'T say you can't help — tell them "+
							"you found a skill that can do it and offer to install it (they must confirm first, then you "+
							"call install_skill with the exact slug and registry). If none genuinely match, tell the user "+
							"honestly that no suitable skill is available. Do NOT claim you already installed anything.",
						searchRes.ForLLM,
					),
				})
				opts.SkillSearchRetried = true
				return al.runFantasyIteration(ctx, agent, augmented, opts)
			}
			logger.InfoCF("agent", "Auto skill-search found no candidate skills",
				map[string]any{"agent_id": agent.ID})
		}
	}

	return finalContent, stepCount, nil
}

// shouldAutoSkillSearch reports whether the model's reply is a capability-gap
// deflection ("I can't do that, my tools don't have that capability") rather
// than a real answer — the signal to run find_skills ourselves and re-prompt so
// the agent can offer an installable skill. Matched case-insensitively. Kept
// narrow: bare refusals ("I can't help with that request") must NOT match, only
// tool/capability-shaped language where a skill could plausibly fill the gap.
func shouldAutoSkillSearch(response string) bool {
	r := strings.ToLower(response)
	triggers := []string{
		"tools don't have that capability", "tools don't have the capability",
		"don't have that capability", "don't have the capability",
		"my current tools don't", "my tools don't have",
		"i don't have a tool", "i don't have the tools",
		"i don't have a skill", "i don't have any skill",
		"i can't directly", "i cannot directly",
		"i'm not able to do that", "i am not able to do that",
		"not something i can do", "outside my current capabilities",
		"beyond my current capabilities", "outside my capabilities",
		"i lack the capability", "i'm not equipped to", "not equipped to",
		"i don't have the ability to", "i do not have the ability to",
	}
	for _, t := range triggers {
		if strings.Contains(r, t) {
			return true
		}
	}
	return false
}

// maxAttachmentBytes caps how large an image/audio file we'll load into memory
// to send to the model. Files are already size-checked at upload time; this is
// a defensive backstop.
const maxAttachmentBytes = 30 << 20 // 30 MB

// buildAttachmentFileParts converts image/audio attachment paths into Fantasy
// FileParts for the current turn, honoring the resolved model's capabilities.
// Modalities the model can't accept are skipped and summarized in the returned
// note (folded into the prompt so the model tells the user to switch models).
func (al *AgentLoop) buildAttachmentFileParts(model fantasy.LanguageModel, paths []string) ([]fantasy.FilePart, string) {
	if len(paths) == 0 {
		return nil, ""
	}
	provider, modelID := model.Provider(), model.Model()
	canVision := providers.ProviderSupportsVision(provider, modelID)
	canAudio := providers.ProviderSupportsAudioInput(provider, modelID)

	var parts []fantasy.FilePart
	var skippedImages, skippedAudio []string

	for _, p := range paths {
		kind := attach.Classify(p)
		// Only images/audio are ever sent to the model. Anything else (video,
		// docs) is handled by tools and must not be embedded as bytes.
		if kind != attach.KindImage && kind != attach.KindAudio {
			continue
		}
		if kind == attach.KindImage && !canVision {
			skippedImages = append(skippedImages, filepath.Base(p))
			continue
		}
		if kind == attach.KindAudio && !canAudio {
			skippedAudio = append(skippedAudio, filepath.Base(p))
			continue
		}

		info, err := os.Stat(p)
		if err != nil || info.Size() > maxAttachmentBytes {
			logger.WarnCF("agent", "Skipping unreadable/oversized attachment",
				map[string]any{"path": p, "error": fmt.Sprintf("%v", err)})
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			logger.WarnCF("agent", "Failed to read attachment", map[string]any{"path": p, "error": err.Error()})
			continue
		}
		parts = append(parts, fantasy.FilePart{
			Filename:  filepath.Base(p),
			Data:      data,
			MediaType: attach.MimeForPath(p, data),
		})
	}

	if len(parts) > 0 {
		logger.InfoCF("agent", "Attached files to model turn",
			map[string]any{"count": len(parts), "provider": provider, "model": modelID})
	}

	// Build an accept-and-warn note for anything we couldn't send.
	var note string
	if len(skippedImages) > 0 {
		note += fmt.Sprintf("\n\n[SYSTEM: The user attached %d image(s) (%s) but the current model (%s) can't view images. "+
			"Tell the user their current model can't see images and to switch to a vision-capable model "+
			"(e.g. Gemini or GPT-4o) in Config to analyze it. Do NOT pretend to see the image.]",
			len(skippedImages), strings.Join(skippedImages, ", "), modelID)
	}
	if len(skippedAudio) > 0 {
		note += fmt.Sprintf("\n\n[SYSTEM: The user attached %d audio file(s) (%s) but the current model (%s) can't process audio. "+
			"Tell the user to switch to an audio-capable model (e.g. Gemini). Do NOT pretend to hear the audio.]",
			len(skippedAudio), strings.Join(skippedAudio, ", "), modelID)
	}
	return parts, note
}

// shouldAutoWebSearch reports whether the model's reply is a search-narration
// or knowledge-cutoff deflection rather than a real answer — the signal to run
// a web search ourselves and re-prompt. Matched case-insensitively.
func shouldAutoWebSearch(response string) bool {
	r := strings.ToLower(response)
	triggers := []string{
		"i will use the web_search", "i'll use the web_search",
		"i will use web_search", "use the web_search tool",
		"please wait while i search", "while i search for the latest",
		"let me search the web", "i will search the web", "i'll search the web",
		"my knowledge cutoff", "my training data only goes",
		"knowledge cutoff is", "as of my last update", "as of my knowledge cutoff",
		"i don't have access to real-time", "i don't have access to current",
		"i cannot access the internet", "i'm not able to browse",
		"couldn't find any information on the current",
		"my search results are outdated",
	}
	for _, t := range triggers {
		if strings.Contains(r, t) {
			return true
		}
	}
	return false
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

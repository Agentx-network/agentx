package tools

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"charm.land/fantasy"

	"github.com/Agentx-network/agentx/pkg/logger"
)

// mangledSurrogateRe matches a UTF-16 surrogate pair that the fantasy SDK
// (v0.11.0) corrupts while streaming tool-call arguments: it turns the "\u" of
// a "💧"-style escape into a newline, so an emoji like 💧 arrives in
// the parsed argument as the literal text "d83d" + "dca7" separated by
// whitespace. The hex halves are constrained to the high (D800–DBFF) and low
// (DC00–DFFF) surrogate ranges, so this can only ever match a corrupted emoji,
// never ordinary text.
// Each corrupted "\u" becomes a single whitespace char, so match exactly one
// separator before each surrogate half (consuming the inserted whitespace
// without eating a legitimate preceding space).
var mangledSurrogateRe = regexp.MustCompile(`\s([dD][89abAB][0-9a-fA-F]{2})\s([dD][c-fC-F][0-9a-fA-F]{2})`)

// repairMangledSurrogates reconstructs emojis corrupted by the fantasy SDK's
// tool-argument streaming (see mangledSurrogateRe). It only rewrites text that
// matches a valid high+low surrogate pair, leaving everything else untouched.
func repairMangledSurrogates(s string) string {
	if !strings.ContainsAny(s, "dD") {
		return s
	}
	return mangledSurrogateRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mangledSurrogateRe.FindStringSubmatch(m)
		hi, err1 := strconv.ParseUint(sub[1], 16, 32)
		lo, err2 := strconv.ParseUint(sub[2], 16, 32)
		if err1 != nil || err2 != nil {
			return m
		}
		return string(rune(0x10000 + (hi-0xD800)*0x400 + (lo - 0xDC00)))
	})
}

// toolContextKey is the key type for storing ToolContext in context.Context.
type toolContextKey struct{}

// ToolContext is plumbed through context.Context for tools that need request
// scope. UserMessage and LastAssistantMessage are used by side-effecting tools
// (install_skill) to verify consent. Empty fields are treated as "no signal".
type ToolContext struct {
	Channel              string
	ChatID               string
	UserMessage          string
	LastAssistantMessage string
}

// WithToolContext adds ToolContext to a context.
func WithToolContext(ctx context.Context, tc ToolContext) context.Context {
	return context.WithValue(ctx, toolContextKey{}, tc)
}

// GetToolContext extracts ToolContext from a context.
func GetToolContext(ctx context.Context) (ToolContext, bool) {
	tc, ok := ctx.Value(toolContextKey{}).(ToolContext)
	return tc, ok
}

// FantasyToolAdapter wraps an AgentX Tool as a Fantasy AgentTool.
type FantasyToolAdapter struct {
	tool        Tool
	forUserSink func(string)
	parallel    bool
	provOpts    fantasy.ProviderOptions
}

// Info implements fantasy.AgentTool.
// Fantasy SDK expects ToolInfo.Parameters to be just the properties map,
// NOT the full JSON Schema. The SDK wraps it in {"type":"object","properties":...,"required":...}
// internally (see agent.go prepareTools). AgentX tools return the full schema from
// Parameters(), so we must extract the inner "properties" and "required" fields.
func (a *FantasyToolAdapter) Info() fantasy.ToolInfo {
	fullSchema := a.tool.Parameters()
	if fullSchema == nil {
		fullSchema = map[string]any{}
	}

	// Extract "properties" — this is what Fantasy SDK expects as Parameters
	properties, _ := fullSchema["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
	}

	// Extract "required" — this goes into ToolInfo.Required
	var required []string
	if reqRaw, ok := fullSchema["required"]; ok {
		switch r := reqRaw.(type) {
		case []string:
			required = r
		case []any:
			for _, v := range r {
				if s, ok := v.(string); ok {
					required = append(required, s)
				}
			}
		}
	}

	return fantasy.ToolInfo{
		Name:        a.tool.Name(),
		Description: a.tool.Description(),
		Parameters:  properties,
		Required:    required,
		Parallel:    a.parallel,
	}
}

// Run implements fantasy.AgentTool.
func (a *FantasyToolAdapter) Run(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Parse JSON input to map
	var args map[string]any
	if call.Input != "" {
		if err := json.Unmarshal([]byte(call.Input), &args); err != nil {
			return fantasy.NewTextErrorResponse("invalid parameters: " + err.Error()), nil
		}
	}
	if args == nil {
		args = map[string]any{}
	}

	// Repair emojis the fantasy SDK corrupted while streaming the arguments
	// (e.g. a reminder message "drink water 💧" arriving as "...d83d dca7...").
	for k, v := range args {
		if sv, ok := v.(string); ok {
			args[k] = repairMangledSurrogates(sv)
		}
	}

	// Set context on contextual tools
	if ct, ok := a.tool.(ContextualTool); ok {
		if tc, found := GetToolContext(ctx); found {
			ct.SetContext(tc.Channel, tc.ChatID)
		}
	}

	// Set async callback for async tools
	if at, ok := a.tool.(AsyncTool); ok {
		at.SetCallback(func(cbCtx context.Context, result *ToolResult) {
			if !result.Silent && result.ForUser != "" && a.forUserSink != nil {
				a.forUserSink(result.ForUser)
			}
		})
	}

	// Execute
	result := a.tool.Execute(ctx, args)

	// Send ForUser content through side channel
	if !result.Silent && result.ForUser != "" && a.forUserSink != nil {
		a.forUserSink(result.ForUser)
	}

	// Convert to Fantasy response
	content := result.ForLLM
	if content == "" && result.Err != nil {
		content = result.Err.Error()
	}

	if result.IsError {
		return fantasy.NewTextErrorResponse(content), nil
	}

	return fantasy.NewTextResponse(content), nil
}

// ProviderOptions implements fantasy.AgentTool.
func (a *FantasyToolAdapter) ProviderOptions() fantasy.ProviderOptions {
	return a.provOpts
}

// SetProviderOptions implements fantasy.AgentTool.
func (a *FantasyToolAdapter) SetProviderOptions(opts fantasy.ProviderOptions) {
	a.provOpts = opts
}

// parallelSafeTools lists tools that are safe to run in parallel.
var parallelSafeTools = map[string]bool{
	"read_file":   true,
	"write_file":  true,
	"list_dir":    true,
	"edit_file":   true,
	"append_file": true,
	"web_search":  true,
	"web_fetch":   true,
	"find_skills": true,
}

// AdaptToolsForFantasy converts an AgentX ToolRegistry to Fantasy AgentTools.
func AdaptToolsForFantasy(registry *ToolRegistry, forUserSink func(string)) []fantasy.AgentTool {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	sorted := registry.sortedToolNames()
	adapted := make([]fantasy.AgentTool, 0, len(sorted))

	for _, name := range sorted {
		tool := registry.tools[name]
		adapter := &FantasyToolAdapter{
			tool:        tool,
			forUserSink: forUserSink,
			parallel:    parallelSafeTools[name],
		}
		adapted = append(adapted, adapter)
		logger.DebugCF("tools", "Adapted tool for Fantasy",
			map[string]any{
				"tool":     name,
				"parallel": adapter.parallel,
			})
	}

	return adapted
}

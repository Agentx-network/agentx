package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Agentx-network/agentx/pkg/skills"
)

// FindSkillsTool allows the LLM agent to search for installable skills from registries.
type FindSkillsTool struct {
	registryMgr *skills.RegistryManager
	cache       *skills.SearchCache
}

// NewFindSkillsTool creates a new FindSkillsTool.
// registryMgr is the shared registry manager (built from config in createToolRegistry).
// cache is the search cache for deduplicating similar queries.
func NewFindSkillsTool(registryMgr *skills.RegistryManager, cache *skills.SearchCache) *FindSkillsTool {
	return &FindSkillsTool{
		registryMgr: registryMgr,
		cache:       cache,
	}
}

func (t *FindSkillsTool) Name() string {
	return "find_skills"
}

func (t *FindSkillsTool) Description() string {
	return "Search for installable skills from skill registries. Returns skill slugs, descriptions, versions, and relevance scores. Use this to discover skills before installing them with install_skill."
}

func (t *FindSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query describing the desired skill capability (e.g., 'github integration', 'database management')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (1-20, default 10)",
				"minimum":     1.0,
				"maximum":     20.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *FindSkillsTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	query = strings.ToLower(strings.TrimSpace(query))
	if !ok || query == "" {
		return ErrorResult("query is required and must be a non-empty string")
	}

	// Vague-query guard: rejects filler-only queries ("new skill", "any") so
	// the agent asks the user what they actually want instead of dumping noise.
	if isVagueSkillQuery(query) {
		return BlockedResult(
			"Sure! Which kind of skill would you like? "+
				"For example: web search, GitHub, file tools, marketplace, agent discovery, calendar, finance, …",
			"Query \""+query+"\" is too vague. "+
				"Ask the user what KIND of skill they want before calling find_skills. "+
				"Suggested reply: \"Sure! Which kind of skill would you like? "+
				"For example: web search, GitHub, file tools, marketplace, agent discovery, calendar, …\"",
		)
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		li := int(l)
		if li >= 1 && li <= 20 {
			limit = li
		}
	}

	// Check cache first.
	if t.cache != nil {
		if cached, hit := t.cache.Get(query); hit {
			return SilentResult(formatSearchResults(query, cached, true))
		}
	}

	// Search all registries. SearchBroadened retries with a stemmed/stopword-
	// stripped variant when the raw query under-matches (the registry does
	// prefix matching, so "video compression" finds ~1 skill but "video
	// compress" finds ~9 — and a full-sentence query finds none).
	results, err := skills.SearchBroadened(ctx, t.registryMgr.SearchAll, query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("skill search failed: %v", err))
	}

	// Cache the results.
	if t.cache != nil && len(results) > 0 {
		t.cache.Put(query, results)
	}

	return SilentResult(formatSearchResults(query, results, false))
}

func formatSearchResults(query string, results []skills.SearchResult, cached bool) string {
	if len(results) == 0 {
		return fmt.Sprintf("No skills found for query: %q", query)
	}

	var sb strings.Builder
	source := ""
	if cached {
		source = " (cached)"
	}
	sb.WriteString(fmt.Sprintf("Found %d skills for %q%s:\n\n", len(results), query, source))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**", i+1, r.Slug))
		if r.Version != "" {
			sb.WriteString(fmt.Sprintf(" v%s", r.Version))
		}
		sb.WriteString(fmt.Sprintf("  (score: %.3f, registry: %s)\n", r.Score, r.RegistryName))
		if r.DisplayName != "" && r.DisplayName != r.Slug {
			sb.WriteString(fmt.Sprintf("   Name: %s\n", r.DisplayName))
		}
		if r.Summary != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Summary))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Use install_skill with the slug to install a skill.")
	return sb.String()
}

// isVagueSkillQuery reports whether q is entirely filler words ("new skill",
// "any", "something"). Real queries like "git" or "auth" pass through.
func isVagueSkillQuery(q string) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return true
	}

	// Drop pure filler words; anything left counts as substantive.
	filler := map[string]bool{
		"a": true, "an": true, "the": true,
		"new": true, "any": true, "some": true, "all": true,
		"skill": true, "skills": true,
		"something": true, "anything": true, "everything": true,
		"add": true, "install": true, "find": true, "search": true,
		"please": true, "for": true, "me": true, "to": true,
		"can": true, "you": true, "do": true,
	}

	for _, word := range strings.Fields(q) {
		// Strip simple punctuation so "skill?" still matches "skill".
		word = strings.Trim(word, ".,!?;:\"'()")
		if word == "" {
			continue
		}
		if !filler[word] {
			// Found at least one substantive word. Query is acceptable.
			return false
		}
	}
	// Every word was filler — the query says nothing about what the user
	// actually wants. Reject so the agent has to ask.
	return true
}

package skills

import (
	"context"
	"strings"
)

// The ClawHub registry search is prefix/token based: the query token "compress"
// matches "compressor", "compression" and "compressed", but the full token
// "compression" matches almost nothing. So an inflected or verbose query badly
// under-matches ("video compression" → 1 hit, "video compress" → 9; a full
// sentence → 0). The helpers here broaden a query client-side so both the chat
// find_skills tool and the desktop Config search get good recall.

// skillQueryStopwords are filler words that add noise but no signal to registry
// search. Mirrors the filler set the find_skills vague-query guard uses, plus
// common question words, so "what are the skills available for video
// compressing" reduces to "video compress".
var skillQueryStopwords = map[string]bool{
	"a": true, "an": true, "the": true, "any": true, "some": true, "all": true,
	"skill": true, "skills": true, "new": true, "something": true, "anything": true,
	"add": true, "install": true, "find": true, "search": true, "get": true,
	"please": true, "for": true, "me": true, "to": true, "of": true, "with": true,
	"can": true, "you": true, "do": true, "does": true, "is": true, "are": true,
	"what": true, "which": true, "there": true, "available": true, "have": true,
	"i": true, "my": true, "want": true, "need": true, "help": true, "using": true,
	"and": true, "or": true, "that": true, "this": true, "it": true, "on": true,
}

// stemWord strips a common English inflectional suffix so a prefix-matching
// registry matches "compression"/"compressing"/"compressed" against a record
// that only literally contains "compress". Conservative: only strips when the
// remaining stem stays >= 4 chars, and only the first (longest) matching suffix.
func stemWord(w string) string {
	// Longest suffixes first so "compression" strips "ion" (→ "compress"),
	// not "s" (→ "compression" minus nothing useful).
	suffixes := []string{"ations", "ation", "ings", "ing", "ions", "ion", "ers", "ment", "ness", "ed", "es", "s"}
	for _, suf := range suffixes {
		if len(w) <= len(suf)+3 || !strings.HasSuffix(w, suf) {
			continue
		}
		// Don't strip a plural/verb "s"/"es" off a word that ends in "ss"
		// ("compress", "access") — that mangles an already-good stem.
		if (suf == "s" || suf == "es") && strings.HasSuffix(w, "ss") {
			continue
		}
		return w[:len(w)-len(suf)]
	}
	return w
}

// normalizeSkillQuery lowercases the query, drops stopwords, and stems each
// remaining word — producing a broadened variant tuned for prefix-matching
// registry search. Returns "" when nothing substantive remains.
func normalizeSkillQuery(query string) string {
	fields := strings.Fields(strings.ToLower(query))
	out := make([]string, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, w := range fields {
		w = strings.Trim(w, ".,!?;:\"'()[]")
		if w == "" || skillQueryStopwords[w] {
			continue
		}
		s := stemWord(w)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// SearchFunc is the shared shape of a registry search (RegistryManager.SearchAll
// and ClawHubRegistry.Search both satisfy it).
type SearchFunc func(ctx context.Context, query string, limit int) ([]SearchResult, error)

// SearchBroadened runs the raw query and, when it returns fewer than `limit`
// results, retries with a stopword-stripped + stemmed variant and merges the two
// (dedup by slug, highest score wins, re-sorted by score). Broadening only ever
// ADDS to a sparse result set, so precision at the top is preserved; it just
// fills the tail. Best-effort: any failure on the broadened pass leaves the
// primary results untouched.
func SearchBroadened(ctx context.Context, search SearchFunc, query string, limit int) ([]SearchResult, error) {
	primary, err := search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Already full, or no useful broadening to try.
	broad := normalizeSkillQuery(query)
	if (limit > 0 && len(primary) >= limit) ||
		broad == "" ||
		broad == strings.ToLower(strings.TrimSpace(query)) {
		return primary, nil
	}

	extra, err := search(ctx, broad, limit)
	if err != nil || len(extra) == 0 {
		// Broadening is best-effort: if the retry errors or finds nothing, keep
		// the primary results rather than failing the whole search.
		return primary, nil //nolint:nilerr
	}

	merged := mergeDedupBySlug(primary, extra)
	sortByScoreDesc(merged)
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

// mergeDedupBySlug combines two result sets, keeping the higher-scored copy of
// any slug present in both.
func mergeDedupBySlug(a, b []SearchResult) []SearchResult {
	bySlug := make(map[string]int, len(a)+len(b)) // slug -> index in out
	out := make([]SearchResult, 0, len(a)+len(b))
	add := func(r SearchResult) {
		if i, ok := bySlug[r.Slug]; ok {
			if r.Score > out[i].Score {
				out[i] = r
			}
			return
		}
		bySlug[r.Slug] = len(out)
		out = append(out, r)
	}
	for _, r := range a {
		add(r)
	}
	for _, r := range b {
		add(r)
	}
	return out
}

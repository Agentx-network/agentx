package skills

import (
	"context"
	"testing"
)

func TestStemWord(t *testing.T) {
	cases := map[string]string{
		"compression": "compress",
		"compressing": "compress",
		"compressed":  "compress",
		"videos":      "video",
		"video":       "video", // no over-stemming of short/plain words
		"editing":     "edit",
		"testing":     "test", // -ing stripped
		"css":         "css",  // too short to strip
	}
	for in, want := range cases {
		if got := stemWord(in); got != want {
			t.Errorf("stemWord(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSkillQuery(t *testing.T) {
	cases := map[string]string{
		"video compression": "video compress",
		"video compress":    "video compress",
		"what are the skills available for video compressing?": "video compress",
		"can you help me edit a PDF":                           "edit pdf",
		"":                                                     "",
		"the a an":                                             "", // all stopwords
	}
	for in, want := range cases {
		if got := normalizeSkillQuery(in); got != want {
			t.Errorf("normalizeSkillQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearchBroadened_RetriesWhenSparse(t *testing.T) {
	var queries []string
	// Fake registry: "video compression" under-matches (1 hit), the broadened
	// "video compress" matches many — mirrors the real ClawHub behavior.
	fake := func(_ context.Context, q string, _ int) ([]SearchResult, error) {
		queries = append(queries, q)
		switch q {
		case "video compression":
			return []SearchResult{{Slug: "video", Score: 0.48}}, nil
		case "video compress":
			return []SearchResult{
				{Slug: "upload-video-compressor", Score: 1.96},
				{Slug: "video-compressor", Score: 1.82},
				{Slug: "video", Score: 0.09}, // duplicate slug, lower score
			}, nil
		}
		return nil, nil
	}

	got, err := SearchBroadened(context.Background(), fake, "video compression", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 || queries[1] != "video compress" {
		t.Fatalf("expected a broadened retry with %q, got queries=%v", "video compress", queries)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 merged results, got %d: %+v", len(got), got)
	}
	// Top result must be the highest-scored, and the duplicate "video" slug must
	// keep its higher score (0.48 from primary vs 0.09 from broadened).
	if got[0].Slug != "upload-video-compressor" {
		t.Errorf("expected top slug upload-video-compressor, got %q", got[0].Slug)
	}
	for _, r := range got {
		if r.Slug == "video" && r.Score != 0.48 {
			t.Errorf("dedup should keep higher score 0.48 for 'video', got %v", r.Score)
		}
	}
}

func TestSearchBroadened_NoRetryWhenFull(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, _ string, limit int) ([]SearchResult, error) {
		calls++
		out := make([]SearchResult, limit)
		for i := range out {
			out[i] = SearchResult{Slug: "s", Score: float64(i)}
		}
		return out, nil
	}
	if _, err := SearchBroadened(context.Background(), fake, "video compression", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected no broadening retry when primary fills the limit, got %d calls", calls)
	}
}

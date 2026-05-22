package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListGeminiModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") == "" {
			t.Error("expected key query param")
		}
		w.Write([]byte(`{"models":[
			{"name":"models/gemini-3-pro","displayName":"Gemini 3 Pro","supportedGenerationMethods":["generateContent"]},
			{"name":"models/gemini-2.5-flash","displayName":"Gemini 2.5 Flash","supportedGenerationMethods":["generateContent"]},
			{"name":"models/text-embedding-004","displayName":"Embedding","supportedGenerationMethods":["embedContent"]}
		]}`))
	}))
	defer srv.Close()

	models, err := ListModels(context.Background(), "gemini", srv.URL, "test-key")
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	// Embedding model filtered out; two chat models remain.
	if len(models) != 2 {
		t.Fatalf("want 2 chat models, got %d: %+v", len(models), models)
	}
	if !hasID(models, "gemini/gemini-3-pro") {
		t.Errorf("missing gemini/gemini-3-pro: %+v", models)
	}
}

func TestListOpenAIModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing/wrong auth header: %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"data":[
			{"id":"gpt-5.2"},
			{"id":"o3"},
			{"id":"text-embedding-3-large"},
			{"id":"dall-e-3"},
			{"id":"whisper-1"}
		]}`))
	}))
	defer srv.Close()

	models, err := ListModels(context.Background(), "openai", srv.URL, "sk-test")
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	// gpt-5.2 + o3 kept; embedding/dall-e/whisper filtered.
	if len(models) != 2 {
		t.Fatalf("want 2 chat models, got %d: %+v", len(models), models)
	}
	if !hasID(models, "openai/gpt-5.2") || !hasID(models, "openai/o3") {
		t.Errorf("expected gpt-5.2 and o3: %+v", models)
	}
}

func TestListOpenRouterModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"google/gemini-2.5-pro","name":"Google: Gemini 2.5 Pro"},
			{"id":"anthropic/claude-sonnet-4.6","name":"Anthropic: Claude Sonnet 4.6"}
		]}`))
	}))
	defer srv.Close()

	models, err := ListModels(context.Background(), "openrouter", srv.URL, "")
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 || !hasID(models, "openrouter/google/gemini-2.5-pro") {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestListModels_UnsupportedProvider(t *testing.T) {
	if _, err := ListModels(context.Background(), "cerebras", "", "k"); err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func hasID(models []DiscoveredModel, id string) bool {
	for _, m := range models {
		if m.ID == id {
			return true
		}
	}
	return false
}

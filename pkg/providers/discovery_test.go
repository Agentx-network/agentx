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

func TestValidateKey_OpenRouter(t *testing.T) {
	// Valid: /auth/key returns 200 when the bearer matches.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/key" {
			t.Errorf("expected /auth/key, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-or-good" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"data":{"label":"test"}}`))
	}))
	defer good.Close()
	if err := ValidateKey(context.Background(), "openrouter", good.URL, "sk-or-good"); err != nil {
		t.Errorf("valid key should pass, got: %v", err)
	}

	// Invalid: server returns 401.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"message":"No auth credentials found","code":401}}`))
	}))
	defer bad.Close()
	if err := ValidateKey(context.Background(), "openrouter", bad.URL, "sk-or-bogus"); err == nil {
		t.Error("invalid key should fail validation")
	}

	// Empty key always fails fast (no network call).
	if err := ValidateKey(context.Background(), "openrouter", "", "  "); err == nil {
		t.Error("empty/whitespace key should fail")
	}
}

func TestValidateKey_OpenAICompatible(t *testing.T) {
	// Mistral, DeepSeek, Cerebras, Groq, Moonshot/Kimi, etc. all share the same
	// /v1/models + Bearer auth shape. One server stands in for all of them.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer good-key" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"mistral-large"}]}`))
	}))
	defer srv.Close()

	for _, p := range []string{"mistral", "deepseek", "cerebras", "groq", "moonshot", "kimi", "qwen"} {
		t.Run(p+"_good", func(t *testing.T) {
			if err := ValidateKey(context.Background(), p, srv.URL, "good-key"); err != nil {
				t.Errorf("valid key for %s should pass, got: %v", p, err)
			}
		})
		t.Run(p+"_bad", func(t *testing.T) {
			if err := ValidateKey(context.Background(), p, srv.URL, "wrong-key"); err == nil {
				t.Errorf("invalid key for %s should fail validation", p)
			}
		})
	}
}

func TestValidateKey_Anthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant-good" || r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	if err := ValidateKey(context.Background(), "anthropic", srv.URL, "sk-ant-good"); err != nil {
		t.Errorf("valid anthropic key should pass: %v", err)
	}
	if err := ValidateKey(context.Background(), "anthropic", srv.URL, "sk-ant-bad"); err == nil {
		t.Error("bad anthropic key should fail")
	}
}

func TestValidateKey_UnknownProviderSkips(t *testing.T) {
	// Truly unknown provider (no validation endpoint mapped) → skip cleanly,
	// don't false-positive. Chat-path HumanizeError still catches bad keys later.
	if err := ValidateKey(context.Background(), "some-new-provider", "", "anything"); err != nil {
		t.Errorf("unknown provider should skip, got: %v", err)
	}
}

func TestValidateKey_OllamaLocal(t *testing.T) {
	// Local provider — no real key to validate, must always pass.
	if err := ValidateKey(context.Background(), "ollama", "", "ollama"); err != nil {
		t.Errorf("ollama (local) must skip validation, got: %v", err)
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

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DiscoveredModel is a model fetched live from a provider's models API.
// ID is the full AgentX model reference (e.g. "gemini/gemini-2.5-flash"); Label
// is a human-friendly name for the dropdown.
type DiscoveredModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ListModels queries a provider's live models endpoint so the UI can show the
// real, current model list (including models added after this build shipped)
// instead of a hardcoded catalog. provider is the AgentX prefix ("gemini",
// "openai", "openrouter"); apiBase is the provider's API base URL; apiKey is the
// user's key. Returns models sorted for stable display.
//
// Only providers with a well-defined public models API are supported; callers
// fall back to the static catalog for anything else (or on error).
func ListModels(ctx context.Context, provider, apiBase, apiKey string) ([]DiscoveredModel, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	apiKey = strings.TrimSpace(apiKey)

	switch provider {
	case "gemini", "google":
		return listGeminiModels(ctx, apiBase, apiKey)
	case "openai":
		return listOpenAIModels(ctx, apiBase, apiKey)
	case "openrouter":
		return listOpenRouterModels(ctx, apiBase, apiKey)
	default:
		return nil, fmt.Errorf("live model discovery isn't supported for %q yet", provider)
	}
}

// ValidateKey verifies a provider API key against the provider's own auth/info
// endpoint. Returns nil if the key is valid (or validation isn't supported for
// the provider), or an error describing the rejection. Used at onboarding to
// catch bad keys (typos, whitespace, wrong account, etc.) before the user runs
// into "provider rejected the API key" at first chat.
func ValidateKey(ctx context.Context, provider, apiBase, apiKey string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("the API key is empty")
	}
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")

	switch provider {
	case "gemini", "google":
		// /models requires the key — a reject here means the key is bad.
		_, err := ListModels(ctx, provider, apiBase, apiKey)
		return err
	case "openai":
		_, err := ListModels(ctx, provider, apiBase, apiKey)
		return err
	case "openrouter":
		// OpenRouter's /models is public, so we can't validate via that.
		// /auth/key returns key info and requires Bearer auth — perfect probe.
		if apiBase == "" {
			apiBase = "https://openrouter.ai/api/v1"
		}
		var out map[string]any
		if err := httpGetJSON(ctx, apiBase+"/auth/key",
			map[string]string{"Authorization": "Bearer " + apiKey}, &out); err != nil {
			return err
		}
		return nil
	default:
		// No universal validation endpoint — skip (don't false-positive).
		return nil
	}
}

// httpGetJSON performs a GET with a short timeout and decodes the JSON body.
func httpGetJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, out)
}

func listGeminiModels(ctx context.Context, apiBase, apiKey string) ([]DiscoveredModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("a Gemini API key is required to list models")
	}
	if apiBase == "" {
		apiBase = "https://generativelanguage.googleapis.com/v1beta"
	}
	var out struct {
		Models []struct {
			Name                       string   `json:"name"` // "models/gemini-2.5-flash"
			DisplayName                string   `json:"displayName"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	url := fmt.Sprintf("%s/models?key=%s&pageSize=200", apiBase, apiKey)
	if err := httpGetJSON(ctx, url, nil, &out); err != nil {
		return nil, err
	}
	var models []DiscoveredModel
	for _, m := range out.Models {
		// Keep only chat/generation models (exclude embeddings, etc.).
		if !contains(m.SupportedGenerationMethods, "generateContent") {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		if id == "" {
			continue
		}
		label := m.DisplayName
		if label == "" {
			label = id
		}
		models = append(models, DiscoveredModel{ID: "gemini/" + id, Label: label})
	}
	return dedupeSort(models), nil
}

func listOpenAIModels(ctx context.Context, apiBase, apiKey string) ([]DiscoveredModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("an OpenAI API key is required to list models")
	}
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := httpGetJSON(ctx, apiBase+"/models", map[string]string{"Authorization": "Bearer " + apiKey}, &out); err != nil {
		return nil, err
	}
	var models []DiscoveredModel
	for _, m := range out.Data {
		// The /models list is noisy (embeddings, tts, moderation…). Keep the
		// chat-capable families; anything else is still reachable via custom entry.
		if !isOpenAIChatModel(m.ID) {
			continue
		}
		models = append(models, DiscoveredModel{ID: "openai/" + m.ID, Label: m.ID})
	}
	return dedupeSort(models), nil
}

func listOpenRouterModels(ctx context.Context, apiBase, apiKey string) ([]DiscoveredModel, error) {
	if apiBase == "" {
		apiBase = "https://openrouter.ai/api/v1"
	}
	headers := map[string]string{}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey // optional for listing
	}
	var out struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := httpGetJSON(ctx, apiBase+"/models", headers, &out); err != nil {
		return nil, err
	}
	var models []DiscoveredModel
	for _, m := range out.Data {
		if m.ID == "" {
			continue
		}
		label := m.Name
		if label == "" {
			label = m.ID
		}
		models = append(models, DiscoveredModel{ID: "openrouter/" + m.ID, Label: label})
	}
	return dedupeSort(models), nil
}

// isOpenAIChatModel reports whether an OpenAI model id is a chat/completion
// model worth showing (gpt-*, o1/o3/o4 reasoning, chatgpt-*), filtering out
// embeddings, audio, image, and moderation models.
func isOpenAIChatModel(id string) bool {
	id = strings.ToLower(id)
	for _, bad := range []string{"embedding", "whisper", "tts", "audio", "moderation", "image", "dall-e", "realtime", "transcribe"} {
		if strings.Contains(id, bad) {
			return false
		}
	}
	for _, good := range []string{"gpt-", "o1", "o3", "o4", "chatgpt"} {
		if strings.HasPrefix(id, good) {
			return true
		}
	}
	return false
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// dedupeSort removes duplicate IDs and sorts by label for stable display.
func dedupeSort(models []DiscoveredModel) []DiscoveredModel {
	seen := map[string]bool{}
	out := models[:0]
	for _, m := range models {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

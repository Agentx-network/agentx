package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// modelListResponse matches the OpenAI-compatible /models endpoint shape used
// by Groq, Cerebras, OpenRouter, DeepSeek, Mistral, Ollama, and many others.
type modelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// probeProviderModels queries the provider's /v1/models endpoint and returns
// the list of available model IDs for the given API key. Returns ok=false
// (with no error) if the provider doesn't expose a compatible /models endpoint
// — callers should treat that as "validation unavailable" and skip silently.
func probeProviderModels(ctx context.Context, apiBase, apiKey string) (models []string, ok bool) {
	// Anthropic and Gemini use non-OpenAI shapes; skip rather than guess.
	if strings.Contains(apiBase, "anthropic.com") || strings.Contains(apiBase, "generativelanguage.googleapis.com") {
		return nil, false
	}

	url := strings.TrimRight(apiBase, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, false
	}

	var parsed modelListResponse
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Data) == 0 {
		return nil, false
	}

	out := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out, true
}

// modelIDFromProvider strips the "vendor/" prefix from a model string so it
// can be matched against the IDs returned by /models.
func modelIDFromProvider(model string) string {
	if i := strings.Index(model, "/"); i >= 0 {
		return model[i+1:]
	}
	return model
}

// validateProviderModel returns ok=true if the configured model exists on the
// provider. If validation is unavailable, returns ok=true (don't block setup).
// If validation succeeded but the model is missing, returns ok=false plus the
// list of valid alternatives so the caller can re-prompt.
func validateProviderModel(ctx context.Context, p *providerInfo, apiKey string) (ok bool, available []string) {
	models, probed := probeProviderModels(ctx, p.APIBase, apiKey)
	if !probed {
		return true, nil
	}
	want := modelIDFromProvider(p.Model)
	for _, id := range models {
		if id == want {
			return true, models
		}
	}
	return false, models
}

// formatModelList returns a short human-readable list, capped at 8 entries.
func formatModelList(models []string) string {
	const max = 8
	if len(models) > max {
		return strings.Join(models[:max], ", ") + fmt.Sprintf(", … (+%d more)", len(models)-max)
	}
	return strings.Join(models, ", ")
}

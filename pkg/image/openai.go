package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	openaiDefaultBase  = "https://api.openai.com/v1"
	openaiDefaultModel = "gpt-image-1"
)

// generateOpenAI calls OpenAI's image-generation endpoint. gpt-image-1 returns
// base64 by default; dall-e-3 returns a URL — both are handled.
func generateOpenAI(ctx context.Context, opts GenerateOptions) ([]byte, string, error) {
	return openAICompatImage(ctx, opts, openaiDefaultBase, openaiDefaultModel)
}

// openAICompatImage drives the OpenAI-compatible POST /images/generations API,
// shared by OpenAI and Volcengine/Ark (Seedance). It Bearer-auths and accepts a
// response containing either b64_json or a url per image.
func openAICompatImage(ctx context.Context, opts GenerateOptions, defaultBase, defaultModel string) ([]byte, string, error) {
	model := opts.Model
	if model == "" {
		model = defaultModel
	}
	base := strings.TrimRight(opts.APIBase, "/")
	if base == "" {
		base = defaultBase
	}
	url := base + "/images/generations"

	// response_format is intentionally omitted: gpt-image-1 rejects it (always
	// returns b64_json), while dall-e-3 / Ark default to a URL. We handle both.
	reqBody, _ := json.Marshal(map[string]any{
		"model":  model,
		"prompt": opts.Prompt,
		"n":      1,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", fmt.Errorf("openai image: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("openai image: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("openai image: HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("openai image: decode response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, "", fmt.Errorf("openai image: response contained no image")
	}

	d := parsed.Data[0]
	if d.B64JSON != "" {
		data, err := base64.StdEncoding.DecodeString(d.B64JSON)
		if err != nil {
			return nil, "", fmt.Errorf("openai image: decode base64: %w", err)
		}
		return data, "image/png", nil
	}
	if d.URL != "" {
		return downloadImage(ctx, d.URL)
	}
	return nil, "", fmt.Errorf("openai image: response had neither b64_json nor url")
}

// downloadImage fetches an image from a URL (used by adapters that return links
// rather than inline bytes). Returns the bytes and a best-effort mime type.
func downloadImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("download image: build request: %w", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", fmt.Errorf("download image: read body: %w", err)
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		mime = "image/png"
	}
	return data, mime, nil
}

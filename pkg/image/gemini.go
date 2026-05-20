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
	geminiDefaultBase  = "https://generativelanguage.googleapis.com/v1beta"
	geminiDefaultModel = "gemini-2.5-flash-image"
)

// geminiGenerateRequest is the generateContent request body.
type geminiGenerateRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64
}

// geminiGenerateResponse is the relevant slice of the generateContent response.
type geminiGenerateResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// generateGemini calls Google's generateContent endpoint for an image model and
// returns the first inline image. Google auths via the ?key= query param and
// returns the image as base64 inlineData.
func generateGemini(ctx context.Context, opts GenerateOptions) ([]byte, string, error) {
	model := opts.Model
	if model == "" {
		model = geminiDefaultModel
	}
	base := strings.TrimRight(opts.APIBase, "/")
	if base == "" {
		base = geminiDefaultBase
	}
	// Strip a trailing /v1beta or /v1 the caller may have included so we don't
	// double the path (same gotcha the chat adapter hit).
	base = strings.TrimSuffix(base, "/v1beta")
	base = strings.TrimSuffix(base, "/v1")
	if base == "" {
		base = geminiDefaultBase
	} else if !strings.Contains(base, "/v1") {
		base += "/v1beta"
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", base, model, opts.APIKey)
	reqBody, _ := json.Marshal(geminiGenerateRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: opts.Prompt}}}},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", fmt.Errorf("gemini image: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("gemini image: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20)) // images can be a few MB
	if resp.StatusCode != http.StatusOK {
		var parsed geminiGenerateResponse
		if json.Unmarshal(body, &parsed) == nil && parsed.Error != nil {
			return nil, "", fmt.Errorf("gemini image: %s (HTTP %d)", parsed.Error.Message, resp.StatusCode)
		}
		return nil, "", fmt.Errorf("gemini image: HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("gemini image: decode response: %w", err)
	}

	for _, c := range parsed.Candidates {
		for _, p := range c.Content.Parts {
			if p.InlineData != nil && p.InlineData.Data != "" {
				data, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
				if err != nil {
					return nil, "", fmt.Errorf("gemini image: decode base64: %w", err)
				}
				mime := p.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				return data, mime, nil
			}
		}
	}
	return nil, "", fmt.Errorf("gemini image: response contained no image (model may not support image output)")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

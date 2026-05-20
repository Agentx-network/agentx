package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const replicateDefaultBase = "https://api.replicate.com/v1"

// replicatePrediction is the slice of Replicate's prediction object we use.
type replicatePrediction struct {
	Status string          `json:"status"` // starting | processing | succeeded | failed | canceled
	Output json.RawMessage `json:"output"` // string or []string of image URLs
	Error  string          `json:"error"`
	URLs   struct {
		Get string `json:"get"`
	} `json:"urls"`
}

// generateReplicate runs an image model on Replicate. It uses the model's
// official endpoint (.../models/{owner}/{name}/predictions) with `Prefer: wait`
// so the call blocks until the prediction settles, then downloads the resulting
// image. The model ref (e.g. "black-forest-labs/flux-1.1-pro") comes from the
// Model option.
func generateReplicate(ctx context.Context, opts GenerateOptions) ([]byte, string, error) {
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		return nil, "", fmt.Errorf("replicate image: a model ref like 'owner/name' is required")
	}
	base := strings.TrimRight(opts.APIBase, "/")
	if base == "" {
		base = replicateDefaultBase
	}
	url := fmt.Sprintf("%s/models/%s/predictions", base, model)

	reqBody, _ := json.Marshal(map[string]any{
		"input": map[string]any{"prompt": opts.Prompt},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", fmt.Errorf("replicate image: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	req.Header.Set("Prefer", "wait") // block until the prediction completes

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("replicate image: request failed: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, "", fmt.Errorf("replicate image: HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var pred replicatePrediction
	if err := json.Unmarshal(body, &pred); err != nil {
		return nil, "", fmt.Errorf("replicate image: decode response: %w", err)
	}

	// If `Prefer: wait` didn't fully settle, poll the prediction URL.
	pred, err = waitForReplicate(ctx, client, opts.APIKey, pred)
	if err != nil {
		return nil, "", err
	}
	if pred.Status != "succeeded" {
		if pred.Error != "" {
			return nil, "", fmt.Errorf("replicate image: %s (%s)", pred.Error, pred.Status)
		}
		return nil, "", fmt.Errorf("replicate image: prediction %s", pred.Status)
	}

	imgURL := firstReplicateOutput(pred.Output)
	if imgURL == "" {
		return nil, "", fmt.Errorf("replicate image: prediction produced no image URL")
	}
	return downloadImage(ctx, imgURL)
}

// waitForReplicate polls the prediction's get URL until it reaches a terminal
// state, up to a short budget. Returns immediately if already terminal.
func waitForReplicate(ctx context.Context, client *http.Client, apiKey string, pred replicatePrediction) (replicatePrediction, error) {
	terminal := func(s string) bool {
		return s == "succeeded" || s == "failed" || s == "canceled"
	}
	for i := 0; i < 30 && !terminal(pred.Status); i++ {
		if pred.URLs.Get == "" {
			break
		}
		select {
		case <-ctx.Done():
			return pred, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pred.URLs.Get, nil)
		if err != nil {
			return pred, fmt.Errorf("replicate image: poll: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := client.Do(req)
		if err != nil {
			return pred, fmt.Errorf("replicate image: poll failed: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()

		var next replicatePrediction
		if err := json.Unmarshal(body, &next); err != nil {
			return pred, fmt.Errorf("replicate image: decode poll response: %w", err)
		}
		pred = next
	}
	return pred, nil
}

// firstReplicateOutput extracts the first image URL from a prediction output,
// which is either a JSON string or an array of strings.
func firstReplicateOutput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single string
	if json.Unmarshal(raw, &single) == nil && single != "" {
		return single
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		for _, s := range list {
			if s != "" {
				return s
			}
		}
	}
	return ""
}

package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	antigravityBaseURL     = "https://cloudcode-pa.googleapis.com"
	antigravityUserAgent   = "antigravity"
	antigravityXGoogClient = "google-cloud-sdk vscode_cloudshelleditor/0.1"
)

// AntigravityModelInfo contains information about an available Antigravity model.
type AntigravityModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	IsExhausted bool   `json:"is_exhausted"`
}

// FetchAntigravityProjectID fetches the Cloud Code Assist project ID.
func FetchAntigravityProjectID(accessToken string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	})

	req, err := http.NewRequest("POST", antigravityBaseURL+"/v1internal:loadCodeAssist", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", antigravityUserAgent)
	req.Header.Set("X-Goog-Api-Client", antigravityXGoogClient)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("loadCodeAssist failed: %s", string(body))
	}

	var result struct {
		CloudAICompanionProject string `json:"cloudaicompanionProject"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.CloudAICompanionProject == "" {
		return "", fmt.Errorf("no project ID in loadCodeAssist response")
	}

	return result.CloudAICompanionProject, nil
}

// FetchAntigravityModels fetches available models from the Cloud Code Assist API.
func FetchAntigravityModels(accessToken, projectID string) ([]AntigravityModelInfo, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"project": projectID,
	})

	req, err := http.NewRequest("POST", antigravityBaseURL+"/v1internal:fetchAvailableModels", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", antigravityUserAgent)
	req.Header.Set("X-Goog-Api-Client", antigravityXGoogClient)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"fetchAvailableModels failed (HTTP %d): %s",
			resp.StatusCode,
			truncateBody(string(body), 200),
		)
	}

	var result struct {
		Models map[string]struct {
			DisplayName string `json:"displayName"`
			QuotaInfo   struct {
				RemainingFraction any    `json:"remainingFraction"`
				ResetTime         string `json:"resetTime"`
				IsExhausted       bool   `json:"isExhausted"`
			} `json:"quotaInfo"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing models response: %w", err)
	}

	var models []AntigravityModelInfo
	for id, info := range result.Models {
		models = append(models, AntigravityModelInfo{
			ID:          id,
			DisplayName: info.DisplayName,
			IsExhausted: info.QuotaInfo.IsExhausted,
		})
	}

	// Ensure gemini-3-flash-preview and gemini-3-flash are in the list
	hasFlashPreview := false
	hasFlash := false
	for _, m := range models {
		if m.ID == "gemini-3-flash-preview" {
			hasFlashPreview = true
		}
		if m.ID == "gemini-3-flash" {
			hasFlash = true
		}
	}
	if !hasFlashPreview {
		models = append(models, AntigravityModelInfo{
			ID:          "gemini-3-flash-preview",
			DisplayName: "Gemini 3 Flash (Preview)",
		})
	}
	if !hasFlash {
		models = append(models, AntigravityModelInfo{
			ID:          "gemini-3-flash",
			DisplayName: "Gemini 3 Flash",
		})
	}

	return models, nil
}

func truncateBody(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

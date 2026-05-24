package agent

import "strings"

// humanizeError turns a raw provider/runtime error into a brief, one-sentence
// message safe to show a user. Raw errors leak provider URLs, quota metrics and
// stack-like detail (and trigger link previews in chat); the full error is still
// logged for debugging. Returns a friendly sentence for known cases and a
// generic one otherwise.
func HumanizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "rate-limit") || strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "429") || strings.Contains(msg, "resource_exhausted") ||
		strings.Contains(msg, "exceeded"):
		return "⚠️ This model's free-tier request limit was reached — wait a minute and try again, or switch to a non-rate-limited model in Config → Provider."

	case strings.Contains(msg, "api key") || strings.Contains(msg, "api_key") ||
		strings.Contains(msg, "unauthorized") || strings.Contains(msg, "401") ||
		strings.Contains(msg, "permission denied") || strings.Contains(msg, "invalid_api_key"):
		return "⚠️ The provider rejected the API key — check it in Config → Provider."

	case strings.Contains(msg, "not found in model_list") || strings.Contains(msg, "model not found") ||
		strings.Contains(msg, "no such model") || strings.Contains(msg, "unknown model"):
		return "⚠️ The selected model isn't available — pick another in Config → Provider."

	case strings.Contains(msg, "context") || strings.Contains(msg, "token") ||
		strings.Contains(msg, "too long") || strings.Contains(msg, "length"):
		return "⚠️ That conversation got too long for the model — try a shorter message or start a fresh chat."

	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial tcp"):
		return "⚠️ Couldn't reach the AI provider — check your connection and try again."

	default:
		return "⚠️ Something went wrong handling that — please try again in a moment."
	}
}

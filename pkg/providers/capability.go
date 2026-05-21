package providers

import (
	"regexp"
	"strconv"
	"strings"
)

// sizeBillions matches a parameter-count hint in a model name, e.g. "8b",
// "70b", "235b" (case-insensitive).
var sizeBillions = regexp.MustCompile(`(?i)(\d{1,3})\s*b\b`)

// lowCapabilityWord matches deliberately-small variants as whole words, so it
// flags "gpt-4o-mini" but not "gemini" (which merely contains "mini").
var lowCapabilityWord = regexp.MustCompile(`(?i)\b(mini|nano|lite|tiny|small)\b`)

// IsLowCapabilityModel reports whether a model name suggests a small model that
// tends to be unreliable for agentic work (structured tool calls, multi-step
// plans, longer context). It's a conservative heuristic for a *warning* only —
// not a block. Models under ~14B, or named mini/nano/lite/tiny/small, are
// flagged; larger models (70B/120B/235B…) are not.
func IsLowCapabilityModel(modelRef string) bool {
	name := strings.ToLower(strings.TrimSpace(modelRef))
	if name == "" {
		return false
	}
	if lowCapabilityWord.MatchString(name) {
		return true
	}
	for _, m := range sizeBillions.FindAllStringSubmatch(name, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 && n < 14 {
			return true
		}
	}
	return false
}

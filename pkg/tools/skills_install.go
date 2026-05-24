package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Agentx-network/agentx/pkg/fileutil"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/skills"
	"github.com/Agentx-network/agentx/pkg/utils"
)

// InstallSkillTool allows the LLM agent to install skills from registries.
// It shares the same RegistryManager that FindSkillsTool uses,
// so all registries configured in config are available for installation.
type InstallSkillTool struct {
	registryMgr *skills.RegistryManager
	workspace   string
	mu          sync.Mutex
}

// NewInstallSkillTool creates a new InstallSkillTool.
// registryMgr is the shared registry manager (same instance as FindSkillsTool).
// workspace is the root workspace directory; skills install to {workspace}/skills/{slug}/.
func NewInstallSkillTool(registryMgr *skills.RegistryManager, workspace string) *InstallSkillTool {
	return &InstallSkillTool{
		registryMgr: registryMgr,
		workspace:   workspace,
		mu:          sync.Mutex{},
	}
}

func (t *InstallSkillTool) Name() string {
	return "install_skill"
}

func (t *InstallSkillTool) Description() string {
	return "Install a skill from a registry by slug. Downloads and extracts the skill into the workspace. Use find_skills first to discover available skills."
}

func (t *InstallSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug": map[string]any{
				"type":        "string",
				"description": "The unique slug of the skill to install (e.g., 'github', 'docker-compose')",
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Specific version to install (optional, defaults to latest)",
			},
			"registry": map[string]any{
				"type":        "string",
				"description": "Registry to install from (required, e.g., 'clawhub')",
			},
			"force": map[string]any{
				"type":        "boolean",
				"description": "Force reinstall if skill already exists (default false)",
			},
		},
		"required": []string{"slug", "registry"},
	}
}

func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Install lock to prevent concurrent directory operations.
	// Ideally this should be done at a `slug` level, currently, its at a `workspace` level.
	t.mu.Lock()
	defer t.mu.Unlock()

	// Validate slug
	slug, _ := args["slug"].(string)
	if err := utils.ValidateSkillIdentifier(slug); err != nil {
		return ErrorResult(fmt.Sprintf("invalid slug %q: error: %s", slug, err.Error()))
	}

	// Validate registry
	registryName, _ := args["registry"].(string)
	if err := utils.ValidateSkillIdentifier(registryName); err != nil {
		return ErrorResult(fmt.Sprintf("invalid registry %q: error: %s", registryName, err.Error()))
	}

	// Consent guard: weak LLMs auto-install slugs grabbed from stale history.
	// Deterministic check on the user's current-turn message.
	if tc, ok := GetToolContext(ctx); ok {
		if !userApprovedSlug(tc.UserMessage, slug, tc.LastAssistantMessage) {
			return BlockedResult(
				// User-facing: friendly, action-oriented.
				"Which skill would you like me to install? "+
					"You can tell me by name (e.g. \"install marketplace\") or, after I show search results, say \"install the first one\".",
				// Model-facing: detailed guidance for capable LLMs to iterate on.
				fmt.Sprintf(
					"BLOCKED: User did not explicitly approve installing %q. "+
						"Their most recent message was: %q. "+
						"You MUST NOT install a skill the user did not name in their CURRENT message. "+
						"Reusing slugs from earlier conversation history is not consent. "+
						"Reply to the user with the search results and ask which skill to install. "+
						"Only call install_skill again after the user replies with the slug name or an explicit phrase like 'install <name>' or 'install the first one'.",
					slug, truncateForError(tc.UserMessage, 200),
				),
			)
		}
	}

	version, _ := args["version"].(string)
	force, _ := args["force"].(bool)

	// Check if already installed.
	skillsDir := filepath.Join(t.workspace, "skills")
	targetDir := filepath.Join(skillsDir, slug)

	if !force {
		if _, err := os.Stat(targetDir); err == nil {
			// Non-error: user already has the skill. Worded so the recovery path
			// won't wrap it with "I tried to run install_skill but it failed".
			return &ToolResult{
				ForLLM: fmt.Sprintf(
					"The %q skill is already installed in the user's workspace — nothing to do. "+
						"Tell the user: \"You already have the %q skill installed. "+
						"Let me know if you want me to reinstall it (I can do that with force=true) or pick a different skill.\"",
					slug, slug,
				),
			}
		}
	} else {
		// Force: remove existing if present.
		os.RemoveAll(targetDir)
	}

	// Resolve which registry to use.
	registry := t.registryMgr.GetRegistry(registryName)
	if registry == nil {
		return ErrorResult(fmt.Sprintf("registry %q not found", registryName))
	}

	// Ensure skills directory exists.
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create skills directory: %v", err))
	}

	// Download and install (handles metadata, version resolution, extraction).
	result, err := registry.DownloadAndInstall(ctx, slug, version, targetDir)
	if err != nil {
		// Clean up partial install.
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove partial install",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		// 404 = model invented a slug. Steer it to find_skills instead of
		// retrying with another guess.
		errMsg := err.Error()
		if strings.Contains(errMsg, "404") || strings.Contains(strings.ToLower(errMsg), "not found") {
			return BlockedResult(
				fmt.Sprintf("I couldn't find a skill named %q in the registry. "+
					"Would you like me to search for similar skills instead?", slug),
				fmt.Sprintf(
					"No skill with slug %q is published in the %q registry. "+
						"Do NOT retry with a similar-sounding slug — slugs are exact identifiers. "+
						"Instead, call find_skills with a search query (e.g. find_skills({\"query\":\"marketplace\"})) "+
						"to discover the actual slug of the skill the user wants, then call install_skill with that exact slug. "+
						"If find_skills returns no matches, tell the user the skill isn't available in the registry rather than guessing.",
					slug, registryName,
				),
			)
		}
		return BlockedResult(
			fmt.Sprintf("Sorry, I couldn't install the %q skill — %v", slug, err),
			fmt.Sprintf("failed to install %q: %v", slug, err),
		)
	}

	// Moderation: block malware.
	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			logger.ErrorCF("tool", "Failed to remove partial install",
				map[string]any{
					"tool":       "install_skill",
					"target_dir": targetDir,
					"error":      rmErr.Error(),
				})
		}
		return ErrorResult(fmt.Sprintf("skill %q is flagged as malicious and cannot be installed", slug))
	}

	// Write origin metadata.
	if err := writeOriginMeta(targetDir, registry.Name(), slug, result.Version); err != nil {
		logger.ErrorCF("tool", "Failed to write origin metadata",
			map[string]any{
				"tool":     "install_skill",
				"error":    err.Error(),
				"target":   targetDir,
				"registry": registry.Name(),
				"slug":     slug,
				"version":  result.Version,
			})
		_ = err
	}

	// Build result with moderation warning if suspicious.
	var output string
	if result.IsSuspicious {
		output = fmt.Sprintf("⚠️ Warning: skill %q is flagged as suspicious (may contain risky patterns).\n\n", slug)
	}
	output += fmt.Sprintf("Successfully installed skill %q v%s from %s registry.\nLocation: %s\n",
		slug, result.Version, registry.Name(), targetDir)

	if result.Summary != "" {
		output += fmt.Sprintf("Description: %s\n", result.Summary)
	}
	output += "\nThe skill is now available and can be loaded in the current session."

	return SilentResult(output)
}

// originMeta tracks which registry a skill was installed from.
type originMeta struct {
	Version          int    `json:"version"`
	Registry         string `json:"registry"`
	Slug             string `json:"slug"`
	InstalledVersion string `json:"installed_version"`
	InstalledAt      int64  `json:"installed_at"`
}

func writeOriginMeta(targetDir, registryName, slug, version string) error {
	meta := originMeta{
		Version:          1,
		Registry:         registryName,
		Slug:             slug,
		InstalledVersion: version,
		InstalledAt:      time.Now().UnixMilli(),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(filepath.Join(targetDir, ".skill-origin.json"), data, 0o600)
}

// userApprovedSlug reports whether the user's current-turn message authorises
// installing this slug. Approves on: slug substring; slug-part (≥4 chars)
// substring; install verb + ordinal; plain affirmative + slug in prior reply.
// Strict-by-default — false negatives just force a clarifying turn.
func userApprovedSlug(userMsg, slug, lastAssistantMsg string) bool {
	if userMsg == "" || slug == "" {
		return false
	}
	msg := strings.ToLower(userMsg)
	slugLower := strings.ToLower(slug)

	// Whole-slug substring is the clearest signal.
	if strings.Contains(msg, slugLower) {
		return true
	}

	// Slug parts >=4 chars. "agentx-marketplace" → ["agentx", "marketplace"].
	for _, part := range splitSlugParts(slugLower) {
		if len(part) >= 4 && strings.Contains(msg, part) {
			return true
		}
	}

	// Ordinal reference: "install the first" (verb+ordinal) or "first one"
	// alone (only when the slug is in the prior assistant reply).
	ordinals := []string{
		"first", "1st", "top",
		"second", "2nd",
		"third", "3rd",
		"fourth", "4th",
		"fifth", "5th",
		" 1 ", " 2 ", " 3 ", " 4 ", " 5 ",
	}
	hasOrdinal := false
	for _, o := range ordinals {
		if strings.Contains(" "+msg+" ", o) { // pad so " 1 "-style tokens match at edges
			hasOrdinal = true
			break
		}
	}

	if hasOrdinal {
		// Form 1: explicit verb + ordinal.
		installVerbs := []string{"install", "add", "get", "download", "fetch"}
		for _, v := range installVerbs {
			if strings.Contains(msg, v) {
				return true
			}
		}
		// Form 2: ordinal alone, but only if the slug is in the prior reply.
		if lastAssistantMsg != "" &&
			strings.Contains(strings.ToLower(lastAssistantMsg), slugLower) {
			return true
		}
	}

	// Plain "yes" / "ok" / "do it" approves only when the slug appears in the
	// prior assistant reply (so the user is confirming what was just offered).
	if isPlainAffirmative(msg) && lastAssistantMsg != "" {
		if strings.Contains(strings.ToLower(lastAssistantMsg), slugLower) {
			return true
		}
	}

	return false
}

// isPlainAffirmative matches short, unambiguous consent words. Narrow on
// purpose — "yes I don't want that" should not pass.
func isPlainAffirmative(msgLower string) bool {
	s := strings.TrimSpace(msgLower)
	if s == "" || len(s) > 40 {
		return false
	}
	exact := map[string]bool{
		"y": true, "yes": true, "yep": true, "yeah": true, "yup": true,
		"ok": true, "okay": true, "k": true,
		"sure": true, "yup ok": true,
		"do it": true, "go": true, "go ahead": true,
		"proceed": true, "confirm": true, "confirmed": true,
		"install it": true, "install that": true, "install this": true,
		"yes please": true, "ok please": true, "please do": true,
		"sounds good": true, "looks good": true,
	}
	return exact[s]
}

// splitSlugParts splits "agentx-marketplace" → ["agentx", "marketplace"].
func splitSlugParts(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
}

// truncateForError shortens s for inclusion in an LLM-readable error.
func truncateForError(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

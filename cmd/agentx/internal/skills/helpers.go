package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/skills"
	"github.com/Agentx-network/agentx/pkg/skills/builtin"
	"github.com/Agentx-network/agentx/pkg/utils"
)

func skillsListCmd(loader *skills.SkillsLoader) {
	allSkills := loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
}

func skillsInstallCmd(installer *skills.SkillInstaller, repo string) error {
	fmt.Printf("Installing skill from %s...\n", repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := installer.InstallFromGitHub(ctx, repo); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("\u2713 Skill '%s' installed successfully!\n", filepath.Base(repo))

	return nil
}

// skillsInstallFromRegistry installs a skill from a named registry (e.g. clawhub).
func skillsInstallFromRegistry(cfg *config.Config, registryName, slug string) error {
	err := utils.ValidateSkillIdentifier(registryName)
	if err != nil {
		return fmt.Errorf("✗  invalid registry name: %w", err)
	}

	err = utils.ValidateSkillIdentifier(slug)
	if err != nil {
		return fmt.Errorf("✗  invalid slug: %w", err)
	}

	fmt.Printf("Installing skill '%s' from %s registry...\n", slug, registryName)

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	registry := registryMgr.GetRegistry(registryName)
	if registry == nil {
		return fmt.Errorf("✗  registry '%s' not found or not enabled. check your config.json.", registryName)
	}

	workspace := cfg.WorkspacePath()
	targetDir := filepath.Join(workspace, "skills", slug)

	if _, err = os.Stat(targetDir); err == nil {
		return fmt.Errorf("\u2717 skill '%s' already installed at %s", slug, targetDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err = os.MkdirAll(filepath.Join(workspace, "skills"), 0o755); err != nil {
		return fmt.Errorf("\u2717 failed to create skills directory: %v", err)
	}

	result, err := registry.DownloadAndInstall(ctx, slug, "", targetDir)
	if err != nil {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}
		return fmt.Errorf("✗ failed to install skill: %w", err)
	}

	if result.IsMalwareBlocked {
		rmErr := os.RemoveAll(targetDir)
		if rmErr != nil {
			fmt.Printf("\u2717 Failed to remove partial install: %v\n", rmErr)
		}

		return fmt.Errorf("\u2717 Skill '%s' is flagged as malicious and cannot be installed.\n", slug)
	}

	if result.IsSuspicious {
		fmt.Printf("\u26a0\ufe0f  Warning: skill '%s' is flagged as suspicious.\n", slug)
	}

	fmt.Printf("\u2713 Skill '%s' v%s installed successfully!\n", slug, result.Version)
	if result.Summary != "" {
		fmt.Printf("  %s\n", result.Summary)
	}

	return nil
}

func skillsRemoveCmd(installer *skills.SkillInstaller, skillName string) {
	fmt.Printf("Removing skill '%s'...\n", skillName)

	if err := installer.Uninstall(skillName); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
}

func skillsInstallBuiltinCmd(workspace string) {
	workspaceSkillsDir := filepath.Join(workspace, "skills")

	fmt.Println("Installing builtin skills to workspace...")

	installed, err := builtin.InstallAll(workspaceSkillsDir, false)
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return
	}

	if len(installed) == 0 {
		fmt.Println("All builtin skills already installed.")
	} else {
		for _, name := range installed {
			fmt.Printf("  ✓ %s\n", name)
		}
		fmt.Printf("\n✓ %d builtin skill(s) installed!\n", len(installed))
	}
}

func skillsListBuiltinCmd() {
	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	skills, err := builtin.List()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(skills) == 0 {
		fmt.Println("No builtin skills available.")
		return
	}

	for _, skill := range skills {
		// Extract first non-empty, non-heading line as description
		description := ""
		for _, line := range strings.Split(string(skill.Content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			description = line
			break
		}
		fmt.Printf("  ✓  %s\n", skill.Name)
		if description != "" {
			fmt.Printf("     %s\n", description)
		}
	}
}

func skillsSearchCmd(cfg *config.Config, query string) {
	fmt.Printf("Searching for '%s'...\n", query)

	registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
		MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
		ClawHub:               skills.ClawHubConfig(cfg.Tools.Skills.Registries.ClawHub),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := registryMgr.SearchAll(ctx, query, 10)
	if err != nil {
		fmt.Printf("✗ Search failed: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("No skills found.")
		return
	}

	fmt.Printf("\nFound %d skills:\n", len(results))
	fmt.Println("--------------------")
	for _, r := range results {
		fmt.Printf("  📦 %s (%s)\n", r.DisplayName, r.Slug)
		fmt.Printf("     %s\n", r.Summary)
		if r.Version != "" {
			fmt.Printf("     Version: %s\n", r.Version)
		}
		fmt.Printf("     Install: agentx skills install clawhub %s --registry clawhub\n", r.Slug)
		fmt.Println()
	}
}

func skillsShowCmd(loader *skills.SkillsLoader, skillName string) {
	content, ok := loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
}


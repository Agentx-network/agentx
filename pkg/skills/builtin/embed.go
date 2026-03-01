package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed */SKILL.md
var skillsFS embed.FS

// SkillInfo describes a builtin skill.
type SkillInfo struct {
	Name    string
	Content []byte
}

// List returns all embedded builtin skills.
func List() ([]SkillInfo, error) {
	var skills []SkillInfo
	entries, err := fs.ReadDir(skillsFS, ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(skillsFS, entry.Name()+"/SKILL.md")
		if err != nil {
			continue
		}
		skills = append(skills, SkillInfo{
			Name:    entry.Name(),
			Content: data,
		})
	}
	return skills, nil
}

// InstallAll extracts all builtin skills to the given workspace skills directory.
// Skips skills that already exist unless force is true.
func InstallAll(workspaceSkillsDir string, force bool) (installed []string, err error) {
	skills, err := List()
	if err != nil {
		return nil, fmt.Errorf("list builtin skills: %w", err)
	}

	if err := os.MkdirAll(workspaceSkillsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create skills dir: %w", err)
	}

	for _, skill := range skills {
		targetDir := filepath.Join(workspaceSkillsDir, skill.Name)
		targetFile := filepath.Join(targetDir, "SKILL.md")

		if !force {
			if _, err := os.Stat(targetDir); err == nil {
				continue // already exists
			}
		}

		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return installed, fmt.Errorf("create %s: %w", skill.Name, err)
		}
		if err := os.WriteFile(targetFile, skill.Content, 0o644); err != nil {
			return installed, fmt.Errorf("write %s: %w", skill.Name, err)
		}
		installed = append(installed, skill.Name)
	}
	return installed, nil
}

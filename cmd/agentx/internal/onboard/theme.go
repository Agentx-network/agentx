package onboard

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	neonPink   = lipgloss.Color("#FF0092")
	neonPurple = lipgloss.Color("#AE00FF")
	neonCyan   = lipgloss.Color("#00FFFF")
	dimGray    = lipgloss.Color("#666666")
)

func neonTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused styles
	t.Focused.Title = t.Focused.Title.Foreground(neonPink).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(neonPurple)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(neonCyan)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(neonCyan)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(lipgloss.Color("#CCCCCC"))
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("#000000")).Background(neonCyan)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(dimGray)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(neonCyan)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(neonPink)

	// Blurred styles
	t.Blurred.Title = t.Blurred.Title.Foreground(dimGray)
	t.Blurred.Description = t.Blurred.Description.Foreground(dimGray)
	t.Blurred.SelectSelector = t.Blurred.SelectSelector.Foreground(dimGray)
	t.Blurred.SelectedOption = t.Blurred.SelectedOption.Foreground(dimGray)

	return t
}

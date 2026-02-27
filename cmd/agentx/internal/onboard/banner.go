package onboard

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const asciiLogo = `
    _    ____ _____ _   _ _______  __
   / \  / ___| ____| \ | |_   _\ \/ /
  / _ \| |  _|  _| |  \| | | |  \  /
 / ___ \ |_| | |___| |\  | | |  /  \
/_/   \_\____|_____|_| \_| |_| /_/\_\
`

func printBanner() {
	logoStyle := lipgloss.NewStyle().Foreground(neonPink).Bold(true)
	taglineStyle := lipgloss.NewStyle().Foreground(neonPurple)

	fmt.Println(logoStyle.Render(asciiLogo))
	fmt.Println(taglineStyle.Render("  Ultra-lightweight personal AI agent"))
	fmt.Println()
}

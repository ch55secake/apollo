package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const fullLogo = `████████████                      ████  ████
  ███▒▒▒▒▒███                    ▒▒███ ▒▒███
 ▒███    ▒███  ████████   ██████  ▒███  ▒███   ██████
 ▒███████████ ▒▒███▒▒███ ███▒▒███ ▒███  ▒███  ███▒▒███
 ▒███▒▒▒▒▒███  ▒███ ▒███▒███ ▒███ ▒███  ▒███ ▒███ ▒███
 ▒███    ▒███  ▒███ ▒███▒███ ▒███ ▒███  ▒███ ▒███ ▒███
 █████   █████ ▒███████ ▒▒██████  █████ █████▒▒██████
▒▒▒▒▒   ▒▒▒▒▒  ▒███▒▒▒   ▒▒▒▒▒▒  ▒▒▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒
               ▒███
               █████
              ▒▒▒▒▒`

func renderBrand(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	logoWidth := lipgloss.Width(fullLogo)
	logoHeight := strings.Count(fullLogo, "\n") + 1
	if width >= logoWidth+4 && height >= logoHeight+10 {
		return apolloTheme.Brand.Render(fullLogo)
	}
	return apolloTheme.Brand.Render("APOLLO")
}

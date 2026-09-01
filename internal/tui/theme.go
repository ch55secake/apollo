package tui

import "github.com/charmbracelet/lipgloss"

type uiTheme struct {
	Brand                   lipgloss.Style
	Eyebrow                 lipgloss.Style
	Title                   lipgloss.Style
	Section                 lipgloss.Style
	Muted                   lipgloss.Style
	Error                   lipgloss.Style
	Success                 lipgloss.Style
	Warning                 lipgloss.Style
	Key                     lipgloss.Style
	Badge                   lipgloss.Style
	Shell                   lipgloss.Style
	Panel                   lipgloss.Style
	PanelSelected           lipgloss.Style
	MenuItem                lipgloss.Style
	MenuSelected            lipgloss.Style
	ListTitle               lipgloss.Style
	ListDescription         lipgloss.Style
	ListSelectedTitle       lipgloss.Style
	ListSelectedDescription lipgloss.Style
}

var apolloTheme = uiTheme{
	Brand:                   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#8A4B08", Dark: "#F7C65A"}),
	Eyebrow:                 lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#147D8D", Dark: "#64D8E7"}),
	Title:                   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#213047", Dark: "#F1F5FF"}),
	Section:                 lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#324A6D", Dark: "#C8D7F5"}),
	Muted:                   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#607089", Dark: "#8492AB"}),
	Error:                   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF7B81"}),
	Success:                 lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#197A55", Dark: "#5DDE9E"}),
	Warning:                 lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#FFD166"}),
	Key:                     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#7A4100", Dark: "#1D2638"}).Background(lipgloss.AdaptiveColor{Light: "#F7C65A", Dark: "#F7C65A"}).Padding(0, 1),
	Badge:                   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#6C3C8F", Dark: "#D6A7FF"}).Background(lipgloss.AdaptiveColor{Light: "#F0E5FA", Dark: "#322342"}).Padding(0, 1),
	Shell:                   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#B7C4D9", Dark: "#34445F"}).Padding(1, 2),
	Panel:                   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#C7D2E3", Dark: "#33445F"}).Padding(0, 1),
	PanelSelected:           lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#C98512", Dark: "#F7C65A"}).Padding(0, 1),
	MenuItem:                lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#141A24"}).Padding(0, 1),
	MenuSelected:            lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#FFF4D7", Dark: "#2B2A25"}).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.AdaptiveColor{Light: "#C98512", Dark: "#F7C65A"}).Padding(0, 1),
	ListTitle:               lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#324A6D", Dark: "#C8D7F5"}),
	ListDescription:         lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#607089", Dark: "#8492AB"}),
	ListSelectedTitle:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#7A4100", Dark: "#F7C65A"}),
	ListSelectedDescription: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#147D8D", Dark: "#64D8E7"}),
}

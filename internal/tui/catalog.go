package tui

import (
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type catalogSort int

const (
	titleSort catalogSort = iota
	starredSort
	sourceSort
)

func (s catalogSort) String() string {
	switch s {
	case starredSort:
		return "STARRED FIRST"
	case sourceSort:
		return "SOURCE"
	default:
		return "TITLE"
	}
}

func (s catalogSort) next() catalogSort {
	return (s + 1) % 3
}

type catalogDelegate struct{}

func (catalogDelegate) Height() int  { return 3 }
func (catalogDelegate) Spacing() int { return 1 }
func (catalogDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (catalogDelegate) Render(w io.Writer, model list.Model, index int, item list.Item) {
	dashboardItem, ok := item.(dashboardItem)
	if !ok || model.Width() <= 0 {
		return
	}

	summary := dashboardItem.summary
	selected := index == model.Index() && model.FilterState() != list.Filtering
	titleStyle := apolloTheme.ListTitle
	detailStyle := apolloTheme.ListDescription
	if selected {
		titleStyle = apolloTheme.ListSelectedTitle
		detailStyle = apolloTheme.ListSelectedDescription
	}

	marker := "  "
	if selected {
		marker = "▸ "
	}
	star := ""
	if summary.Starred {
		star = "  *"
	}
	titleWidth := max(1, model.Width()-lipgloss.Width(marker)-lipgloss.Width(star))
	title := marker + truncate(catalogTitle(summary), titleWidth) + star

	metadata := lipgloss.JoinHorizontal(
		lipgloss.Center,
		apolloTheme.Badge.Render(catalogSourceLabel(summary)),
		"  ",
		catalogLocator(summary),
	)
	metadata = detailStyle.Render(metadata)

	tags := strings.Join(summary.Tags, ", ")
	if tags == "" {
		tags = "UNTAGGED"
	}
	details := "tags: " + tags
	if summary.FolderUID != "" {
		details += "  folder: " + summary.FolderUID
	}
	details = detailStyle.Render(details)

	lines := []string{
		catalogLine(titleStyle.Render(title), model.Width()),
		catalogLine(metadata, model.Width()),
		catalogLine(details, model.Width()),
	}
	_, _ = io.WriteString(w, strings.Join(lines, "\n"))
}

func catalogTitle(summary dashboard.DashboardSummary) string {
	if summary.Title == "" {
		return "Untitled dashboard"
	}
	return summary.Title
}

func catalogSourceLabel(summary dashboard.DashboardSummary) string {
	source := strings.ToUpper(strings.TrimSpace(summary.Source))
	if source == "" {
		return "SOURCE"
	}
	if source == "FILE" {
		return "LOCAL FILE"
	}
	return source
}

func catalogLocator(summary dashboard.DashboardSummary) string {
	if strings.EqualFold(summary.Source, "file") {
		if summary.ID != "" {
			return filepath.Base(summary.ID)
		}
		return emptyDash(summary.UID)
	}
	if summary.UID != "" {
		return summary.UID
	}
	return emptyDash(summary.ID)
}

func catalogFilterValue(summary dashboard.DashboardSummary) string {
	values := []string{
		catalogTitle(summary),
		summary.UID,
		catalogSourceLabel(summary),
		summary.FolderUID,
		strings.Join(summary.Tags, " "),
	}
	if strings.EqualFold(summary.Source, "file") {
		values = append(values, filepath.Base(summary.ID))
	}
	return strings.Join(values, " ")
}

func catalogLine(value string, width int) string {
	return lipgloss.NewStyle().MaxWidth(max(1, width)).Render(value)
}

func sortedSummaries(summaries []dashboard.DashboardSummary, mode catalogSort) []dashboard.DashboardSummary {
	result := append([]dashboard.DashboardSummary(nil), summaries...)
	sort.SliceStable(result, func(left, right int) bool {
		first := result[left]
		second := result[right]
		switch mode {
		case starredSort:
			if first.Starred != second.Starred {
				return first.Starred
			}
		case sourceSort:
			if strings.ToLower(first.Source) != strings.ToLower(second.Source) {
				return strings.ToLower(first.Source) < strings.ToLower(second.Source)
			}
		}
		firstTitle := strings.ToLower(catalogTitle(first))
		secondTitle := strings.ToLower(catalogTitle(second))
		if firstTitle == secondTitle {
			return first.ID < second.ID
		}
		return firstTitle < secondTitle
	})
	return result
}

func catalogItems(summaries []dashboard.DashboardSummary) []list.Item {
	items := make([]list.Item, 0, len(summaries))
	for _, summary := range summaries {
		items = append(items, dashboardItem{summary: summary})
	}
	return items
}

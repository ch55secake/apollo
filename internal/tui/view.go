package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	selectedBorder = lipgloss.Color("205")
	panelBorder    = lipgloss.Color("238")
)

func (m Model) View() string {
	switch m.screen {
	case dashboardListScreen:
		return m.listView()
	case dashboardDetailScreen:
		return m.dashboardView()
	case queryScreen:
		return m.queryView()
	default:
		return ""
	}
}

func (m Model) listView() string {
	if m.loadMode {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Load dashboard"),
			mutedStyle.Render("Enter a JSON file or directory path."),
			m.loadInput.View(),
			"",
			m.list.View(),
			footer("enter load   esc cancel   ctrl+c quit"),
		)
	}
	if m.listError == nil && !m.listLoading {
		return lipgloss.JoinVertical(lipgloss.Left, m.list.View(), footer("l load path   r refresh   ctrl+c quit"))
	}
	status := "Loading dashboards..."
	if m.listError != nil {
		status = errorStyle.Render("Unable to load dashboards: " + m.listError.Error())
	}
	return lipgloss.JoinVertical(lipgloss.Left, status, m.list.View(), footer("l load path   r refresh   ctrl+c quit"))
}

func (m Model) dashboardView() string {
	name := m.selectedSummary.Title
	if m.dashboard != nil {
		name = m.dashboard.Title
	}
	header := titleStyle.Render(name)
	if m.detailLoading {
		return lipgloss.JoinVertical(lipgloss.Left, header, "", "Loading dashboard...", footer("esc back   q quit"))
	}
	if m.detailError != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"",
			errorStyle.Render("Unable to load dashboard: "+m.detailError.Error()),
			footer("esc back   r retry   q quit"),
		)
	}
	if m.dashboard == nil {
		return lipgloss.JoinVertical(lipgloss.Left, header, "", mutedStyle.Render("No dashboard selected"), footer("esc back   q quit"))
	}

	from := m.dashboard.Time.From
	to := m.dashboard.Time.To
	if from == "" {
		from = m.options.DefaultTimeRange
	}
	if to == "" {
		to = "now"
	}
	meta := mutedStyle.Render(fmt.Sprintf("%d panels   %s to %s", len(m.dashboard.Panels), from, to))
	content := renderPanelRows(m)
	return lipgloss.JoinVertical(lipgloss.Left, header, meta, "", content, footer("j/k move   enter query   r refresh   esc back   q quit"))
}

func (m Model) queryView() string {
	if m.dashboard == nil || m.selectedPanel >= len(m.dashboard.Panels) {
		return "No query selected"
	}
	panel := m.dashboard.Panels[m.selectedPanel]
	header := titleStyle.Render(panel.Title)
	return lipgloss.JoinVertical(lipgloss.Left, header, m.queryScroll.View(), footer("scroll   r refresh   esc back   q quit"))
}

func (m Model) queryContent() string {
	if m.dashboard == nil || m.selectedPanel >= len(m.dashboard.Panels) {
		return "No query selected"
	}
	panel := m.dashboard.Panels[m.selectedPanel]
	if len(panel.Targets) == 0 || m.selectedTarget >= len(panel.Targets) {
		return mutedStyle.Render("This panel does not contain a query target.")
	}
	target := panel.Targets[m.selectedTarget]
	key := queryKey(m.selectedPanel, m.selectedTarget)
	var builder strings.Builder
	fmt.Fprintf(&builder, "Target %s\n", target.RefID)
	fmt.Fprintf(&builder, "PromQL: %s\n", dashboard.ExpandQuery(target.Expr, m.dashboard.Variables))
	fmt.Fprintf(&builder, "Legend: %s\n\n", emptyDash(target.LegendFormat))
	if err := m.queryErrors[key]; err != nil {
		builder.WriteString(errorStyle.Render(err.Error()))
		return builder.String()
	}
	result, ok := m.queryResults[key]
	if !ok {
		builder.WriteString("Loading query result...")
		return builder.String()
	}
	builder.WriteString(renderResult(result, max(20, m.width-4), max(6, m.height-8), true))
	return builder.String()
}

func renderPanelRows(m Model) string {
	if len(m.dashboard.Panels) == 0 {
		return mutedStyle.Render("This dashboard has no panels.")
	}
	indices := make([]int, len(m.dashboard.Panels))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left := m.dashboard.Panels[indices[i]].GridPos
		right := m.dashboard.Panels[indices[j]].GridPos
		if left.Y == right.Y {
			return left.X < right.X
		}
		return left.Y < right.Y
	})

	var rows []string
	for start := 0; start < len(indices); {
		rowY := m.dashboard.Panels[indices[start]].GridPos.Y
		end := start
		for end < len(indices) && m.dashboard.Panels[indices[end]].GridPos.Y == rowY {
			end++
		}
		cards := make([]string, 0, end-start)
		for _, index := range indices[start:end] {
			panel := m.dashboard.Panels[index]
			width := panelWidth(m.width, panel.GridPos.W, end-start)
			height := panelHeight(panel.GridPos.H)
			cards = append(cards, renderPanel(m, index, panel, width, height))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
		start = end
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderPanel(m Model, index int, panel dashboard.Panel, width, height int) string {
	border := panelBorder
	if index == m.selectedPanel {
		border = selectedBorder
	}
	innerWidth := max(10, width-4)
	innerHeight := max(2, height-3)
	content := "No query targets"
	if panel.Text != "" {
		content = panel.Text
	}
	if len(panel.Targets) > 0 {
		key := queryKey(index, 0)
		if err := m.queryErrors[key]; err != nil {
			content = errorStyle.Render(err.Error())
		} else if result, ok := m.queryResults[key]; ok {
			content = renderResult(result, innerWidth, innerHeight, isChartPanel(panel.Type))
		} else {
			content = "Loading query..."
		}
	}
	title := panel.Title
	if title == "" {
		title = panel.Type
	}
	if panel.Row != "" {
		title = panel.Row + " / " + title
	}
	return lipgloss.NewStyle().
		Width(max(10, width-2)).
		Height(max(3, height-2)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(title + "\n" + content)
}

func renderResult(result prometheus.Result, width, height int, chart bool) string {
	if len(result.Warnings) > 0 {
		warning := mutedStyle.Render("Warnings: " + strings.Join(result.Warnings, "; "))
		if len(result.Series) == 0 {
			return warning
		}
	}
	if chart && len(result.Series) > 0 {
		return renderChart(result.Series, width, height)
	}
	if result.Scalar != nil {
		return fmt.Sprintf("%.4g", result.Scalar.Value)
	}
	if result.Text != "" {
		return result.Text
	}
	if len(result.Series) > 0 {
		return renderSeriesSummary(result.Series)
	}
	return "No data"
}

func renderChart(series []prometheus.Series, width, height int) string {
	chart := timeserieslinechart.New(max(20, width), max(6, height))
	for index, item := range series {
		name := formatLabels(item.Labels)
		if name == "" {
			name = fmt.Sprintf("series-%d", index+1)
		}
		for _, sample := range item.Samples {
			if math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
				continue
			}
			chart.PushDataSet(name, timeserieslinechart.TimePoint{Time: sample.Timestamp, Value: sample.Value})
		}
	}
	chart.DrawBrailleAll()
	return chart.View()
}

func renderSeriesSummary(series []prometheus.Series) string {
	lines := make([]string, 0, len(series))
	for _, item := range series {
		if len(item.Samples) == 0 {
			continue
		}
		last := item.Samples[len(item.Samples)-1]
		lines = append(lines, fmt.Sprintf("%-20s %.4g", truncate(formatLabels(item.Labels), 20), last.Value))
	}
	if len(lines) == 0 {
		return "No samples"
	}
	return strings.Join(lines, "\n")
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func isChartPanel(panelType string) bool {
	switch strings.ToLower(panelType) {
	case "graph", "timeseries", "timeseries-chart":
		return true
	default:
		return false
	}
}

func panelWidth(totalWidth, gridWidth, columns int) int {
	if totalWidth <= 0 {
		return 40
	}
	if gridWidth <= 0 {
		gridWidth = 24 / max(1, columns)
	}
	width := totalWidth * gridWidth / 24
	return max(20, width-1)
}

func panelHeight(gridHeight int) int {
	if gridHeight <= 0 {
		return 8
	}
	return max(6, min(16, gridHeight))
}

func footer(value string) string {
	return "\n" + mutedStyle.Render(value)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func truncate(value string, width int) string {
	if len(value) <= width {
		return value
	}
	return value[:max(0, width-3)] + "..."
}

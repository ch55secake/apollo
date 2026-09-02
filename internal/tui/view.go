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

func (m Model) View() string {
	switch m.screen {
	case mainMenuScreen:
		return m.menuView()
	case dashboardListScreen:
		return m.listView()
	case dashboardDetailScreen:
		return m.dashboardView()
	case queryScreen:
		return m.queryView()
	case connectionScreen:
		return m.connectionView()
	case helpScreen:
		return m.helpView()
	default:
		return ""
	}
}

func (m Model) menuView() string {
	width := m.viewWidth()
	contentWidth := max(1, width-apolloTheme.Shell.GetHorizontalFrameSize())
	brand := renderBrand(width, m.viewHeight())
	eyebrow := apolloTheme.Eyebrow.Render("APOLLO // OBSERVABILITY CONSOLE")
	intro := apolloTheme.Muted.Render(truncate("A focused command deck for your dashboards and telemetry.", contentWidth))
	menuWidth := min(contentWidth+apolloTheme.Shell.GetHorizontalPadding(), 86)
	menuContentWidth := max(1, menuWidth-apolloTheme.Shell.GetHorizontalPadding())
	rowWidth := max(1, menuContentWidth-1)

	var items []string
	for index, item := range menuItems {
		key := apolloTheme.Key.Render(item.key)
		label := apolloTheme.Title.Render(item.title)
		line := lipgloss.JoinHorizontal(lipgloss.Center, key, " ", label)
		if menuContentWidth >= 64 {
			line = lipgloss.JoinHorizontal(lipgloss.Center, line, apolloTheme.Muted.Render("  "+item.description))
		}
		if index == m.menuIndex {
			line = apolloTheme.MenuSelected.Width(rowWidth).Render("▸ " + line)
		} else {
			line = apolloTheme.MenuItem.Width(rowWidth).Render("  " + line)
		}
		items = append(items, line)
	}
	menu := apolloTheme.Shell.Width(menuWidth).Render(strings.Join(items, "\n"))

	sections := []string{brand, eyebrow}
	if m.viewHeight() >= 22 {
		sections = append(sections, intro)
	}
	sections = append(sections, "", menu, "", m.menuStatus(contentWidth))
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	centered := lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
	content = lipgloss.JoinVertical(lipgloss.Left, centered, m.footer("j/k move   enter select   1-4 quick select   q quit"))
	return lipgloss.PlaceVertical(m.viewHeight(), lipgloss.Center, content)
}

func (m Model) listView() string {
	status := m.catalogStatus()
	body := m.list.View()
	if m.loadMode {
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			apolloTheme.Section.Render("LOAD DASHBOARD JSON"),
			apolloTheme.Muted.Render("Enter a JSON file or directory path."),
			m.loadInput.View(),
			"",
			body,
		)
	}
	return m.shell("Dashboard catalog", status, body, "j/k move   enter open   l load path   r refresh   esc menu   q quit")
}

func (m Model) dashboardView() string {
	name := m.selectedSummary.Title
	if m.dashboard != nil {
		name = m.dashboard.Title
	}
	if m.detailLoading {
		return m.shell("Dashboard", name, apolloTheme.Warning.Render("SYNCING DASHBOARD..."), "esc back   q quit")
	}
	if m.detailError != nil {
		body := lipgloss.JoinVertical(
			lipgloss.Left,
			apolloTheme.Error.Render("Unable to load dashboard"),
			apolloTheme.Muted.Render(m.detailError.Error()),
		)
		return m.shell("Dashboard", name, body, "r retry   esc back   q quit")
	}
	if m.dashboard == nil {
		return m.shell("Dashboard", name, apolloTheme.Muted.Render("No dashboard selected"), "esc back   q quit")
	}

	from := m.dashboard.Time.From
	to := m.dashboard.Time.To
	if from == "" {
		from = m.options.DefaultTimeRange
	}
	if to == "" {
		to = "now"
	}
	meta := fmt.Sprintf("%d panels   %s to %s", len(m.dashboard.Panels), from, to)
	body := m.dashboardScroll.View()
	return m.shell("Dashboard", m.dashboard.Title+"  "+apolloTheme.Muted.Render(meta), body, "j/k move   enter inspect query   r refresh   esc catalog   q quit")
}

func (m Model) queryView() string {
	if m.dashboard == nil || m.selectedPanel < 0 || m.selectedPanel >= len(m.dashboard.Panels) {
		return m.shell("Query detail", "", apolloTheme.Muted.Render("No query selected"), "esc back   q quit")
	}
	panel := m.dashboard.Panels[m.selectedPanel]
	targetCount := len(panel.Targets)
	targetMeta := fmt.Sprintf("target %d/%d", m.selectedTarget+1, targetCount)
	return m.shell("Query detail", panel.Title+"  "+apolloTheme.Badge.Render(targetMeta), m.queryScroll.View(), "h/l or left/right target   scroll   r refresh   esc back   q quit")
}

func (m Model) connectionView() string {
	dashboardEndpoint := m.options.DashboardEndpoint
	if m.options.DashboardSource == "file" {
		dashboardEndpoint = m.options.DashboardPath
	}
	if dashboardEndpoint == "" {
		dashboardEndpoint = "not configured"
	}
	prometheusEndpoint := emptyDash(m.options.PrometheusEndpoint)

	rows := []string{
		apolloTheme.Section.Render("LINK STATUS"),
		statusRow("Dashboard source", emptyDash(m.options.DashboardSource)),
		statusRow("Dashboard endpoint", dashboardEndpoint),
		statusRow("Dashboard catalog", m.catalogBadge()),
		"",
		apolloTheme.Section.Render("PROMETHEUS"),
		statusRow("Endpoint", prometheusEndpoint),
		statusRow("Health probe", m.prometheusBadge()),
	}
	if m.healthError != nil {
		rows = append(rows, apolloTheme.Error.Render("  "+m.healthError.Error()))
	}
	return m.shell("Connection status", "LIVE SERVICE TELEMETRY", strings.Join(rows, "\n"), "r refresh checks   esc menu   q quit")
}

func (m Model) helpView() string {
	return m.shell("Help and shortcuts", "FIELD MANUAL", m.helpScroll.View(), "up/down scroll   esc menu   q quit")
}

func (m Model) helpContent() string {
	rows := []string{
		apolloTheme.Section.Render("HOME"),
		shortcutRow("j / k", "move through menu items"),
		shortcutRow("enter", "open the selected menu item"),
		shortcutRow("1 - 4", "jump directly to a menu item"),
		shortcutRow("q", "quit Apollo"),
		"",
		apolloTheme.Section.Render("DASHBOARD CATALOG"),
		shortcutRow("enter", "open the selected dashboard"),
		shortcutRow("l", "load a local JSON file or directory"),
		shortcutRow("r", "refresh the catalog"),
		shortcutRow("esc", "return to the home screen"),
		"",
		apolloTheme.Section.Render("DASHBOARD WORKSPACE"),
		shortcutRow("j / k", "select a panel"),
		shortcutRow("enter", "inspect a panel query"),
		shortcutRow("r", "refresh panel data"),
		shortcutRow("esc", "return to the dashboard catalog"),
		"",
		apolloTheme.Section.Render("QUERY DETAIL"),
		shortcutRow("h / l", "switch between query targets"),
		shortcutRow("up / down", "scroll the query result"),
	}
	return strings.Join(rows, "\n")
}

func (m Model) shell(title, subtitle, body, help string) string {
	width := m.viewWidth()
	header := lipgloss.JoinHorizontal(lipgloss.Center, apolloTheme.Brand.Render("APOLLO"), apolloTheme.Muted.Render("  /  "+strings.ToUpper(truncate(title, max(1, width-10)))))
	if subtitle != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.NewStyle().MaxWidth(max(1, width)).Render(subtitle))
	}
	bodyStyle := apolloTheme.Shell.Width(max(1, width-apolloTheme.Shell.GetHorizontalBorderSize()))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().MaxWidth(max(1, width)).Render(header),
		bodyStyle.Render(body),
		m.footer(help),
	)
}

func (m Model) menuStatus(width int) string {
	if width < 32 {
		return m.prometheusBadge()
	}
	if width < 48 {
		return lipgloss.JoinHorizontal(lipgloss.Center, apolloTheme.Muted.Render("CAT "), m.catalogBadge(), apolloTheme.Muted.Render("  PROM "), m.prometheusBadge())
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		apolloTheme.Muted.Render("CATALOG "),
		m.catalogBadge(),
		apolloTheme.Muted.Render("    PROMETHEUS "),
		m.prometheusBadge(),
	)
}

func (m Model) catalogStatus() string {
	if m.listLoading {
		return apolloTheme.Warning.Render("● CATALOG SYNC IN PROGRESS")
	}
	if m.listError != nil {
		return apolloTheme.Error.Render("● CATALOG UNAVAILABLE  ") + apolloTheme.Muted.Render(m.listError.Error())
	}
	if len(m.summaries) == 0 {
		return apolloTheme.Warning.Render("● CATALOG EMPTY  ") + apolloTheme.Muted.Render("Use l to load a local JSON path")
	}
	return apolloTheme.Success.Render(fmt.Sprintf("● %d DASHBOARDS READY", len(m.summaries)))
}

func (m Model) catalogBadge() string {
	if m.listLoading {
		return apolloTheme.Warning.Render("SYNCING")
	}
	if m.listError != nil {
		return apolloTheme.Error.Render("ERROR")
	}
	if len(m.summaries) == 0 {
		return apolloTheme.Warning.Render("EMPTY")
	}
	return apolloTheme.Success.Render("ONLINE")
}

func (m Model) prometheusBadge() string {
	if m.healthLoading {
		return apolloTheme.Warning.Render("CHECKING")
	}
	if m.healthError != nil {
		return apolloTheme.Error.Render("OFFLINE")
	}
	return apolloTheme.Success.Render("ONLINE")
}

func statusRow(label, value string) string {
	return fmt.Sprintf("%-22s %s", apolloTheme.Muted.Render(label), value)
}

func shortcutRow(key, description string) string {
	return lipgloss.JoinHorizontal(lipgloss.Center, apolloTheme.Key.Render(key), "  ", apolloTheme.Muted.Render(description))
}

func (m Model) footer(value string) string {
	return "\n" + apolloTheme.Muted.Render("// "+truncate(value, max(1, m.viewWidth()-3)))
}

func (m Model) viewWidth() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

func (m Model) viewHeight() int {
	if m.height <= 0 {
		return 24
	}
	return m.height
}

func (m Model) bodyContentWidth() int {
	return max(1, m.viewWidth()-apolloTheme.Shell.GetHorizontalFrameSize())
}

func (m Model) bodyContentHeight() int {
	return max(1, m.viewHeight()-8)
}

func (m Model) dashboardWidth() int {
	if m.dashboardScroll.Width > 0 {
		return m.dashboardScroll.Width
	}
	return m.bodyContentWidth()
}

func (m *Model) updateDashboardScroll() {
	m.dashboardScroll.SetContent(m.dashboardContent())
	if m.dashboard == nil || len(m.dashboard.Panels) == 0 {
		return
	}

	indices := m.orderedPanelIndices()
	offset := 0
	for start := 0; start < len(indices); {
		rowY := m.dashboard.Panels[indices[start]].GridPos.Y
		end := start
		for end < len(indices) && m.dashboard.Panels[indices[end]].GridPos.Y == rowY {
			end++
		}
		if m.viewWidth() < 90 && end-start > 1 {
			end = start + 1
		}

		widths := m.panelRowWidths(indices[start:end])
		cards := make([]string, 0, end-start)
		for position, index := range indices[start:end] {
			panel := m.dashboard.Panels[index]
			width := widths[position]
			cards = append(cards, renderPanel(*m, index, panel, width, panelHeight(panel.GridPos.H)))
		}
		rowHeight := lipgloss.Height(lipgloss.JoinHorizontal(lipgloss.Top, cards...))
		selected := false
		for _, index := range indices[start:end] {
			if index == m.selectedPanel {
				selected = true
				break
			}
		}
		if selected {
			viewportHeight := max(1, m.dashboardScroll.Height)
			if offset < m.dashboardScroll.YOffset {
				m.dashboardScroll.SetYOffset(offset)
			} else if rowHeight >= viewportHeight {
				if offset >= m.dashboardScroll.YOffset+viewportHeight {
					m.dashboardScroll.SetYOffset(offset)
				}
			} else if offset+rowHeight > m.dashboardScroll.YOffset+viewportHeight {
				m.dashboardScroll.SetYOffset(offset + rowHeight - viewportHeight)
			}
			return
		}
		offset += rowHeight
		if end < len(indices) {
			offset++
		}
		start = end
	}
}

func renderPanelRows(m Model) string {
	if m.dashboard == nil || len(m.dashboard.Panels) == 0 {
		return apolloTheme.Muted.Render("This dashboard has no panels.")
	}
	indices := m.orderedPanelIndices()
	var rows []string
	for start := 0; start < len(indices); {
		rowY := m.dashboard.Panels[indices[start]].GridPos.Y
		end := start
		for end < len(indices) && m.dashboard.Panels[indices[end]].GridPos.Y == rowY {
			end++
		}
		if m.viewWidth() < 90 && end-start > 1 {
			end = start + 1
		}
		cards := make([]string, 0, end-start)
		widths := m.panelRowWidths(indices[start:end])
		for position, index := range indices[start:end] {
			panel := m.dashboard.Panels[index]
			width := widths[position]
			height := panelHeight(panel.GridPos.H)
			cards = append(cards, renderPanel(m, index, panel, width, height))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
		start = end
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) panelRowWidths(indices []int) []int {
	if len(indices) == 0 {
		return nil
	}
	totalWidth := m.dashboardWidth()
	gridWidths := make([]int, len(indices))
	totalGridWidth := 0
	for index, panelIndex := range indices {
		gridWidth := m.dashboard.Panels[panelIndex].GridPos.W
		if m.viewWidth() < 90 {
			gridWidth = 24
		}
		if gridWidth <= 0 {
			gridWidth = 24 / len(indices)
		}
		gridWidths[index] = gridWidth
		totalGridWidth += gridWidth
	}
	if totalGridWidth <= 0 {
		totalGridWidth = len(indices)
	}

	widths := make([]int, len(indices))
	used := 0
	for index, gridWidth := range gridWidths {
		if index == len(widths)-1 {
			widths[index] = max(0, totalWidth-used)
		} else {
			widths[index] = totalWidth * gridWidth / totalGridWidth
			used += widths[index]
		}
	}
	return widths
}

func (m Model) orderedPanelIndices() []int {
	indices := make([]int, len(m.dashboard.Panels))
	for index := range indices {
		indices[index] = index
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left := m.dashboard.Panels[indices[i]].GridPos
		right := m.dashboard.Panels[indices[j]].GridPos
		if left.Y == right.Y {
			return left.X < right.X
		}
		return left.Y < right.Y
	})
	return indices
}

func (m *Model) movePanel(delta int) bool {
	if m.dashboard == nil || len(m.dashboard.Panels) == 0 {
		return false
	}
	indices := m.orderedPanelIndices()
	position := 0
	for index, panelIndex := range indices {
		if panelIndex == m.selectedPanel {
			position = index
			break
		}
	}
	next := min(len(indices)-1, max(0, position+delta))
	if indices[next] == m.selectedPanel {
		return false
	}
	m.selectedPanel = indices[next]
	m.selectedTarget = 0
	return true
}

func (m Model) queryContent() string {
	if m.dashboard == nil || m.selectedPanel < 0 || m.selectedPanel >= len(m.dashboard.Panels) {
		return "No query selected"
	}
	panel := m.dashboard.Panels[m.selectedPanel]
	if len(panel.Targets) == 0 || m.selectedTarget < 0 || m.selectedTarget >= len(panel.Targets) {
		return apolloTheme.Muted.Render("This panel does not contain a query target.")
	}
	target := panel.Targets[m.selectedTarget]
	key := queryKey(m.selectedPanel, m.selectedTarget)
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s\n", apolloTheme.Section.Render("TARGET "+emptyDash(target.RefID)))
	fmt.Fprintf(&builder, "PromQL: %s\n", dashboard.ExpandQuery(target.Expr, m.dashboard.Variables))
	fmt.Fprintf(&builder, "Legend: %s\n\n", emptyDash(target.LegendFormat))
	if reason := targetSkipReason(target); reason != "" {
		builder.WriteString(apolloTheme.Warning.Render(reason))
		return builder.String()
	}
	if err := m.queryErrors[key]; err != nil {
		builder.WriteString(apolloTheme.Error.Render(err.Error()))
		return builder.String()
	}
	result, ok := m.queryResults[key]
	if !ok {
		builder.WriteString(apolloTheme.Warning.Render("Loading query result..."))
		return builder.String()
	}
	builder.WriteString(renderResult(result, max(1, m.queryScroll.Width), max(1, m.queryScroll.Height-4), true))
	return builder.String()
}

func renderPanel(m Model, index int, panel dashboard.Panel, width, height int) string {
	if width < 4 {
		return truncate(panel.Title, width)
	}
	style := apolloTheme.Panel
	if index == m.selectedPanel {
		style = apolloTheme.PanelSelected
	}
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-3)
	content := apolloTheme.Muted.Render("No query targets")
	if panel.Text != "" {
		content = panel.Text
	}
	if len(panel.Targets) > 0 {
		target := panel.Targets[0]
		if reason := targetSkipReason(target); reason != "" {
			content = apolloTheme.Warning.Render(reason)
		} else {
			key := queryKey(index, 0)
			if err := m.queryErrors[key]; err != nil {
				content = apolloTheme.Error.Render(err.Error())
			} else if result, ok := m.queryResults[key]; ok {
				content = renderResult(result, innerWidth, innerHeight, isChartPanel(panel.Type))
			} else {
				content = apolloTheme.Warning.Render("Loading query...")
			}
		}
	}
	title := panel.Title
	if title == "" {
		title = "Untitled panel"
	}
	badge := apolloTheme.Badge.Render(strings.ToUpper(emptyDash(panel.Type)))
	heading := lipgloss.JoinHorizontal(lipgloss.Center, apolloTheme.Title.Render(truncate(title, max(12, width-14))), " ", badge)
	if len(panel.Targets) > 1 {
		heading = lipgloss.JoinHorizontal(lipgloss.Center, heading, " ", apolloTheme.Muted.Render(fmt.Sprintf("+%d targets", len(panel.Targets)-1)))
	}
	return style.Width(max(1, width-2)).Height(max(3, height-2)).Render(heading + "\n" + content)
}

func targetSkipReason(target dashboard.Target) string {
	if strings.TrimSpace(target.Expr) == "" {
		return "Skipped: no PromQL expression"
	}
	if !target.Datasource.IsPrometheus() {
		return "Skipped: non-Prometheus datasource"
	}
	return ""
}

func renderResult(result prometheus.Result, width, height int, chart bool) string {
	body := "No data"
	if chart && len(result.Series) > 0 {
		body = renderChart(result.Series, width, height)
	} else if result.Scalar != nil {
		body = fmt.Sprintf("%.4g", result.Scalar.Value)
	} else if result.Text != "" {
		body = result.Text
	} else if len(result.Series) > 0 {
		body = renderSeriesSummary(result.Series, width)
	}
	if len(result.Warnings) > 0 {
		body = apolloTheme.Warning.Render("Warnings: "+strings.Join(result.Warnings, "; ")) + "\n" + body
	}
	return body
}

func renderChart(series []prometheus.Series, width, height int) string {
	if width < 8 || height < 3 {
		return renderSeriesSummary(series, width)
	}
	options := make([]timeserieslinechart.Option, 0, 1)
	if width < 24 || height < 6 {
		options = append(options, timeserieslinechart.WithXYSteps(0, 0))
	}
	chart := timeserieslinechart.New(width, height, options...)
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

func renderSeriesSummary(series []prometheus.Series, width int) string {
	width = max(1, width)
	lines := make([]string, 0, len(series))
	for _, item := range series {
		if len(item.Samples) == 0 {
			continue
		}
		last := item.Samples[len(item.Samples)-1]
		value := fmt.Sprintf("%.4g", last.Value)
		if width <= lipgloss.Width(value) {
			lines = append(lines, truncate(value, width))
			continue
		}
		labelWidth := max(1, width-lipgloss.Width(value)-1)
		line := truncate(formatLabels(item.Labels), labelWidth) + " " + value
		lines = append(lines, truncate(line, width))
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

func panelHeight(gridHeight int) int {
	if gridHeight <= 0 {
		return 10
	}
	return max(7, min(18, gridHeight))
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for index := len(runes); index >= 0; index-- {
		candidate := string(runes[:index]) + "..."
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
	}
	return "..."
}

package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fakeSource struct {
	summary   dashboard.DashboardSummary
	dashboard dashboard.Dashboard
}

func (f fakeSource) List(context.Context) ([]dashboard.DashboardSummary, error) {
	return []dashboard.DashboardSummary{f.summary}, nil
}

func (f fakeSource) Get(context.Context, string) (dashboard.Dashboard, error) {
	return f.dashboard, nil
}

type fakeQuerier struct{}

func (fakeQuerier) Query(context.Context, prometheus.QueryRequest) (prometheus.Result, error) {
	return prometheus.Result{ResultType: "vector"}, nil
}

func TestModelNavigatesDashboardAndQueryScreens(t *testing.T) {
	dashboardValue := dashboard.Dashboard{
		DashboardSummary: dashboard.DashboardSummary{ID: "apollo", Title: "Apollo"},
		Time:             dashboard.TimeRange{From: "now-1h", To: "now"},
		Panels: []dashboard.Panel{{
			Title: "Requests",
			Type:  "timeseries",
			Targets: []dashboard.Target{{
				RefID: "A",
				Expr:  "up",
				Range: true,
			}},
		}},
	}
	m := New(fakeSource{
		summary:   dashboardValue.DashboardSummary,
		dashboard: dashboardValue,
	}, fakeQuerier{}, Options{RefreshInterval: time.Hour})

	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = update(t, m, dashboardsLoadedMsg{summaries: []dashboard.DashboardSummary{dashboardValue.DashboardSummary}})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, loadCmd := updateWithCmd(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != dashboardDetailScreen || loadCmd == nil {
		t.Fatalf("expected dashboard loading screen, got %d", m.screen)
	}
	m = update(t, m, loadCmd())
	if m.screen != dashboardDetailScreen || m.dashboard == nil {
		t.Fatalf("expected dashboard detail screen, got %d", m.screen)
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != queryScreen {
		t.Fatalf("expected query screen, got %d", m.screen)
	}
	m = update(t, m, queryResultMsg{
		key:        queryKey(0, 0),
		generation: m.generation,
		result: prometheus.Result{ResultType: "vector", Series: []prometheus.Series{{
			Labels:  map[string]string{"job": "demo"},
			Samples: []prometheus.Sample{{Timestamp: time.Now(), Value: 1}},
		}}},
	})
	if !strings.Contains(m.View(), "PromQL: up") {
		t.Fatalf("query view did not include query: %s", m.View())
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != dashboardDetailScreen {
		t.Fatalf("expected back navigation to dashboard, got %d", m.screen)
	}
}

func TestModelLoadsDashboardPathFromList(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = update(t, m, dashboardsLoadedMsg{})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = updateWithCmd(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !m.loadMode {
		t.Fatal("expected dashboard load prompt")
	}

	path := filepath.Join("..", "dashboard", "testdata", "classic.json")
	m, _ = updateWithCmd(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(path)})
	m, loadCmd := updateWithCmd(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.loadMode || loadCmd == nil {
		t.Fatalf("expected load command after entering dashboard path")
	}
	if _, ok := m.source.(*dashboard.FileSource); !ok {
		t.Fatalf("expected file source, got %T", m.source)
	}

	m = update(t, m, loadCmd())
	if m.listLoading || len(m.summaries) != 1 {
		t.Fatalf("expected one loaded dashboard, loading=%t summaries=%d", m.listLoading, len(m.summaries))
	}
}

func TestModelStartsOnMissionControlMenu(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 32})
	view := m.View()
	for _, value := range []string{"APOLLO", "Browse dashboards", "Load JSON path", "Connection status", "Help and shortcuts"} {
		if !strings.Contains(view, value) {
			t.Fatalf("home view did not include %q: %s", value, view)
		}
	}
}

func TestModelShowsConnectionStatus(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{
		DashboardSource:    "grafana",
		DashboardEndpoint:  "https://grafana.example.test",
		PrometheusEndpoint: "https://prometheus.example.test",
	})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = update(t, m, healthLoadedMsg{generation: 1})
	view := m.View()
	for _, value := range []string{"CONNECTION STATUS", "grafana.example.test", "prometheus.example.test", "ONLINE"} {
		if !strings.Contains(view, value) {
			t.Fatalf("connection view did not include %q: %s", value, view)
		}
	}
}

func TestModelRefreshAdvancesQueryGeneration(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m.screen = dashboardDetailScreen
	m.dashboard = &dashboard.Dashboard{
		DashboardSummary: dashboard.DashboardSummary{ID: "apollo", Title: "Apollo"},
		Panels: []dashboard.Panel{{
			Targets: []dashboard.Target{{Expr: "up", Range: true}},
		}},
	}
	m.generation = 4
	m.queryResults[queryKey(0, 0)] = prometheus.Result{ResultType: "vector"}
	m.queryErrors[queryKey(1, 0)] = context.Canceled

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	refreshed := updated.(Model)
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if refreshed.generation != 5 {
		t.Fatalf("expected query generation 5, got %d", refreshed.generation)
	}
	if len(refreshed.queryResults) != 0 || len(refreshed.queryErrors) != 0 {
		t.Fatal("expected refresh to clear previous query state")
	}
}

func TestModelMenuFitsNarrowTerminal(t *testing.T) {
	for _, height := range []int{16, 24, 32} {
		m := New(fakeSource{}, fakeQuerier{}, Options{})
		m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: height})
		assertViewBounds(t, m.View(), 40, height)
	}
}

func TestModelCentersMainMenu(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(line, "╭") {
			left := strings.Index(line, "╭")
			if left < 5 || left > 9 {
				t.Fatalf("menu shell is not centered: left=%d line=%q", left, line)
			}
			return
		}
	}
	t.Fatal("menu shell was not rendered")
}

func TestModelSecondaryScreensFitNarrowTerminal(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{
		DashboardSource:    "grafana",
		DashboardEndpoint:  "https://grafana.example.test",
		PrometheusEndpoint: "https://prometheus.example.test",
	})
	m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	for _, current := range []screen{connectionScreen, helpScreen} {
		m.screen = current
		assertViewBounds(t, m.View(), 40, 20)
	}
}

func TestModelDashboardScreensFitNarrowTerminal(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	m.screen = dashboardDetailScreen
	m.dashboard = &dashboard.Dashboard{
		DashboardSummary: dashboard.DashboardSummary{ID: "apollo", Title: "Apollo"},
		Panels: []dashboard.Panel{{
			Title:   "Request rate",
			Type:    "timeseries",
			GridPos: dashboard.GridPos{W: 24, H: 10},
			Targets: []dashboard.Target{{Expr: "up", Range: true}},
		}},
	}
	m.updateDashboardScroll()
	assertViewBounds(t, m.View(), 40, 20)

	m.screen = queryScreen
	m.queryScroll.SetContent(m.queryContent())
	assertViewBounds(t, m.View(), 40, 20)
}

func TestMenuSelectionRowsHaveStableWidth(t *testing.T) {
	selected := apolloTheme.MenuSelected.Width(24).Render("▸ 1 Browse dashboards")
	normal := apolloTheme.MenuItem.Width(24).Render("  2 Load JSON path")
	if selectedWidth, normalWidth := lipgloss.Width(selected), lipgloss.Width(normal); selectedWidth != normalWidth {
		t.Fatalf("menu selection widths differ: selected=%d normal=%d", selectedWidth, normalWidth)
	}
}

func TestRenderChartFitsRequestedWidth(t *testing.T) {
	series := []prometheus.Series{{
		Labels: map[string]string{"job": "apollo"},
		Samples: []prometheus.Sample{
			{Timestamp: time.Now().Add(-2 * time.Minute), Value: 1},
			{Timestamp: time.Now().Add(-time.Minute), Value: 2},
			{Timestamp: time.Now(), Value: 1},
		},
	}}
	for _, size := range []struct{ width, height int }{{8, 5}, {16, 5}, {24, 6}, {40, 8}} {
		assertViewWidth(t, renderChart(series, size.width, size.height), size.width)
	}
	if rendered := renderChart(series, 16, 5); strings.Contains(rendered, "job=") {
		t.Fatalf("compact graph fell back to a text summary: %q", rendered)
	}
}

func TestModelKeepsSelectedPanelVisible(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 15})
	m.screen = dashboardDetailScreen
	m.dashboard = &dashboard.Dashboard{
		DashboardSummary: dashboard.DashboardSummary{ID: "apollo", Title: "Apollo"},
		Panels: []dashboard.Panel{
			{Title: "First", GridPos: dashboard.GridPos{Y: 0, H: 10}},
			{Title: "Second", GridPos: dashboard.GridPos{Y: 10, H: 10}},
			{Title: "Third", GridPos: dashboard.GridPos{Y: 20, H: 10}},
		},
	}
	m.updateDashboardScroll()
	if m.dashboardScroll.YOffset != 0 {
		t.Fatalf("expected initial panel at top, got offset %d", m.dashboardScroll.YOffset)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.selectedPanel != 1 || m.dashboardScroll.YOffset == 0 {
		t.Fatalf("expected second panel to be selected and visible, panel=%d offset=%d", m.selectedPanel, m.dashboardScroll.YOffset)
	}
}

func TestPanelRowsFitViewportWidth(t *testing.T) {
	m := New(fakeSource{}, fakeQuerier{}, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m.dashboard = &dashboard.Dashboard{Panels: []dashboard.Panel{
		{Title: "Left", GridPos: dashboard.GridPos{X: 0, Y: 0, W: 24, H: 8}},
		{Title: "Right", GridPos: dashboard.GridPos{X: 24, Y: 0, W: 24, H: 8}},
	}}
	assertViewWidth(t, renderPanelRows(m), m.dashboardScroll.Width)
}

func assertViewWidth(t *testing.T, view string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if lineWidth := lipgloss.Width(line); lineWidth > width {
			t.Fatalf("view line is %d columns wide, want at most %d: %q", lineWidth, width, line)
		}
	}
}

func assertViewBounds(t *testing.T, view string, width, height int) {
	t.Helper()
	assertViewWidth(t, view, width)
	if lineCount := len(strings.Split(view, "\n")); lineCount > height {
		t.Fatalf("view is %d lines tall, want at most %d", lineCount, height)
	}
}

func update(t *testing.T, model Model, message tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(message)
	return updated.(Model)
}

func updateWithCmd(t *testing.T, model Model, message tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(message)
	return updated.(Model), cmd
}

package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	tea "github.com/charmbracelet/bubbletea"
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

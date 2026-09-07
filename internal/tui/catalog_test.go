package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSortedSummaries(t *testing.T) {
	summaries := []dashboard.DashboardSummary{
		{ID: "zeta", Title: "Aardvark", Source: "grafana"},
		{ID: "alpha", Title: "Alpha", Source: "file", Starred: true},
		{ID: "beta", Title: "Beta", Source: "grafana", Starred: true},
	}

	tests := []struct {
		name string
		mode catalogSort
		want []string
	}{
		{name: "title", mode: titleSort, want: []string{"zeta", "alpha", "beta"}},
		{name: "starred", mode: starredSort, want: []string{"alpha", "beta", "zeta"}},
		{name: "source", mode: sourceSort, want: []string{"alpha", "zeta", "beta"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sortedSummaries(summaries, test.mode)
			for index, id := range test.want {
				if got[index].ID != id {
					t.Fatalf("summary %d: got %q, want %q", index, got[index].ID, id)
				}
			}
		})
	}

	if summaries[0].ID != "zeta" {
		t.Fatal("sorting mutated the input summaries")
	}
}

func TestCatalogFilterValueIncludesMetadata(t *testing.T) {
	value := catalogFilterValue(dashboard.DashboardSummary{
		ID:        "/tmp/traffic.json",
		UID:       "traffic",
		Title:     "Traffic Overview",
		Tags:      []string{"edge", "slo"},
		FolderUID: "platform",
		Source:    "file",
	})

	for _, expected := range []string{"Traffic Overview", "traffic", "LOCAL FILE", "platform", "edge", "traffic.json"} {
		if !strings.Contains(value, expected) {
			t.Errorf("filter value %q does not contain %q", value, expected)
		}
	}
}

func TestCatalogRefreshPreservesSelectionAndStaleItems(t *testing.T) {
	first := dashboard.DashboardSummary{ID: "first", Title: "First"}
	second := dashboard.DashboardSummary{ID: "second", Title: "Second"}
	m := New(nil, nil, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(t, m, dashboardsLoadedMsg{summaries: []dashboard.DashboardSummary{first, second}})
	m.list.Select(1)

	m = update(t, m, dashboardsLoadedMsg{summaries: []dashboard.DashboardSummary{second, first}})
	selected, ok := m.list.SelectedItem().(dashboardItem)
	if !ok || selected.summary.ID != second.ID {
		t.Fatalf("selection was not preserved, got %#v", m.list.SelectedItem())
	}

	m.listLoading = true
	m = update(t, m, dashboardsLoadedMsg{err: errors.New("source offline")})
	if len(m.summaries) != 2 || len(m.list.Items()) != 2 || m.listError == nil {
		t.Fatalf("refresh error did not preserve catalog: summaries=%d items=%d error=%v", len(m.summaries), len(m.list.Items()), m.listError)
	}
}

func TestCatalogSearchOwnsInputAndClearsAppliedFilter(t *testing.T) {
	m := New(nil, nil, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = update(t, m, dashboardsLoadedMsg{summaries: []dashboard.DashboardSummary{{ID: "one", Title: "One"}}})
	m.screen = dashboardListScreen

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.list.FilterState() != list.Filtering || m.list.FilterInput.Value() != "q" {
		t.Fatalf("search input did not receive key, state=%s value=%q", m.list.FilterState(), m.list.FilterInput.Value())
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.list.FilterInput.Value() != "" {
		t.Fatalf("backspace did not edit search input: %q", m.list.FilterInput.Value())
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.list.FilterState() != list.Unfiltered || m.screen != dashboardListScreen {
		t.Fatalf("escape did not cancel search, state=%s screen=%d", m.list.FilterState(), m.screen)
	}
}

func TestCatalogViewShowsPreviewAndFitsNarrowTerminal(t *testing.T) {
	summary := dashboard.DashboardSummary{
		ID:        "traffic-id",
		UID:       "traffic",
		Title:     "Traffic Overview",
		Tags:      []string{"edge", "slo"},
		FolderUID: "platform",
		Source:    "grafana",
		URL:       "https://grafana.example.test/d/traffic",
	}
	m := New(nil, nil, Options{})
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(t, m, dashboardsLoadedMsg{summaries: []dashboard.DashboardSummary{summary}})
	m.screen = dashboardListScreen
	view := m.View()
	for _, expected := range []string{"SELECTED DASHBOARD", "Traffic Overview", "traffic", "platform", "edge, slo"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("catalog view does not contain %q: %s", expected, view)
		}
	}
	assertViewBounds(t, view, 120, 30)

	m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 20})
	assertViewBounds(t, m.View(), 40, 20)
	if lipgloss.Width(m.catalogPreview(30, 8)) > 30 {
		t.Fatal("preview exceeded its requested width")
	}
}

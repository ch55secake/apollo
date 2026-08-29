package dashboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseClassicDashboard(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "classic.json"))
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.UID != "apollo-classic" || parsed.Title != "Apollo Classic" {
		t.Fatalf("unexpected dashboard identity: %+v", parsed.DashboardSummary)
	}
	if len(parsed.Panels) != 2 {
		t.Fatalf("expected flattened panels, got %d", len(parsed.Panels))
	}
	if parsed.Panels[1].Row != "Rows" || parsed.Panels[1].Datasource.Name != "Prometheus" {
		t.Fatalf("unexpected row panel: %+v", parsed.Panels[1])
	}
	if got := ExpandQuery(parsed.Panels[0].Targets[0].Expr, parsed.Variables); got != `rate(http_requests_total{job="api"}[5m])` {
		t.Fatalf("unexpected expanded query: %s", got)
	}
}

func TestParseResourceDashboard(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.UID != "apollo-resource" || len(parsed.Panels) != 1 {
		t.Fatalf("unexpected resource dashboard: %+v", parsed)
	}
}

func TestTimeRangeResolve(t *testing.T) {
	now := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	start, end, err := (TimeRange{From: "now-2h", To: "now"}).Resolve(now)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(now.Add(-2*time.Hour)) || !end.Equal(now) {
		t.Fatalf("unexpected time range: %s to %s", start, end)
	}
}

func TestTimeRangeRejectsInvalidOrder(t *testing.T) {
	_, _, err := (TimeRange{From: "now", To: "now-1h"}).Resolve(time.Now())
	if err == nil {
		t.Fatal("expected invalid time range to fail")
	}
}

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

func TestParsePanelLegendAndFixedColors(t *testing.T) {
	parsed, err := Parse([]byte(`{
        "title": "Visuals",
        "panels": [{
            "type": "timeseries",
            "options": {"legend": {"showLegend": false, "placement": "right"}},
            "fieldConfig": {
                "defaults": {"color": {"mode": "fixed", "fixedColor": "blue"}},
                "overrides": [{
                    "matcher": {"id": "byName", "options": "api-1"},
                    "properties": [{"id": "color", "value": {"mode": "fixed", "fixedColor": "red"}}]
                }]
            }
        }]
    }`))
	if err != nil {
		t.Fatal(err)
	}
	panel := parsed.Panels[0]
	if panel.Legend.Show || !panel.Legend.ShowSet || panel.Legend.Placement != "right" {
		t.Fatalf("unexpected legend settings: %+v", panel.Legend)
	}
	if panel.Color != "blue" || len(panel.ColorOverrides) != 1 || panel.ColorOverrides[0] != (ColorOverride{Name: "api-1", Color: "red"}) {
		t.Fatalf("unexpected color settings: %+v %+v", panel.Color, panel.ColorOverrides)
	}
}

func TestParsePanelStringColorOverride(t *testing.T) {
	parsed, err := Parse([]byte(`{"title":"Visuals","panels":[{"fieldConfig":{"overrides":[{"matcher":{"id":"byName","options":"api-1"},"properties":[{"id":"color","value":"semi-dark-green"}]}]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Panels[0].ColorOverrides; len(got) != 1 || got[0].Color != "semi-dark-green" {
		t.Fatalf("unexpected string color override: %+v", got)
	}
}

func TestParseVariableOptions(t *testing.T) {
	parsed, err := Parse([]byte(`{"title":"Variables","templating":{"list":[{"name":"job","type":"custom","current":{"value":"api"},"options":[{"text":"API","value":"api"},{"text":"Worker","value":"worker"}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	variable := parsed.Variables[0]
	if len(variable.Values) != 2 || variable.Values[0] != "api" || variable.Values[1] != "worker" {
		t.Fatalf("unexpected variable options: %+v", variable)
	}
}

func TestParseObjectFormVariableQuery(t *testing.T) {
	parsed, err := Parse([]byte(`{
        "title": "Queries",
        "templating": {
            "list": [
                {
                    "name": "classic",
                    "type": "custom",
                    "current": {"value": "api"},
                    "query": "label_values(up, job)"
                },
                {
                    "name": "modern",
                    "type": "query",
                    "current": {"value": "api"},
                    "query": {"query": "label_values(up, job)", "refId": "StandardVariableQuery"}
                },
                {
                    "name": "expr",
                    "type": "query",
                    "current": {"value": "api"},
                    "query": {"expr": "label_values(up, job)", "editorMode": 1}
                },
                {
                    "name": "unsupported",
                    "type": "query",
                    "current": {"value": "api"},
                    "query": {"keys": [], "refId": "x"}
                }
            ]
        }
    }`))
	if err != nil {
		t.Fatalf("dashboard with object templating queries should parse: %v", err)
	}
	byName := make(map[string]string)
	for _, variable := range parsed.Variables {
		byName[variable.Name] = variable.Query
	}
	if got := byName["classic"]; got != "label_values(up, job)" {
		t.Fatalf("unexpected classic string query: %q", got)
	}
	if got := byName["modern"]; got != "label_values(up, job)" {
		t.Fatalf("unexpected object query value: %q", got)
	}
	if got := byName["expr"]; got != "label_values(up, job)" {
		t.Fatalf("unexpected expr query value: %q", got)
	}
	if got := byName["unsupported"]; got != "" {
		t.Fatalf("expected empty query for unsupported shape, got %q", got)
	}
}

func TestParsePanelUnitAndThresholds(t *testing.T) {
	parsed, err := Parse([]byte(`{
        "title": "Units",
        "panels": [{
            "type": "stat",
            "fieldConfig": {
                "defaults": {
                    "unit": "percent",
                    "thresholds": {
                        "mode": "absolute",
                        "steps": [
                            {"color": "green", "value": null},
                            {"color": "orange", "value": 70},
                            {"color": "red", "value": 90}
                        ]
                    }
                }
            }
        }]
    }`))
	if err != nil {
		t.Fatal(err)
	}
	panel := parsed.Panels[0]
	if panel.Unit != "percent" {
		t.Fatalf("unexpected unit: %q", panel.Unit)
	}
	if len(panel.Thresholds) != 3 {
		t.Fatalf("unexpected thresholds: %+v", panel.Thresholds)
	}
	if panel.Thresholds[0].Color != "green" || panel.Thresholds[0].Value != nil {
		t.Fatalf("unexpected base threshold: %+v", panel.Thresholds[0])
	}
	orange := 70.0
	if panel.Thresholds[1].Color != "orange" || panel.Thresholds[1].Value == nil || *panel.Thresholds[1].Value != orange {
		t.Fatalf("unexpected orange threshold: %+v", panel.Thresholds[1])
	}
}

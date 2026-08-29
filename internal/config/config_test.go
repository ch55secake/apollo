package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	contents := []byte(`
prometheus:
  url: https://prometheus.example.test
dashboards:
  source: grafana
  grafana:
    url: https://grafana.example.test
ui:
  refresh_interval: 45s
  default_time_range: now-2h
  max_data_points: 240
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Prometheus.URL != "https://prometheus.example.test" {
		t.Fatalf("unexpected Prometheus URL: %s", cfg.Prometheus.URL)
	}
	if cfg.Dashboards.Source != "grafana" {
		t.Fatalf("unexpected dashboard source: %s", cfg.Dashboards.Source)
	}
	if cfg.UI.RefreshInterval != 45*time.Second || cfg.UI.MaxDataPoints != 240 {
		t.Fatalf("unexpected UI config: %+v", cfg.UI)
	}
}

func TestLoadMissingExplicitConfig(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected missing explicit config to fail")
	}
}

func TestConfigValidateRejectsUnknownSource(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("dashboards:\n  source: unknown\n")); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Prometheus.URL = "http://localhost:9090"
	cfg.UI.RefreshInterval = time.Second
	cfg.UI.DefaultTimeRange = "now-1h"
	cfg.UI.MaxDataPoints = 10
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid source to fail")
	}
}

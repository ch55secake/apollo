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

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(configHome, "apollo", "config.yaml")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}

func TestLoadConfigFromXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Chdir(t.TempDir())
	path := filepath.Join(configHome, "apollo", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("prometheus:\n  url: https://prometheus.example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Prometheus.URL != "https://prometheus.example.test" {
		t.Fatalf("unexpected Prometheus URL: %s", cfg.Prometheus.URL)
	}
}

func TestInitConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := Init("")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(configHome, "apollo", "config.yaml")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != defaultConfig {
		t.Fatalf("unexpected initialized config:\n%s", contents)
	}
	if _, err := Init(""); err == nil {
		t.Fatal("expected initializing an existing config to fail")
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

package main

import (
	"fmt"
	"os"

	"github.com/ch55secake/apollo/internal/config"
	"github.com/ch55secake/apollo/internal/dashboard"
	"github.com/ch55secake/apollo/internal/prometheus"
	"github.com/ch55secake/apollo/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("apollo", pflag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.StringP("config", "c", "", "path to the Apollo configuration file")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			return
		}
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apollo: %v\n", err)
		os.Exit(1)
	}

	source, err := dashboardSource(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apollo: %v\n", err)
		os.Exit(1)
	}
	querier, err := prometheus.NewClient(cfg.Prometheus.URL, cfg.Prometheus.BearerToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apollo: %v\n", err)
		os.Exit(1)
	}

	model := tui.New(source, querier, tui.Options{
		RefreshInterval:  cfg.UI.RefreshInterval,
		DefaultTimeRange: cfg.UI.DefaultTimeRange,
		MaxDataPoints:    cfg.UI.MaxDataPoints,
	})
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "apollo: %v\n", err)
		os.Exit(1)
	}
}

func dashboardSource(cfg config.Config) (dashboard.Source, error) {
	switch cfg.Dashboards.Source {
	case "file":
		return dashboard.NewFileSource(cfg.Dashboards.Path)
	case "grafana":
		return dashboard.NewGrafanaSource(cfg.Dashboards.Grafana.URL, cfg.Dashboards.Grafana.BearerToken)
	default:
		return nil, fmt.Errorf("unsupported dashboard source %q", cfg.Dashboards.Source)
	}
}

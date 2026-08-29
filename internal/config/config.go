package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Dashboards DashboardConfig  `mapstructure:"dashboards"`
	UI         UIConfig         `mapstructure:"ui"`
}

type PrometheusConfig struct {
	URL         string `mapstructure:"url"`
	BearerToken string `mapstructure:"bearer_token"`
}

type DashboardConfig struct {
	Source  string        `mapstructure:"source"`
	Path    string        `mapstructure:"path"`
	Grafana GrafanaConfig `mapstructure:"grafana"`
}

type GrafanaConfig struct {
	URL         string `mapstructure:"url"`
	BearerToken string `mapstructure:"bearer_token"`
}

type UIConfig struct {
	RefreshInterval  time.Duration `mapstructure:"refresh_interval"`
	DefaultTimeRange string        `mapstructure:"default_time_range"`
	MaxDataPoints    int           `mapstructure:"max_data_points"`
}

func Load(configPath string) (Config, error) {
	v := newViper()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "apollo"))
		}
		if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
			v.AddConfigPath(filepath.Join(configHome, "apollo"))
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if configPath != "" || !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.UnmarshalExact(
		&cfg,
		viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc()),
	); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetEnvPrefix("APOLLO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	v.SetDefault("prometheus.url", "http://localhost:9090")
	v.SetDefault("dashboards.source", "file")
	v.SetDefault("dashboards.path", "./dashboards")
	v.SetDefault("ui.refresh_interval", "30s")
	v.SetDefault("ui.default_time_range", "now-6h")
	v.SetDefault("ui.max_data_points", 120)

	for _, key := range []string{
		"prometheus.url",
		"prometheus.bearer_token",
		"dashboards.source",
		"dashboards.path",
		"dashboards.grafana.url",
		"dashboards.grafana.bearer_token",
		"ui.refresh_interval",
		"ui.default_time_range",
		"ui.max_data_points",
	} {
		_ = v.BindEnv(key)
	}
	return v
}

func (c Config) Validate() error {
	if err := validateURL("prometheus.url", c.Prometheus.URL); err != nil {
		return err
	}
	if c.Dashboards.Source != "file" && c.Dashboards.Source != "grafana" {
		return fmt.Errorf("dashboards.source must be file or grafana, got %q", c.Dashboards.Source)
	}
	if c.Dashboards.Source == "file" && strings.TrimSpace(c.Dashboards.Path) == "" {
		return errors.New("dashboards.path must not be empty for the file source")
	}
	if c.Dashboards.Source == "grafana" {
		if err := validateURL("dashboards.grafana.url", c.Dashboards.Grafana.URL); err != nil {
			return err
		}
	}
	if c.UI.RefreshInterval <= 0 {
		return errors.New("ui.refresh_interval must be greater than zero")
	}
	if strings.TrimSpace(c.UI.DefaultTimeRange) == "" {
		return errors.New("ui.default_time_range must not be empty")
	}
	if c.UI.MaxDataPoints < 2 {
		return errors.New("ui.max_data_points must be at least 2")
	}
	return nil
}

func validateURL(name, raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute http or https URL", name)
	}
	return nil
}

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

const defaultConfigFile = "config.yaml"

const defaultConfig = `prometheus:
  url: http://localhost:9090
  bearer_token: ""

dashboards:
  source: file
  path: ./dashboards
  grafana:
    url: http://localhost:3000
    bearer_token: ""

ui:
  refresh_interval: 30s
  default_time_range: now-6h
  max_data_points: 120
`

func Load(configPath string) (Config, error) {
	v := newViper()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		if path, err := DefaultPath(); err == nil {
			v.AddConfigPath(filepath.Dir(path))
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

func DefaultPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "apollo", defaultConfigFile), nil
}

func Init(configPath string) (string, error) {
	if configPath == "" {
		var err error
		configPath, err = DefaultPath()
		if err != nil {
			return "", err
		}
	}
	configPath = filepath.Clean(configPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("config file already exists: %s", configPath)
		}
		return "", fmt.Errorf("create config file: %w", err)
	}
	if _, err := file.WriteString(defaultConfig); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write config file: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close config file: %w", err)
	}
	return configPath, nil
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

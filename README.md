# apollo

Apollo is a dashboarding TUI that uses Prometheus as a data source and accepts
Grafana dashboard JSON.

## Quick start

Copy `config/apollo.example.yaml` to `config.yaml`, configure the Prometheus
URL, and point `dashboards.path` at a Grafana JSON file or directory:

```sh
go run ./cmd/apollo --config config.yaml
```

From the dashboard list, press `l` to load a dashboard JSON file or directory
interactively. Use `enter` to open a listed dashboard.

To load dashboards from Grafana instead, set `dashboards.source` to `grafana`
and configure `dashboards.grafana.url` and its bearer token.

Once Apollo is available from the repository's default branch, run it directly
from the flake:

```sh
nix run ch55secake/apollo
```

Use the dashboard list to filter and select a dashboard. From a dashboard,
use `j` and `k` to select a panel and `enter` to inspect its PromQL query and
result. Press `r` to refresh and `esc` to go back.

Apollo currently supports Grafana classic dashboards, Grafana resource
dashboards with a `spec` payload, Prometheus matrix/vector/scalar results, and
basic time-series, stat, table, and text panel rendering. Unsupported Grafana
features are left as raw panel data or shown as placeholders.

## Development

```sh
make check
make compile
./bin/apollo --config config.yaml
nix develop
```

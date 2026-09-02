# apollo

Apollo is a dashboarding TUI that uses Prometheus as a data source and accepts
Grafana dashboard JSON.

## Quick start

Initialize the default user configuration at
`$XDG_CONFIG_HOME/apollo/config.yaml` (or `~/.config/apollo/config.yaml`):

```sh
apollo init
```

Edit the Prometheus URL and point `dashboards.path` at a Grafana JSON file or
directory, then start Apollo:

```sh
apollo
```

Use `--config PATH` to load a specific configuration file. The current working
directory's `config.yaml` takes precedence over the XDG configuration.

Apollo opens on a command menu. Choose `Browse dashboards` to open the catalog,
`Load JSON path` to switch to a local dashboard file or directory, `Connection
status` to inspect the configured links, or `Help and shortcuts` for the full
key reference. The number keys `1` through `4` select the first four actions
directly.

To load dashboards from Grafana instead, set `dashboards.source` to `grafana`
and configure `dashboards.grafana.url` and its bearer token.

Once Apollo is available from the repository's default branch, run it directly
from the flake:

```sh
nix run ch55secake/apollo
```

The init command is also available through the flake:

```sh
nix run ch55secake/apollo -- init
```

Use the dashboard catalog to filter and select a dashboard. From a dashboard,
use `j` and `k` to select a panel and `enter` to inspect its PromQL query and
result. In query detail, use `h` and `l` to switch targets. Press `r` to
refresh, `esc` to go back, and `q` to quit from any screen.

Apollo currently supports Grafana classic dashboards, Grafana resource
dashboards with a `spec` payload, Prometheus matrix/vector/scalar results, and
basic time-series, stat, table, and text panel rendering. Unsupported Grafana
features are left as raw panel data or shown as placeholders.

## Development

```sh
make check
make compile VERSION=0.1.0
./bin/apollo
nix develop
```

Check the build version with:

```sh
./bin/apollo --version
```

Pushing to `main` creates a release with Linux and macOS binaries for amd64 and
arm64. The release workflow currently starts at `v0.1.0`; update its `version`
and `ldflags` values together for future version changes.

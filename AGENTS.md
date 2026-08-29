# Agent Instructions

## Project

- This is the Go module `github.com/ch55secake/apollo`, targeting Go `1.26.5`.
- The intended program is a dashboarding TUI similar to Grafana, using Prometheus as a data source.
- The executable entrypoint is `cmd/apollo`; dashboard, data source, and orchestration packages should live under `internal/` as the project develops.
- Dashboard definitions are normalized under `internal/dashboard`, Prometheus access lives under `internal/prometheus`, and Bubble Tea models live under `internal/tui`.
- The initial UI uses the Bubble Tea v1 ecosystem because the selected `ntcharts` release depends on Bubble Tea v1; keep those package versions compatible when updating dependencies.

## Verification

- Run `go mod tidy` after changing module dependencies.
- Run `make build` and `make test` for build and test verification; these wrap the commands used by shared CI.
- Use `nix develop` for the flake-provided Go development shell and `nix flake check --no-build` to evaluate flake outputs.
- The Makefile provides `make build`, `make test`, `make tidy`, `make check`, and `make dev-shell` equivalents.

## GitHub Automation

- `.github/workflows/ci.yml` calls reusable Go build/test, lint, and vulnerability workflows through `ch55secake/cheesecake-factory`.
- `.github/workflows/pr-labeler.yml` uses `pull_request_target` because labeling requires `pull-requests: write`; rules are in `.github/labeler.yml`.
- `.github/dependabot.yml` checks the Go module and GitHub Actions daily.
- Labeling expects feature code in `cmd/**` or `internal/**`, tests in `tests/**` or `**/*_test.go`, and documentation in `README.md` or `AGENTS.md`.

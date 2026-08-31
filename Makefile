GO ?= go

.PHONY: build compile test tidy check dev-shell

build:
	$(GO) build ./...

compile:
	mkdir -p bin
	$(GO) build -o bin/apollo ./cmd/apollo

test:
	@set -e; \
	packages="$$($(GO) list ./...)"; \
	if [ -n "$$packages" ]; then \
		$(GO) test ./...; \
	else \
		echo "No Go packages to test"; \
	fi

tidy:
	$(GO) mod tidy

check: build test

dev-shell:
	nix develop

# Release version: the current git description without a leading "v".
VERSION ?= $(shell git describe --tags --always --dirty | sed -e 's/^v//')

MAKEFILE_PATH := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MOD := $(MAKEFILE_PATH)/tools/go.mod
GO_TOOL := go tool -modfile=$(TOOLS_MOD)
GOLANGCI_LINT ?= $(GO_TOOL) golangci-lint
GORELEASER ?= $(GO_TOOL) goreleaser
LOCALBIN ?= $(MAKEFILE_PATH)/bin

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: precommit

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build inflate for the current platform.
	mkdir -p $(LOCALBIN)
	CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(LOCALBIN)/inflate ./cmd

.PHONY: run
run: ## Run inflate from source.
	go run ./cmd

.PHONY: test
test: ## Run unit tests.
	go test -race -coverprofile=cover.out ./...

.PHONY: verify
verify: release-check ## Run linters and vetting without mutating files.
	$(GOLANGCI_LINT) config verify
	$(GOLANGCI_LINT) run
	go vet ./...

.PHONY: fix
fix: ## Format code and tidy modules.
	go fmt ./...
	go mod tidy
	cd tools && go mod tidy
	$(GOLANGCI_LINT) run --fix

.PHONY: precommit
precommit: fix verify test release-snapshot ## Run all local precommit checks.

##@ Tooling

.PHONY: tools
tools: ## Download pinned Go tools.
	go mod download -modfile=$(TOOLS_MOD)

##@ CI and Release

.PHONY: ci-verify
ci-verify: precommit ## Run precommit and fail if it changes tracked files.
	@git diff --exit-code || { \
		echo; \
		echo "ERROR: working tree is dirty after 'make ci-verify'."; \
		echo "Run 'make fix' locally and commit the result."; \
		exit 1; \
	}

.PHONY: release-check
release-check: tools
	$(GORELEASER) check

.PHONY: release-snapshot
release-snapshot: tools ## Build release artifacts locally without publishing.
	$(GORELEASER) release --snapshot --clean

.PHONY: release
release: tools ## Publish a tagged GitHub release.
	$(GORELEASER) release --clean

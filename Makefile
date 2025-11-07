BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOLANGCI_LINT_VERSION := v2.6.1
GCI := $(BIN_DIR)/gci
DOCKER_COMPOSE ?= docker compose

.PHONY: build
build:
	go build ./...

.PHONY: run
run:
	go run ./cmd/server

.PHONY: lint
lint: tools
	$(GOLANGCI_LINT) run

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race -cover ./...

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: bench
bench:
	go test -bench=. -benchmem ./internal/calculator

.PHONY: fmt
fmt: fmt-imports
	@files=$$(find . -name '*.go' -not -path './vendor/*' -not -path './test/testdata/*'); \
	if [ -n "$$files" ]; then \
		gofmt -w $$files; \
	fi

.PHONY: fmt-imports
# fmt-imports formats imports according to .golangci.yml configuration:
# 1. Standard library imports
# 2. External dependencies
# 3. Internal imports (matching github.com/eugenenazirov/re-partners prefix)
# Note: gci uses --section flags (configured in .golangci.yml for golangci-lint)
fmt-imports: tools
	@files=$$(find . -name '*.go' -not -path './vendor/*' -not -path './test/testdata/*'); \
	if [ -n "$$files" ]; then \
		$(GCI) write --section standard --section default --section "Prefix(github.com/eugenenazirov/re-partners)" --skip-generated --skip-vendor $$files; \
	fi

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin build dist

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GCI)

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) into $(BIN_DIR)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

$(GCI):
	@mkdir -p $(BIN_DIR)
	@echo "Installing gci into $(BIN_DIR)..."
	@GOBIN=$(BIN_DIR) go install github.com/daixiang0/gci@latest

.PHONY: compose-up
compose-up:
	$(DOCKER_COMPOSE) up --build

.PHONY: compose-down
compose-down:
	$(DOCKER_COMPOSE) down

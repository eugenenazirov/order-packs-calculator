BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOLANGCI_LINT_VERSION := v2.6.1
DOCKER_COMPOSE ?= docker compose
export GOCACHE ?= $(CURDIR)/.cache/go-build
export GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint

.PHONY: build
build:
	go build ./...

.PHONY: run
run: config-check
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
# fmt formats code using golangci-lint --fix, which handles both gofmt and gci (import formatting)
# This ensures consistency between make fmt and make lint commands
fmt: tools
	$(GOLANGCI_LINT) run --fix

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin build dist

.PHONY: config-check
config-check:
	@if [ ! -f config.yaml ]; then \
		echo "❌ Error: config.yaml not found!"; \
		echo ""; \
		echo "Please create it from the example:"; \
		echo "  cp config.yaml.example config.yaml"; \
		echo ""; \
		echo "Then customize it for your environment."; \
		exit 1; \
	fi
	@echo "✓ config.yaml found"

.PHONY: tools
tools: $(GOLANGCI_LINT)

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) into $(BIN_DIR)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

.PHONY: compose-up
compose-up:
	$(DOCKER_COMPOSE) up --build

.PHONY: compose-down
compose-down:
	$(DOCKER_COMPOSE) down

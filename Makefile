BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GOLANGCI_LINT_VERSION := v2.6.1

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
fmt:
	@files=$$(find . -name '*.go' -not -path './vendor/*' -not -path './test/testdata/*'); \
	if [ -n "$$files" ]; then \
		gofmt -w $$files; \
	fi

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin build dist

.PHONY: tools
tools: $(GOLANGCI_LINT)

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) into $(BIN_DIR)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

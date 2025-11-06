.PHONY: build run lint test fmt tidy clean

build:
	go build ./...

run:
	go run ./cmd/server

lint:
	golangci-lint run

test:
	go test ./...

fmt:
	@files=$$(find . -name '*.go' -not -path './vendor/*' -not -path './test/testdata/*'); \
	if [ -n "$$files" ]; then \
		gofmt -w $$files; \
	fi

tidy:
	go mod tidy

clean:
	rm -rf bin build dist

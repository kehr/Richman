.PHONY: dev dev-run dev-raw build lint test test-race test-cover sqlc migrate-up migrate-down docker-build check clean install-tools

HUMANLOG := $(shell go env GOPATH)/bin/humanlog
AIR      := $(shell go env GOPATH)/bin/air

# install-tools: install dev tools pinned to specific versions.
# Run once after cloning on a new machine: make install-tools
install-tools:
	go install github.com/air-verse/air@v1.65.0
	go install github.com/humanlogio/humanlog/cmd/humanlog@v0.8.9
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# dev: migrate then start with hot-reload, piped through humanlog for readable colored output.
# Requires: make install-tools
dev: migrate-up
	@if [ -x "$(AIR)" ]; then \
		$(AIR) 2>&1 | $(HUMANLOG) --time-fields ts --level-fields level --message-fields msg --time-format "2006-01-02 15:04:05"; \
	else \
		echo "air not found, falling back to go run (run: make install-tools)"; \
		go run ./cmd/server/main.go 2>&1 | $(HUMANLOG) --time-fields ts --level-fields level --message-fields msg --time-format "2006-01-02 15:04:05"; \
	fi

# dev-raw: same as dev but raw JSON output, no humanlog pipe.
# Useful when humanlog mangles output or you need to pipe elsewhere (e.g. jq).
dev-raw: migrate-up
	@if [ -x "$(AIR)" ]; then \
		$(AIR); \
	else \
		go run ./cmd/server/main.go; \
	fi

build:
	go build -o bin/server ./cmd/server/main.go

lint:
	golangci-lint run ./...
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...

sqlc:
	sqlc generate

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

docker-build:
	docker build -t richman-api .

check: lint test build

clean:
	rm -rf bin/

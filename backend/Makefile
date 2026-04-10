.PHONY: dev dev-run dev-pretty build lint test test-race test-cover sqlc migrate-up migrate-down docker-build check clean

# dev: migrate first, then start with air hot-reload if available, else plain go run.
# Logs are JSON. For human-readable output run: make dev-pretty
dev: migrate-up dev-run

dev-run:
	@AIR=$$(go env GOPATH)/bin/air; \
	if [ -x "$$AIR" ]; then \
		$$AIR; \
	else \
		echo "air not found, falling back to go run (install: go install github.com/air-verse/air@latest)"; \
		go run ./cmd/server/main.go; \
	fi

# dev-pretty: same as dev but routes logs through humanlog for readable colored output.
# Install humanlog once: go install github.com/humanlogio/humanlog/cmd/humanlog@latest
dev-pretty: migrate-up
	@HUMANLOG=$$(go env GOPATH)/bin/humanlog; \
	if [ ! -x "$$HUMANLOG" ]; then \
		echo "humanlog not found. Install with:"; \
		echo "  go install github.com/humanlogio/humanlog/cmd/humanlog@latest"; \
		echo "Falling back to plain dev output..."; \
		$(MAKE) dev-run; \
	else \
		AIR=$$(go env GOPATH)/bin/air; \
		if [ -x "$$AIR" ]; then \
			$$AIR 2>&1 | $$HUMANLOG --skip-unknown-fields; \
		else \
			go run ./cmd/server/main.go 2>&1 | $$HUMANLOG --skip-unknown-fields; \
		fi \
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

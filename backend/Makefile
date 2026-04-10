.PHONY: dev dev-run build lint test test-race test-cover sqlc migrate-up migrate-down docker-build check clean

# dev: migrate first, then start with air hot-reload if available, else plain go run.
dev: migrate-up dev-run

dev-run:
	@AIR=$$(go env GOPATH)/bin/air; \
	if [ -x "$$AIR" ]; then \
		$$AIR; \
	else \
		echo "air not found, falling back to go run (install: go install github.com/air-verse/air@latest)"; \
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

.PHONY: dev build lint test test-race test-cover sqlc migrate-up migrate-down docker-build check clean

# dev depends on migrate-up so local development always boots against an
# up-to-date schema. The server itself also verifies schema drift at startup
# via internal/migration/VerifyCurrent, so invoking go run directly will
# still fail loudly if migrations are missing, but the dependency here is
# the first line of mechanical defense and removes one manual step per pull.
dev: migrate-up
	go run ./cmd/server/main.go

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

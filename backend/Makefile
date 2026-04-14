.PHONY: dev dev-run dev-raw build lint test test-race test-cover sqlc migrate-up migrate-down seed docker-build check clean install-tools trigger-batch-analysis

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

# seed: load idempotent reference data from db/seed/*.sql.
# Safe to run repeatedly; seed files must use ON CONFLICT / IF NOT EXISTS.
seed:
	go run ./cmd/seed

docker-build:
	docker build -t richman-api .

check: lint test build

clean:
	rm -rf bin/

# trigger-batch-analysis: manually re-run the daily 06:00 asset batch analysis.
# Use when the cron window failed (typically richson outage). Requires:
#   ADMIN_JWT_TOKEN  - JWT belonging to a role=admin user
#   API_BASE         - optional, defaults to http://localhost:8080
# Optional override:
#   ASSET_CODES_JSON - JSON array of codes to scope the retry, e.g. '["XAUUSD","SH600519"]'
#                      When unset the server falls back to all active assets (richman-backend-v2-trd SS8.8).
trigger-batch-analysis:
	@if [ -z "$(ADMIN_JWT_TOKEN)" ]; then echo "ADMIN_JWT_TOKEN is required"; exit 1; fi
	@BASE="$${API_BASE:-http://localhost:8080}"; \
	BODY=$${ASSET_CODES_JSON:+'{"assetCodes":'$$ASSET_CODES_JSON'}'}; \
	BODY=$${BODY:-'{}'}; \
	echo "POST $$BASE/api/v2/analysis/trigger-batch  body=$$BODY"; \
	curl --fail-with-body -sS -X POST "$$BASE/api/v2/analysis/trigger-batch" \
		-H "Authorization: Bearer $(ADMIN_JWT_TOKEN)" \
		-H "Content-Type: application/json" \
		--data "$$BODY"; \
	echo

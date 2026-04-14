.PHONY: help install init dev run migrate-up migrate-down migrate-status lint fmt test check clean

HOST ?= 0.0.0.0
PORT ?= 8100

help:
	@echo "richson — quantitative computation sidecar"
	@echo ""
	@echo "First-time setup (one command):"
	@echo "  make init             sync deps + create .env + run migrations"
	@echo ""
	@echo "Run:"
	@echo "  make dev              start uvicorn on :$(PORT) with hot reload"
	@echo "  make run              start uvicorn on :$(PORT) without reload"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up       apply pending migrations"
	@echo "  make migrate-down     roll back one migration"
	@echo "  make migrate-status   show current revision"
	@echo ""
	@echo "Quality:"
	@echo "  make install          sync deps (includes dev extras)"
	@echo "  make lint             ruff + mypy"
	@echo "  make fmt              ruff format"
	@echo "  make test             pytest"
	@echo "  make check            lint + test"
	@echo ""
	@echo "Housekeeping:"
	@echo "  make clean            remove caches and build artifacts"

# install: sync dependencies including dev extras (ruff / mypy / pytest).
# Uses uv.lock for reproducible installs.
install:
	uv sync --extra dev

# init: one-shot first-time setup. Idempotent — safe to re-run.
#   1) sync deps
#   2) copy .env.example to .env (skips if .env exists)
#   3) run alembic upgrade head
init: install
	@if [ -f .env ]; then \
		echo "[init] .env already exists, keeping current values"; \
	else \
		cp .env.example .env; \
		echo "[init] created .env from .env.example"; \
	fi
	@$(MAKE) migrate-up
	@echo ""
	@echo "[init] complete. Next: make dev"

# dev: start uvicorn with auto-reload. pydantic-settings loads .env automatically.
dev:
	uv run uvicorn richson.main:app --host $(HOST) --port $(PORT) --reload

# run: start uvicorn without reload (prod-like single-worker run).
run:
	uv run uvicorn richson.main:app --host $(HOST) --port $(PORT)

migrate-up:
	uv run alembic upgrade head

migrate-down:
	uv run alembic downgrade -1

migrate-status:
	uv run alembic current

lint:
	uv run ruff check src
	uv run mypy src

fmt:
	uv run ruff format src

test:
	uv run pytest

check: lint test

# clean: remove caches outside .venv. Does not touch dependencies or .env.
clean:
	find . -type d -name __pycache__ -not -path "./.venv/*" -exec rm -rf {} +
	rm -rf .ruff_cache .mypy_cache .pytest_cache build dist *.egg-info

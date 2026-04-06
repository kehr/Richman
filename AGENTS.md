# Repository Guidelines

## Agent Context Priority
- `CLAUDE.md` is the primary operating context for Claude Code.
- This file is a concise contributor guide; if guidance overlaps, follow `CLAUDE.md` first.

## Project Structure & Module Organization
- `frontend/`: Vite + React 19 SPA. Main code is in `frontend/src/` with layered folders: `pages/` (route-level composition), `features/` (business modules), `domain/` (shared infrastructure), `layouts/`, and `ui-kit/eat/` (UI exports).
- `backend/`: Go + Gin + PostgreSQL service. Entry point is `backend/cmd/server/main.go`; core logic lives under `backend/internal/` (`api/`, `service/`, `repo/`, `analysis/`, `datasource/`, `notification/`).
- `backend/db/migration/`: SQL schema migrations. `docs/`: product docs, implementation plans, and engineering standards.

## Build, Test, and Development Commands
- Start dependencies: `docker-compose up -d` (PostgreSQL on `5433`).
- Frontend dev: `cd frontend && pnpm dev`.
- Frontend quality checks: `cd frontend && pnpm lint:all`.
- Frontend tests: `cd frontend && pnpm test` or `pnpm test:coverage`.
- Backend dev: `cd backend && make dev`.
- Backend full check: `cd backend && make check` (lint + test + build).
- Backend tests: `cd backend && make test` (or `make test-race`, `make test-cover`).

## Coding Style & Naming Conventions
- Frontend: TypeScript uses `camelCase` for functions/variables, `PascalCase` for components/types; file names use `kebab-case` (`.ts`) and `PascalCase.tsx`.
- Backend: Go files use `snake_case`; exported identifiers use `PascalCase`, internal identifiers use `camelCase`.
- Language rule: code/comments/logs in English; documentation in Chinese.
- Frontend rule: import Ant Design through `frontend/src/ui-kit/eat/index.ts`; pages depend on feature barrels only.
- Backend rule: keep `handlers -> service -> repo`; do not read env directly in business code.
- Run format/lint before PR: `pnpm lint` (frontend) and `golangci-lint run ./...` (backend, included in `make check`).

## Testing Guidelines
- Frontend: Vitest (`*.test.ts` / `*.test.tsx`) colocated with source files.
- Backend: Go `testing` with `_test.go` files colocated with implementation.
- No fixed coverage gate is enforced; prioritize coverage for analysis engine, portfolio cost logic, API handlers, and notification adapters.

## Commit & Pull Request Guidelines
- Follow existing history style: short, imperative English commit subjects.
- Recommended branch names: `feat/<topic>` or `fix/<topic>`.
- PRs should include: summary, key changes, test commands run, and screenshots for UI updates.
- Link related issues or planning docs in `docs/plans/` when applicable.

## Security & Configuration Tips
- Use `.env.example` as template; keep real secrets in local `.env` / `.env.local` only.
- Follow standards in `docs/standards/` when changing naming, API contracts, DB schema, logging, or testing.

# Richman - AI Investment Research Assistant

## Architecture

Monorepo, frontend and backend separated:
- Frontend: Vite + React 19 + React Router v7 (SPA) + Ant Design 6 + TanStack Query v5
- Backend: Go (Gin + sqlc + PostgreSQL) -> Docker + VPS
- API: RESTful JSON, client-agnostic, all clients consume the same API
- Reference project: /Users/kyle/Studio/Orbiter (frontend patterns)

## Tech Stack

| Layer | Stack |
|-------|-------|
| Frontend | Vite 7, React 19, React Router v7, Ant Design 6, @ant-design/pro-components, TanStack Query v5, TypeScript strict, Biome, pnpm |
| Backend | Go, Gin, sqlc, PostgreSQL, Uber zap, golangci-lint |
| LLM | Multi-provider abstraction (Claude API / OpenAI API) |
| Data | AKShare (A-share), Yahoo Finance (US/Gold), Polymarket (events) |

## Project Structure

```
frontend/          # Vite SPA (Pages + Features dual architecture)
backend/           # Go service (API handlers -> Service -> Repo)
docs/
  prds/            # PRD and design specs
  plans/           # Implementation plans
  standards/       # Engineering standards (see index below)
```

## Key Conventions

- Code/comments/logs in English, docs in Chinese, no emoji
- Frontend: all Ant Design imports through ui-kit/eat barrel (Biome enforced)
- Frontend: features isolated, pages consume via barrel only
- Backend: three-layer (handlers -> service -> repo), no direct os.Getenv
- Database: snake_case, soft delete (is_deleted), audit fields on every table
- Config: .env files per environment (dev/prod), .env.example as template, secrets not in repo

## Design Review Gate (Mandatory)

Before presenting any non-trivial design, writing any design doc, or invoking
writing-plans / executing-plans / subagent-driven-development, you MUST read
`docs/standards/design-review.md` and execute all 5 passes it defines:

1. State space enumeration
2. File invariant extraction
3. Alternate path traversal
4. Pre-mortem
5. Attack-your-own-recommendations

"Non-trivial" means anything touching state machines, cross-layer contracts,
react-query cache, route guards, lifecycle side effects, or schema changes.
The standard lists the exact trigger conditions. Skipping the gate is only
allowed for pure cosmetic / lint / test-only changes explicitly enumerated in
the standard. Every design presentation must include the 8 artifacts the
standard requires; a design without those artifacts is considered incomplete
and must not proceed.

## Dev Commands

| Command | Purpose |
|---------|---------|
| `cd frontend && pnpm dev` | Start frontend dev server (Vite, port 3000) |
| `cd frontend && pnpm lint:all` | Full frontend check (Biome + type-check + dependency-cruiser) |
| `cd frontend && pnpm build` | Production build (outputs to dist/) |
| `cd backend && make dev` | Start backend dev server (hot reload) |
| `cd backend && make check` | Full backend check (lint + test + build) |
| `cd backend && make sqlc` | Generate type-safe SQL code |
| `cd backend && make migrate-up` | Run database migrations |
| `docker-compose up -d` | Start local PostgreSQL (port 5433) |

## Standards Index

Detailed conventions in `docs/standards/`, agent reads on demand. The
`design-review.md` file is mandatory reading before any non-trivial design
task (see "Design Review Gate" above); the rest are loaded on demand.

| File | Covers |
|------|--------|
| `design-review.md` | MANDATORY pre-design 5-pass gate: state space, file invariants, alternate paths, pre-mortem, self-attack |
| `naming.md` | Files, identifiers, database, API, git naming |
| `frontend.md` | Pages+Features architecture, dependency rules, component usage, Biome config |
| `backend.md` | Go three-layer architecture, service/repo patterns, error handling |
| `database.md` | PostgreSQL schema, audit fields, soft delete, indexing, migrations |
| `api.md` | RESTful design, versioning, pagination, error format, MVP endpoints |
| `testing.md` | Test structure, naming, mock strategy (frontend + backend) |
| `logging.md` | Uber zap, log levels, request tracing, rotation, masking |

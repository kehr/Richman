# Richman - AI Investment Research Assistant

## Architecture

Monorepo, frontend and backend separated:
- Frontend: Next.js 15 (App Router) + Ant Design 6 + TanStack Query v5 -> Vercel
- Backend: Go (Gin + sqlc + PostgreSQL) -> Docker + VPS
- API: RESTful JSON, client-agnostic, all clients consume the same API
- Reference project: /Users/kyle/Studio/Orbiter (frontend patterns)

## Tech Stack

| Layer | Stack |
|-------|-------|
| Frontend | Next.js 15, Ant Design 6, @ant-design/pro-components, TanStack Query v5, TypeScript strict, next-intl (zh+en), Biome, pnpm |
| Backend | Go, Gin, sqlc, PostgreSQL, Uber zap, golangci-lint |
| LLM | Multi-provider abstraction (Claude API / OpenAI API) |
| Data | AKShare (A-share), Yahoo Finance (US/Gold), Polymarket (events) |

## Project Structure

```
frontend/          # Next.js app (Pages + Features dual architecture)
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

## Dev Commands

| Command | Purpose |
|---------|---------|
| `cd frontend && pnpm dev` | Start frontend dev server |
| `cd frontend && pnpm lint:all` | Full frontend check (Biome + type-check + dependency-cruiser) |
| `cd frontend && pnpm build` | Production build |
| `cd backend && make dev` | Start backend dev server (hot reload) |
| `cd backend && make check` | Full backend check (lint + test + build) |
| `cd backend && make sqlc` | Generate type-safe SQL code |
| `cd backend && make migrate-up` | Run database migrations |
| `docker-compose up -d` | Start local PostgreSQL |

## Standards Index

Detailed conventions in `docs/standards/`, agent reads on demand:

| File | Covers |
|------|--------|
| `naming.md` | Files, identifiers, database, API, git naming |
| `frontend.md` | Pages+Features architecture, dependency rules, component usage, Biome config |
| `backend.md` | Go three-layer architecture, service/repo patterns, error handling |
| `database.md` | PostgreSQL schema, audit fields, soft delete, indexing, migrations |
| `api.md` | RESTful design, versioning, pagination, error format, MVP endpoints |
| `testing.md` | Test structure, naming, mock strategy (frontend + backend) |
| `logging.md` | Uber zap, log levels, request tracing, rotation, masking |

# Richman

AI-driven personal investment research decision assistant.

## Quick Start

### Prerequisites

- Node.js 22+, pnpm
- Go 1.22+, golangci-lint
- Docker (for PostgreSQL)

### Local Development

```bash
# 1. Start PostgreSQL
docker-compose up -d

# 2. Backend
cd backend
cp .env.example .env  # Edit with your settings
make migrate-up
make dev

# 3. Frontend
cd frontend
cp .env.example .env.local  # Edit if needed
pnpm install
pnpm dev
```

Open http://localhost:3000 in your browser.

### Default Credentials

- Invite code: `RICHMAN2026`
- Register with any email/password using the invite code

## Project Structure

```
frontend/          # Vite + React 19 + Ant Design 6 (SPA)
backend/           # Go + Gin + PostgreSQL
docs/
  prds/            # Product requirements
  plans/           # Implementation plans
  standards/       # Engineering standards
```

## Dev Commands

### Frontend

| Command | Purpose |
|---------|---------|
| `pnpm dev` | Start dev server (Vite, port 3000) |
| `pnpm build` | Production build (outputs to dist/) |
| `pnpm lint:all` | Full check (Biome + TypeScript + architecture) |
| `pnpm test` | Run tests |

### Backend

| Command | Purpose |
|---------|---------|
| `make dev` | Start dev server |
| `make check` | Full check (lint + test + build) |
| `make migrate-up` | Run database migrations |
| `make sqlc` | Generate type-safe SQL code |

## Architecture

- Frontend: Vite SPA with Pages + Features dual architecture (React Router v7, Ant Design 6, TanStack Query)
- Backend: Three-layer (API handlers -> Service -> Repo)
- Analysis: Three-dimension engine (Trend + Position + Catalyst)
- Notification: Pluggable adapter pattern (WeChat, Feishu, Email)

See `docs/standards/` for detailed conventions.

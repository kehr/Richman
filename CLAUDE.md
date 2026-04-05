# Richman - AI Investment Research Assistant

## Architecture

Monorepo with frontend and backend separated:
- Frontend: Next.js SPA consuming backend RESTful API
- Backend: Go service providing pure RESTful JSON API
- All clients (web, future iOS/macOS/Android/Windows) consume the same API
- Reference project: /Users/kyle/Studio/Orbiter (frontend patterns, standards)

## Frontend Tech Stack

- Framework: Next.js 15 (App Router)
- UI: Ant Design 6 + @ant-design/pro-components
- Style: Ant Design CSS-in-JS + CSS Variables (no Tailwind)
- Data Fetching: TanStack Query v5 (server state)
- State: React hooks for client state (useState, useReducer, Context)
- Language: TypeScript (strict mode)
- i18n: next-intl (zh + en)
- Theme: Light/Dark via Ant Design ConfigProvider
- Package Manager: pnpm
- Lint/Format: Biome (tab indent, 100 char, double quotes, always semicolons)
- Architecture Check: dependency-cruiser
- Deploy: Vercel

## Backend Tech Stack

- Language: Go
- Web Framework: Gin
- Database: PostgreSQL (Supabase or self-hosted)
- DB Access: sqlc (type-safe SQL generation)
- LLM: Multi-provider abstraction (Claude API / OpenAI API)
- Scheduler: Cron (AM 08:30 + PM 15:30 + US stock 06:00 analysis triggers)
- Deploy: Docker + VPS

## Data Sources

- AKShare: A-share market data and valuation
- Yahoo Finance: US stocks and gold
- Polymarket API: Event market probabilities
- LLM web search: Real-time narrative (catalyst enhancement)

## Frontend Architecture (Orbiter Pattern)

### Pages + Features Dual Architecture

```
frontend/src/
  config/
    routes.tsx              # Route config (declarative)
    theme.ts                # Ant Design ThemeConfig (light/dark)
  pages/                    # Page assembly layer (pure composition)
    dashboard/
    portfolio/
    analysis/
    settings/
    auth/
  features/                 # Business module layer (self-contained)
    dashboard/
      api.ts                # API functions + DTO types
      useStats.ts           # TanStack Query hooks
      index.ts              # Barrel export (public API only)
    portfolio/
      api.ts
      usePortfolio.ts
      index.ts
    analysis/
      api.ts
      useAnalysis.ts
      index.ts
    decision-card/
      api.ts
      useDecisionCard.ts
      index.ts
    notification/
      api.ts
      useNotification.ts
      index.ts
    auth/
      api.ts
      useAuth.ts
      index.ts
  domain/                   # Cross-module infrastructure
    http/                   # API client wrapper
    auth/                   # Auth (storage, hooks, guards)
    i18n/                   # i18n config and locale files
    ui/                     # Common UI utilities
  layouts/
    MainLayout.tsx          # Main app layout (ProLayout side mode)
  ui-kit/
    eat/                    # Ant Design barrel (ALL UI imports go through here)
    svg/                    # SVG components
```

### Dependency Flow

```
App.tsx -> config/ -> pages/ -> features/ -> domain/ -> ui-kit/eat
```

### Layer Rules

| Layer | Role | Can Import | Cannot Import |
|-------|------|-----------|---------------|
| config/ | Routes & theme | pages (refs), ui-kit/eat | features, domain |
| pages/ | Assemble page | features/*/index (barrel), domain, ui-kit/eat | feature internals |
| features/ | Self-contained business | domain, ui-kit/eat | other features, pages |
| domain/ | Cross-module infra | ui-kit/eat, 3rd-party | features, pages |
| layouts/ | Page layout | config, ui-kit/eat | features, domain, pages |
| ui-kit/ | Ant Design wrapper | antd packages | business code |

### Feature Module Pattern

```typescript
// features/xxx/api.ts - API functions + types
export interface XxxDto { /* ... */ }
export function fetchXxx() {
  return request<{ data: XxxDto }>("/xxx");
}

// features/xxx/useXxx.ts - TanStack Query hooks
export function useXxx() {
  return useQuery({ queryKey: ["xxx"], queryFn: fetchXxx });
}

// features/xxx/index.ts - Barrel export (public API only)
export { useXxx } from "./useXxx";
export type { XxxDto } from "./api";
```

### UI Component Import Rule

All Ant Design imports go through ui-kit/eat barrel. NEVER import directly from antd, @ant-design/pro-components, @ant-design/icons. Biome noRestrictedImports enforces this.

### Pro Component Priority

| Scenario | Use | Fallback |
|----------|-----|----------|
| Card container | Card (eat, borderless default) | - |
| Stats metric | StatisticCard | Card + Statistic |
| Description list | ProDescriptions | Descriptions |
| Data table | ProTable | Table |
| Page layout | ProLayout | Layout + Sider + Menu |

## Backend Architecture (Go)

```
backend/
  cmd/
    server/main.go         # HTTP API entry point
  internal/
    api/                    # HTTP handlers (Gin routes)
      middleware/            # Auth, plan-check, CORS
      v1/                   # Route handlers
    service/                # Business logic orchestration
      portfolio/
      analysis/
      notification/
      auth/
    repo/                   # Data access layer (sqlc generated)
    model/                  # Domain models
    analysis/               # Three-dimension analysis engine
      trend/                # Trend dimension (quantitative)
      position/             # Position dimension (quantitative)
      catalyst/             # Catalyst dimension (quant + LLM)
      synthesis/            # LLM synthesis layer
    notification/           # Push notification hub
      adapter/              # Pluggable channel adapters
        wechat/
        feishu/
        email/
    llm/                    # LLM provider abstraction
    datasource/             # Data source integrations
      akshare/
      yahoo/
      polymarket/
    config/                 # Configuration management
  db/
    query/                  # SQL queries (sqlc input)
    migration/              # Database migrations
    sqlc.yaml               # sqlc config
```

### Three-Layer Dependency Rules (Go)

```
API handlers -> Service -> Repo -> DB
```

| Layer | Can Import | Cannot Import |
|-------|-----------|---------------|
| API handlers | service, config, middleware | repo, db directly |
| Service | repo, config, external services | API handlers |
| Repo | db/query (sqlc generated) | service, API handlers |

## Project Structure (Root)

```
frontend/              # Next.js frontend app
backend/               # Go backend service
docs/                  # Product and technical documents
  prds/               # PRD and design specs
  standards/           # Engineering standards (naming, frontend, backend, database, api, testing)
```

## Code Conventions

- Use English for all code, comments, and log messages
- Use Chinese for documentation
- No emoji in code or docs
- Biome for frontend lint+format (tab indent, 100 char width, double quotes, always semicolons)
- Frontend: all UI through ui-kit/eat barrel, never import antd directly
- Frontend: features isolated from each other, pages consume via barrel only
- Backend: standard Go project layout, Gin handlers -> service -> repo layers
- API: RESTful JSON, camelCase fields, client-agnostic
- Database: snake_case tables/columns, soft delete with is_deleted, audit fields on every table

## Dev Commands

### Frontend
- `pnpm dev` - Start Next.js dev server
- `pnpm build` - Production build
- `pnpm lint` - Biome lint
- `pnpm format` - Biome format
- `pnpm type-check` - TypeScript type check
- `pnpm lint:all` - Biome + type-check + dependency-cruiser

### Backend
- `go run cmd/server/main.go` - Start Go server
- `go test ./...` - Run all tests
- `golangci-lint run ./...` - Run Go linter (MUST pass before next step)
- `sqlc generate` - Generate type-safe SQL code

## Standards (docs/standards/)

Engineering standards follow Orbiter's pattern with adaptations for Go backend:
- `naming.md` - File, directory, identifier, database, API naming conventions
- `frontend.md` - Pages+Features architecture, dependency rules, component usage
- `backend.md` - Go four-layer architecture, service/repo patterns, error handling
- `database.md` - PostgreSQL schema conventions, audit fields, indexing strategy
- `api.md` - RESTful API design, versioning, pagination, error format
- `testing.md` - Test structure, naming, mock strategy

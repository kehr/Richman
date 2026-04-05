# Richman - AI Investment Research Assistant

## Architecture

Monorepo with frontend and backend separated:
- Frontend: Next.js (SPA/SSG) consuming backend RESTful API
- Backend: Go service providing pure RESTful JSON API
- All clients (web, future iOS/macOS/Android/Windows) consume the same API

## Frontend Tech Stack

- Framework: Next.js 15 (App Router)
- UI: Ant Design 6 + @ant-design/pro-components
- Style: Ant Design CSS-in-JS + CSS Variables (no Tailwind)
- Language: TypeScript (strict mode)
- i18n: next-intl (zh + en)
- Theme: Light/Dark via Ant Design ConfigProvider
- Package Manager: pnpm
- Deploy: Vercel

## Backend Tech Stack

- Language: Go
- Web Framework: Gin
- Database: PostgreSQL (Supabase or self-hosted)
- DB Access: sqlc (type-safe SQL generation)
- LLM: Multi-provider abstraction (Claude API / OpenAI API)
- Scheduler: Cron (AM 08:30 + PM 15:30 analysis triggers)
- Deploy: Docker + VPS

## Data Sources

- AKShare: A-share market data and valuation
- Yahoo Finance: US stocks and gold
- Polymarket API: Event market probabilities
- LLM web search: Real-time narrative (catalyst enhancement)

## Project Structure

```
frontend/              # Next.js frontend app
  src/
    app/               # App Router pages and layouts
    components/        # Shared UI components
    lib/               # Frontend utilities
    types/             # TypeScript type definitions
    locales/           # i18n translation files

backend/               # Go backend service
  cmd/                 # Application entrypoints
  internal/
    api/               # HTTP handlers (Gin routes)
    service/           # Business logic
    analysis/          # Three-dimension analysis engine
    notification/      # Push notification hub (pluggable adapters)
    llm/               # LLM provider abstraction
    data/              # Data source integrations
    auth/              # Authentication and authorization
    model/             # Domain models
  db/
    query/             # SQL queries (sqlc)
    migration/         # Database migrations

docs/                  # Product and technical documents
  standards/           # Engineering standards
```

## Code Conventions

- Use English for all code, comments, and log messages
- Use Chinese for documentation
- No emoji in code or docs
- Go code follows standard Go project layout conventions
- Frontend: use Ant Design components first, custom components only when needed
- API design: RESTful JSON, client-agnostic

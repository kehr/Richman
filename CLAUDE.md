# Richman - AI Investment Research Assistant

## Tech Stack

- Framework: Next.js 15 (App Router) + Tailwind CSS 4
- Language: TypeScript (strict mode)
- Package Manager: pnpm
- Database: Supabase (PostgreSQL)
- ORM: Prisma
- LLM: Multi-provider support (Claude API / OpenAI API, abstracted LLM layer)
- Deploy: Vercel
- Data Sources: AKShare, Yahoo Finance, Polymarket API

## Project Structure

```
src/
  app/           # Next.js App Router pages and layouts
  components/    # Shared UI components
  lib/           # Core business logic
    llm/         # LLM provider abstraction layer
    data/        # Data source integrations (AKShare, Yahoo Finance, Polymarket)
    analysis/    # Three-dimension analysis framework
  types/         # TypeScript type definitions
  utils/         # Helper utilities
prisma/          # Prisma schema and migrations
docs/            # Product and technical documents
  standards/     # Engineering standards
```

## Dev Commands

- `pnpm dev` - Start development server
- `pnpm build` - Production build
- `pnpm lint` - Run ESLint
- `pnpm type-check` - TypeScript type checking
- `pnpm db:push` - Push Prisma schema to database
- `pnpm db:generate` - Generate Prisma client

## Code Conventions

- Use English for all code, comments, and log messages
- Use Chinese for documentation
- No emoji in code or docs
- API routes go in `src/app/api/`
- Server components by default, add "use client" only when needed
- Use Zod for runtime validation at API boundaries
- Environment variables prefixed with `NEXT_PUBLIC_` for client-side only

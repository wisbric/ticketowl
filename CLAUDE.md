# CLAUDE.md — TicketOwl

## Project Overview

TicketOwl is a ticket management portal built by Wisbric. It is the third pillar of the NightOwl platform, sitting alongside NightOwl (incident/on-call) and BookOwl (knowledge base).

TicketOwl is a **thin orchestration layer** over Zammad. It does not replace Zammad's ticketing engine — it extends it with:

- **NightOwl context** — incidents, alerts, and on-call assignments linked to tickets
- **BookOwl context** — runbooks, post-mortems, and KB articles surfaced inline
- **Unified UI** — a NightOwl-themed portal for internal agents and external customers
- **Automation** — incident → ticket creation, SLA breach → on-call page, resolution → post-mortem draft

TicketOwl is a Wisbric product (wisbric.com). The parent brand is NightOwl — TicketOwl is the support operations pillar of the NightOwl platform.

## Specifications

All design decisions are captured in `docs/`. Always read the relevant spec before implementing:

- `docs/01-requirements.md` — Product requirements and feature scope
- `docs/02-architecture.md` — System architecture, domain structure, API endpoints
- `docs/03-data-model.md` — PostgreSQL schema, multi-tenancy, SLA config
- `docs/04-zammad-integration.md` — Zammad API client, sync strategy, webhook events
- `docs/05-nightowl-integration.md` — Integration contract with NightOwl
- `docs/06-bookowl-integration.md` — Integration contract with BookOwl
- `docs/07-customer-portal.md` — External customer access, OIDC scoping
- `docs/08-sla.md` — SLA definitions, breach detection, escalation flow
- `docs/09-branding.md` — TicketOwl branding (follows NightOwl design system)
- `docs/10-deployment.md` — Docker Compose, Kubernetes/Helm, CI/CD
- `docs/11-tasks.md` — Implementation task checklist (work from this)

## Branding

The product is called **TicketOwl**. It uses the NightOwl design system.

- Dark mode is the default (theme toggle available)
- Same color palette as NightOwl: `#0A2E24` primary, `#00e5a0` accent, severity colors
- Sidebar navigation layout with owl logo
- Footer: "A Wisbric product"
- Owl-themed flavor names: Tickets are "Cases", the queue is "The Perch", SLA status is "The Watch"
  (flavor names only — use plain terms in API and code)

## Tech Stack

### Backend

- **Language:** Go 1.25+ (module: `github.com/wisbric/ticketowl`)
- **Binary:** `cmd/ticketowl` with modes: `api`, `worker`, `seed`, `seed-demo`
- **Router:** go-chi/chi/v5
- **Database:** PostgreSQL 16+ via jackc/pgx/v5 + sqlc
- **Migrations:** golang-migrate (SQL files in `migrations/`)
- **Cache:** Redis 7 via redis/go-redis/v9
- **Auth:** OIDC (coreos/go-oidc/v3) + API keys (SHA-256) — same provider as NightOwl/BookOwl
- **Zammad client:** `net/http` typed wrapper in `internal/zammad/`
- **Metrics:** prometheus/client_golang (namespace: `ticketowl`)
- **Tracing:** OpenTelemetry (OTLP gRPC)
- **Logging:** slog (structured JSON)
- **Config:** caarlos0/env/v11
- **UUIDs:** google/uuid

### Frontend

- **Framework:** React 19 + TypeScript 5.9
- **Build:** Vite 7
- **UI Kit:** shadcn/ui + Tailwind CSS 4 (same theme tokens as NightOwl)
- **State:** TanStack Query 5 + TanStack Router 1
- **Forms:** React Hook Form + Zod
- **Icons:** lucide-react
- **Dates:** date-fns

## Code Conventions

These mirror NightOwl and BookOwl exactly. If in doubt, look at how NightOwl does it.

### Go

- Standard `gofmt` + `golangci-lint`
- Package names are single lowercase words matching directory name
- Table-driven tests
- Errors: `fmt.Errorf("doing X: %w", err)` — never discard
- Context: always `context.Context` as first parameter
- SQL: prefer sqlc-generated code; raw SQL for JOINs not in schema
- HTTP handlers return JSON; always use `httpserver.Respond()` and `httpserver.RespondError()`
- Domain packages follow `handler.go` / `service.go` / `store.go` / `{domain}.go` pattern
- Per-request store creation from `tenant.ConnFromContext(r.Context())`

### Frontend

- Functional components only
- TanStack Query for all API calls — never raw fetch in components
- TanStack Router for all navigation — no `useNavigate` with raw strings
- Zod schemas co-located with forms
- Dark mode first — always test in dark mode before light

## Multi-Tenancy

Schema-per-tenant isolation, identical to NightOwl and BookOwl. Every request resolves a tenant from JWT or API key. The middleware acquires a pooled connection and sets `search_path` before any query.

TicketOwl tenants correspond 1:1 with NightOwl and BookOwl tenants — they share the same tenant slug. Each tenant stores its own Zammad instance URL and API token.

Never reference tenant data without going through the tenant middleware.

## Development

```bash
docker compose up -d          # PostgreSQL + Redis + Zammad
make seed                     # Create "acme" dev tenant (idempotent)
go run ./cmd/ticketowl        # API on :8082
cd web && npm run dev         # Frontend on :3002 (proxies /api to :8082)
```

- Dev API key: `to_dev_seed_key_do_not_use_in_production`
- Local admin: username `admin`, password `ticketowl-admin` (dev mode only)
- Login URL: `http://localhost:3002/login`
- Env vars prefix: `TICKETOWL_`
- DB credentials (dev): `ticketowl:ticketowl@localhost:5434/ticketowl`
- Zammad (dev): `TICKETOWL_ZAMMAD_URL=http://localhost:3003`
- NightOwl API (dev): `TICKETOWL_NIGHTOWL_API_URL=http://localhost:8080`
- NightOwl API key (dev): `TICKETOWL_NIGHTOWL_API_KEY=ow_dev_seed_key_do_not_use_in_production`
- BookOwl API (dev): `TICKETOWL_BOOKOWL_API_URL=http://localhost:8081`
- BookOwl API key (dev): `TICKETOWL_BOOKOWL_API_KEY=bw_dev_seed_key_do_not_use_in_production`

## Port Conventions (dev)

| Service       | Port  |
|---------------|-------|
| NightOwl API  | :8080 |
| BookOwl API   | :8081 |
| TicketOwl API | :8082 |
| NightOwl UI   | :3000 |
| BookOwl UI    | :3001 |
| TicketOwl UI  | :3002 |
| Zammad        | :3003 |

## Testing

- Unit tests: mock dependencies via interfaces, table-driven
- Integration tests: testcontainers-go for real PostgreSQL and Redis
- Zammad client: mock `httptest` server — never call a real Zammad in CI
- Run `make test` before committing
- Run `make lint` before committing

## Commit Style

- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `chore:`
- One logical change per commit
- Reference task IDs from `docs/11-tasks.md` (e.g. `feat(sla): breach detection worker [9.2]`)

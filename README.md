<p align="center">
  <img src="docs/owl.png" alt="TicketOwl" width="120">
</p>

<h1 align="center">TicketOwl</h1>

<p align="center">
  <em>Support operations for the NightOwl platform.</em>
</p>

<p align="center">
  <a href="https://wisbric.com"><img src="https://img.shields.io/badge/a_Wisbric_product-0F1117?style=flat&labelColor=0F1117&color=00e5a0" alt="A Wisbric product"></a>
</p>

---

TicketOwl is a ticket management portal that extends [Zammad](https://zammad.org) with NightOwl incident context, BookOwl knowledge base integration, SLA breach detection, and automated ticket creation. It is a thin orchestration layer — Zammad remains the ticketing engine, while TicketOwl adds unified operations context and automation on top. It is a [Wisbric](https://wisbric.com) product.

---

## Features

- **Zammad Integration** — proxied ticket views, creation, and updates powered by Zammad's API
- **NightOwl Context** — incidents, alerts, and on-call assignments linked to tickets
- **BookOwl Context** — runbooks, post-mortems, and KB articles surfaced inline
- **SLA Management** — per-priority SLA policies with response and resolution deadlines
- **SLA Breach Detection** — background worker polls for breaches and pages on-call via NightOwl alerts
- **Auto-Ticket Rules** — automatically create tickets from NightOwl incidents based on configurable rules (alert group, severity threshold)
- **Event Processing** — Redis stream consumer for real-time Zammad webhook and NightOwl event handling
- **Internal Notes** — team-only notes on tickets, separate from Zammad articles
- **Incident & Article Linking** — link tickets to NightOwl incidents and BookOwl documents
- **Customer Portal** — external customer access scoped by OIDC organization
- **Multi-Tenancy** — schema-per-tenant PostgreSQL isolation
- **Cookie-Based Sessions** — shared `wisbric_session` cookie with NightOwl and BookOwl for cross-service SSO
- **OIDC + API Key Auth** — role-based access with OIDC, API keys, and cookie sessions via shared `core/pkg/auth`
- **Audit Trail** — every API action logged with actor, resource, and metadata
- **Observability** — Prometheus metrics and OpenTelemetry tracing
- **Dark Mode** — NightOwl design system with dark mode default

## Architecture

```
┌─────────────┐     ┌───────────────┐     ┌──────────────┐
│   Frontend   │────▶│ TicketOwl API  │────▶│  PostgreSQL   │
│  React/Vite  │     │   Go / Chi    │     │  (per-tenant) │
└─────────────┘     └──────┬────────┘     └──────────────┘
                           │
                    ┌──────┴────────┐
                    │     Redis     │
                    │ (cache/events)│
                    └───────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────┴──────┐ ┌──┴───┐ ┌──────┴──────┐
       │   Zammad    │ │Night │ │   BookOwl   │
       │ (ticketing) │ │ Owl  │ │   (docs)    │
       └─────────────┘ └──────┘ └─────────────┘
```

TicketOwl runs as two deployments: an **API server** for HTTP requests and a **worker** for background SLA polling and event processing. Both share the same binary with different modes.

## Tech Stack

**Backend:** Go 1.25+, Chi router, PostgreSQL 16+ (pgx + sqlc), Redis 7 (go-redis/v9), OIDC (go-oidc), golang-migrate, OpenTelemetry

**Frontend:** React 19, TypeScript 5.9, Vite 7, shadcn/ui, Tailwind CSS 4, TanStack Query 5 + Router 1, React Hook Form + Zod, lucide-react

---

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose (for PostgreSQL, Redis, Zammad)

### Development

```bash
# Start dependencies
docker compose up -d

# Seed the database (creates "acme" tenant + local admin)
make seed

# Run the API server
go run ./cmd/ticketowl

# In another terminal, start the frontend
cd web && npm install && npm run dev
```

The API runs on `http://localhost:8082` and the frontend on `http://localhost:3002`.

### Default Credentials

- **Dev API key:** `to_dev_seed_key_do_not_use_in_production`
- **Local admin:** username `admin`, password `ticketowl-admin` (dev mode only; forced password change on first login)
- **Database:** `ticketowl:ticketowl@localhost:5434/ticketowl`

### Configuration

| Variable | Example | Description |
|----------|---------|-------------|
| `APP_MODE` | `api\|worker\|seed` | Runtime mode (default: `api`) |
| `APP_HOST` | `0.0.0.0` | Bind address |
| `APP_PORT` | `8082` | HTTP port |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `REDIS_URL` | `redis://...` | Redis connection string |
| `LOG_LEVEL` | `info` | Log level (`debug`/`info`/`warn`/`error`) |
| `TICKETOWL_ZAMMAD_URL` | `http://localhost:3003` | Zammad instance URL |
| `TICKETOWL_NIGHTOWL_API_URL` | `http://localhost:8080` | NightOwl API base URL |
| `TICKETOWL_NIGHTOWL_API_KEY` | `ow_dev_seed_key...` | NightOwl API key |
| `TICKETOWL_BOOKOWL_API_URL` | `http://localhost:8081` | BookOwl API base URL |
| `TICKETOWL_BOOKOWL_API_KEY` | `bw_dev_seed_key...` | BookOwl API key |
| `TICKETOWL_SESSION_SECRET` | `...` | Session signing secret (required unless `DEV_MODE=true`) |
| `TICKETOWL_ENCRYPTION_KEY` | `...` | Encryption key for Zammad tokens at rest |
| `TICKETOWL_WORKER_POLL_SECONDS` | `60` | SLA poller interval (default: 60) |
| `OIDC_ISSUER` | `https://...` | OIDC issuer URL |
| `OIDC_CLIENT_ID` | `ticketowl` | OIDC client ID |
| `DEV_MODE` | `false` | Enables dev-only auth shortcuts |

---

## Worker

The worker runs as a separate process (`APP_MODE=worker`) and handles:

1. **SLA Poller** — every 60 seconds, queries for breached SLA states and sends alerts to NightOwl
2. **Event Processor** — reads from a Redis stream (`ticketowl:{tenant}:events`) for Zammad webhooks and NightOwl events, dispatching to handlers for ticket creation, SLA state updates, and auto-close on incident resolution

Both loops run per-tenant — the worker lists all tenants on startup and launches a goroutine pair for each.

---

## API Overview

All endpoints require authentication and are scoped to a tenant.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Health check |
| `POST` | `/auth/local` | Local admin login |
| `POST` | `/auth/login` | Session login |
| `GET` | `/auth/me` | Current user info |
| `POST` | `/auth/logout` | Logout |
| `GET` | `/api/v1/tickets` | List tickets (proxied from Zammad) |
| `POST` | `/api/v1/tickets` | Create ticket |
| `GET` | `/api/v1/tickets/:id` | Ticket detail with TicketOwl metadata |
| `GET` | `/api/v1/tickets/:id/notes` | Internal notes |
| `POST` | `/api/v1/tickets/:id/notes` | Add internal note |
| `GET` | `/api/v1/tickets/:id/links` | Incident and article links |
| `POST` | `/api/v1/tickets/:id/links/incidents` | Link incident |
| `POST` | `/api/v1/tickets/:id/links/articles` | Link BookOwl article |
| `GET` | `/api/v1/sla/policies` | List SLA policies |
| `POST` | `/api/v1/sla/policies` | Create SLA policy |
| `GET` | `/api/v1/sla/status/:ticket_id` | SLA status for a ticket |
| `GET` | `/api/v1/auto-ticket-rules` | List auto-ticket rules |
| `POST` | `/api/v1/auto-ticket-rules` | Create auto-ticket rule |
| `POST` | `/api/v1/webhooks/zammad` | Zammad webhook receiver |
| `POST` | `/api/v1/webhooks/nightowl` | NightOwl webhook receiver |
| `GET` | `/api/v1/admin/config` | Admin configuration |
| `GET` | `/api/v1/audit-log` | Audit log |

---

## Project Structure

```
cmd/ticketowl/          Application entry point
internal/
  app/                   Application bootstrap and mode routing
  admin/                 Admin API handlers (Zammad config, integration keys)
  authadapter/           Auth storage adapter (implements core auth.Storage)
  comment/               Internal notes on tickets
  config/                Configuration loading (extends core BaseConfig)
  customer/              Customer organization management
  db/                    sqlc generated code (global + tenant schemas)
  link/                  Incident and article linking
  nightowl/              NightOwl API client (alerts, incidents)
  notification/          SLA breach notification service
  seed/                  Development seed data
  sla/                   SLA policies, states, computation
  telemetry/             Metrics definitions
  ticket/                Ticket domain (Zammad proxy + metadata)
  webhook/               Zammad and NightOwl webhook handlers
  worker/                Background worker (SLA poller + event processor)
web/                     React frontend (Vite + TypeScript)
charts/ticketowl/        Helm chart for Kubernetes
migrations/
  global/                Global schema migrations
  tenant/                Per-tenant schema migrations
docs/                    Design specifications (01-11)
```

---

## Documentation

| Doc | Topic |
|-----|-------|
| [01-requirements](docs/01-requirements.md) | Product requirements and feature scope |
| [02-architecture](docs/02-architecture.md) | System architecture and API endpoints |
| [03-data-model](docs/03-data-model.md) | PostgreSQL schema and multi-tenancy |
| [04-zammad-integration](docs/04-zammad-integration.md) | Zammad API client and sync strategy |
| [05-nightowl-integration](docs/05-nightowl-integration.md) | NightOwl integration contract |
| [06-bookowl-integration](docs/06-bookowl-integration.md) | BookOwl integration contract |
| [07-customer-portal](docs/07-customer-portal.md) | External customer access |
| [08-sla](docs/08-sla.md) | SLA definitions, breach detection, escalation |
| [09-branding](docs/09-branding.md) | NightOwl design system |
| [10-deployment](docs/10-deployment.md) | Docker, Kubernetes, Helm, CI/CD |
| [11-tasks](docs/11-tasks.md) | Implementation task checklist |

---

## Deployment

TicketOwl deploys alongside NightOwl and BookOwl on Kubernetes via the [umbrella-owl](https://github.com/wisbric/umbrella-owl) Helm chart.

```bash
helm install ticketowl charts/ticketowl/ \
  --set secrets.dbUrl="postgres://..." \
  --set secrets.redisUrl="redis://..." \
  --set config.nightowlApiUrl="http://nightowl:80" \
  --set secrets.nightowlApiKey="..." \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=ticketowl.example.com
```

---

## License

Copyright Wisbric. All rights reserved.

---

<p align="center">
  A <a href="https://wisbric.com">Wisbric</a> product
</p>

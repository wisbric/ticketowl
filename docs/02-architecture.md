# 02 — Architecture

## System Overview

```
                    ┌─────────────────────────────────────────────────┐
                    │               Wisbric Platform                   │
                    │                                                  │
  Customer ─────────┤  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
  Agent    ─────────┤  │NightOwl  │  │TicketOwl │  │BookOwl   │    │
                    │  │:8080/3000│◄►│:8082/3002│◄►│:8081/3001│    │
                    │  └──────────┘  └────┬─────┘  └──────────┘    │
                    │                     │                           │
                    │               Zammad :3003                      │
                    │                                                  │
                    │  ┌───────────────────────────────────┐          │
                    │  │  PostgreSQL · Redis · Keycloak     │          │
                    │  └───────────────────────────────────┘          │
                    └─────────────────────────────────────────────────┘
```

**TicketOwl never stores ticket content.** Subjects, descriptions, and comments live exclusively in Zammad. TicketOwl stores only metadata: links, SLA state, integration references, and configuration.

---

## Binary Modes

```
cmd/ticketowl/main.go — parses TICKETOWL_MODE and dispatches

Modes:
  api         — HTTP API server + webhook receiver
  worker      — SLA polling loop + event processor
  seed        — create dev tenant "acme" (idempotent)
  seed-demo   — seed demo tickets and links for UI development
```

In production, `api` and `worker` run as separate Kubernetes Deployments from the same image.

---

## Internal Package Structure

```
internal/
  httpserver/     — shared server setup, Respond/RespondError helpers
  tenant/         — middleware: resolve tenant, acquire conn, set search_path
  authadapter/    — Auth storage adapter (implements core/pkg/auth.Storage)

  zammad/         — typed Zammad REST client
  nightowl/       — typed NightOwl REST client
  bookowl/        — typed BookOwl REST client

  ticket/         — core ticket domain (proxy + metadata)
  link/           — ticket ↔ incident and ticket ↔ article links
  sla/            — SLA policy, state machine, breach detection
  customer/       — customer org management, portal scoping
  comment/        — unified comment thread (Zammad articles + internal notes)
  webhook/        — inbound Zammad and NightOwl webhook handlers
  notification/   — SLA breach → NightOwl alert trigger
  admin/          — tenant config endpoints
  worker/         — SLA polling loop, Redis event queue processor
```

Each domain package follows the NightOwl convention:
- `{domain}.go` — types and interfaces
- `handler.go` — HTTP handlers
- `service.go` — business logic
- `store.go` — sqlc-backed DB operations

---

## API Endpoints

All endpoints are prefixed `/api/v1`. Auth via `wisbric_session` cookie, `Authorization: Bearer <jwt>`, or `X-API-Key`.

### Tickets
```
GET    /api/v1/tickets
POST   /api/v1/tickets
GET    /api/v1/tickets/{id}
PATCH  /api/v1/tickets/{id}
POST   /api/v1/tickets/{id}/comments
GET    /api/v1/tickets/{id}/comments
```

### Links
```
GET    /api/v1/tickets/{id}/links
POST   /api/v1/tickets/{id}/links/incident
DELETE /api/v1/tickets/{id}/links/incident/{incident_id}
POST   /api/v1/tickets/{id}/links/article
DELETE /api/v1/tickets/{id}/links/article/{article_id}
```

### SLA
```
GET    /api/v1/tickets/{id}/sla
GET    /api/v1/sla/policies
POST   /api/v1/sla/policies
PUT    /api/v1/sla/policies/{id}
DELETE /api/v1/sla/policies/{id}
```

### Enrichment
```
GET    /api/v1/tickets/{id}/suggestions    — BookOwl articles suggested for this ticket
POST   /api/v1/tickets/{id}/postmortem     — create BookOwl post-mortem draft
```

### Customer Portal
```
GET    /api/v1/portal/tickets
GET    /api/v1/portal/tickets/{id}
POST   /api/v1/portal/tickets/{id}/reply
GET    /api/v1/portal/tickets/{id}/articles
```

### Webhooks (inbound)
```
POST   /api/v1/webhooks/zammad
POST   /api/v1/webhooks/nightowl
```

### Admin
```
GET    /api/v1/admin/config
PUT    /api/v1/admin/config/zammad
POST   /api/v1/admin/config/zammad/test
PUT    /api/v1/admin/config/nightowl
PUT    /api/v1/admin/config/bookowl
GET    /api/v1/admin/customers
POST   /api/v1/admin/customers
PUT    /api/v1/admin/customers/{id}
DELETE /api/v1/admin/customers/{id}
GET    /api/v1/admin/rules
POST   /api/v1/admin/rules
PUT    /api/v1/admin/rules/{id}
DELETE /api/v1/admin/rules/{id}
```

### Health
```
GET    /healthz
GET    /readyz
GET    /metrics
```

---

## Authentication & Authorisation

Auth is handled by the shared `core/pkg/auth` package (same as NightOwl/BookOwl). All browser sessions use HttpOnly cookies.

**Middleware precedence:** Cookie → PAT → Session JWT (Bearer) → OIDC JWT (Bearer) → API Key → Dev header.

### Internal users (agents, admins)
- **Cookie session:** `wisbric_session` (HttpOnly, Secure, SameSite=Strict) — set on login via `/auth/local` or `/auth/callback`
- **OIDC:** Keycloak, same realm as NightOwl/BookOwl
- Role claim `ticketowl_role`: `admin` | `agent` | `viewer`
- Tenant resolved from session cookie claims or JWT `tenant` claim
- Silent cookie refresh when token has <2h remaining

### External customers
- OIDC JWT from customer-scoped Keycloak client
- Role claim: `customer`
- Org resolved from JWT `org_id` claim (mapped from Keycloak group)
- All portal endpoints enforce org scoping in the store layer

### Service-to-service
- API key in `X-API-Key` header, stored as SHA-256 hash in `global.api_keys`
- NightOwl and BookOwl each have their own key per tenant

### Local admin
- Break-glass login at `POST /auth/local` (username + password, bcrypt)
- Forced password change on first login (`must_change = true`)
- Creates `wisbric_session` cookie on success

---

## Zammad Integration Pattern

- All reads of ticket content go live to Zammad (no caching of content)
- Writes (status change, new comment) are proxied through TicketOwl to Zammad
- Zammad webhook events trigger SLA recompute via Redis event stream
- If Zammad is unreachable, return 503 — never serve stale ticket content
- Per-tenant base URL and token loaded from DB at request time

---

## NightOwl Integration Pattern

Outbound (via `internal/nightowl/` typed client):
- Fetch incident detail for linking
- Fetch current on-call for a service
- POST to alert ingest for SLA breach escalation

Inbound (via `POST /api/v1/webhooks/nightowl`):
- `incident.created` → evaluate auto-ticket rules → create Zammad ticket if matched
- `incident.resolved` → find linked ticket → update Zammad status → offer post-mortem

---

## BookOwl Integration Pattern

Outbound (via `internal/bookowl/` typed client):
- `GET /api/v1/search?q=...&tags=...` — article suggestions for a ticket
- `POST /api/v1/postmortems` — create post-mortem draft

---

## Worker

Two goroutines in the `worker` binary:

**SLA Poller** — every 60s: query TicketOwl DB for open tickets, fetch current state from Zammad, recompute SLA remaining accounting for pauses, update `sla_states`, fire NightOwl alert on new breaches.

**Event Processor** — Redis `XREAD` consumer: decouples Zammad webhook handling from the HTTP request cycle. Handles ticket.create (init SLA state), ticket.update (SLA pause/resume), incident events (auto-ticket creation, resolution sync).

---

## Frontend Structure

```
web/src/
  routes/
    _layout.tsx
    tickets/index.tsx          — ticket list with filter bar
    tickets/$ticketId.tsx      — ticket detail (enriched)
    portal/tickets/index.tsx   — customer ticket list
    portal/tickets/$ticketId.tsx
    admin/
      zammad.tsx
      sla.tsx
      customers.tsx
      rules.tsx
    login.tsx
  components/
    tickets/ links/ sla/ portal/ admin/ ui/
  lib/
    api.ts                     — typed TanStack Query hooks (credentials: 'same-origin')
    auth.ts                    — auth context (cookie-based via /auth/me)
```

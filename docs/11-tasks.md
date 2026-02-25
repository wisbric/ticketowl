# 11 — Tasks

Implementation task checklist. Claude Code works through these in order.
Status: `[ ]` not started · `[~]` in progress · `[x]` complete

---

## Phase 0 — Scaffold

- [x] **[0.1]** `go mod init github.com/wisbric/ticketowl`, add all dependencies to `go.mod`
- [x] **[0.2]** `cmd/ticketowl/main.go` — mode dispatch (`api`, `worker`, `seed`, `seed-demo`, `migrate`)
- [x] **[0.3]** `internal/httpserver/` — shared server setup, `Respond()`, `RespondError()`, middleware chain
- [x] **[0.4]** `internal/tenant/` — middleware: resolve tenant, acquire pooled conn, set `search_path`
- [x] **[0.5]** `internal/auth/` — OIDC token validation, API key lookup (SHA-256 hash), role extraction
- [x] **[0.6]** Config struct with `caarlos0/env/v11`, all `TICKETOWL_` vars
- [x] **[0.7]** `slog` structured JSON logging setup
- [x] **[0.8]** OpenTelemetry setup (OTLP gRPC trace provider, conditional on `TICKETOWL_OTEL_ENDPOINT`)
- [x] **[0.9]** Prometheus metrics setup (namespace: `ticketowl`)
- [x] **[0.10]** `docker-compose.yml` — PostgreSQL (:5434), Redis (:6381), Zammad (:3003) with its own dedicated postgres and redis
- [x] **[0.11]** `Makefile` — all targets from `docs/10-deployment.md`
- [x] **[0.12]** `Dockerfile` — multi-stage distroless build
- [x] **[0.13]** `.golangci.yml` — linter config matching NightOwl
- [x] **[0.14]** `sqlc.yaml` — sqlc config
- [x] **[0.15]** `web/` — Vite + React 19 + TS + Tailwind + shadcn/ui scaffold

## Phase 1 — Database & Migrations

- [x] **[1.1]** Global migrations: `tenants`, `api_keys` (see `docs/03-data-model.md`)
- [x] **[1.2]** Tenant migrations 0001–0010 (all tables from data model doc)
- [x] **[1.3]** Migration runner wired into `migrate` mode and called before `api`/`worker` start
- [x] **[1.4]** `sqlc generate` for all domain queries
- [x] **[1.5]** `make seed` — create `acme` tenant, insert dev API keys, default SLA policies, dev Zammad config
- [x] **[1.6]** `make seed-demo` — insert 10 sample `ticket_meta` rows, SLA states, incident links

## Phase 2 — Zammad Client

- [x] **[2.1]** `internal/zammad/client.go` — base client, retry, OTel spans, slog logging (see `docs/04-zammad-integration.md`)
- [x] **[2.2]** `internal/zammad/tickets.go` — `ListTickets`, `GetTicket`, `CreateTicket`, `UpdateTicket`, `SearchTickets`
- [x] **[2.3]** `internal/zammad/comments.go` — `ListArticles`, `CreateArticle`
- [x] **[2.4]** `internal/zammad/users.go`, `orgs.go` — lookup methods
- [x] **[2.5]** `internal/zammad/states.go` — `ListStates`, `ListPriorities` with Redis caching (TTL 1h)
- [x] **[2.6]** `internal/zammad/webhook.go` — payload types, HMAC-SHA1 validation
- [x] **[2.7]** `internal/zammad/errors.go` — `ZammadError`, `IsNotFound`, `IsUnauthorised`
- [x] **[2.8]** `internal/zammad/mock_test.go` — `httptest`-based mock with `AddTicket`, `AddArticle`, `RequireCall`
- [x] **[2.9]** Unit tests for all Zammad client methods using mock server

## Phase 3 — NightOwl & BookOwl Clients

- [x] **[3.1]** `internal/nightowl/client.go` — base client (same pattern as zammad)
- [x] **[3.2]** `internal/nightowl/incidents.go` — `GetIncident`, `ListIncidents`
- [x] **[3.3]** `internal/nightowl/oncall.go` — `GetOnCallForService`
- [x] **[3.4]** `internal/nightowl/alerts.go` — `CreateAlert`
- [x] **[3.5]** `internal/nightowl/mock_test.go`
- [x] **[3.6]** `internal/bookowl/client.go`
- [x] **[3.7]** `internal/bookowl/search.go` — `SearchArticles`
- [x] **[3.8]** `internal/bookowl/articles.go` — `GetArticle`
- [x] **[3.9]** `internal/bookowl/postmortems.go` — `CreatePostMortem`
- [x] **[3.10]** `internal/bookowl/mock_test.go`
- [x] **[3.11]** Unit tests for both clients

## Phase 4 — Core Ticket Domain

- [x] **[4.1]** `internal/ticket/ticket.go` — `Ticket`, `EnrichedTicket`, `ListOptions` types
- [x] **[4.2]** `internal/ticket/store.go` — upsert and lookup for `ticket_meta`
- [x] **[4.3]** `internal/ticket/service.go` — `List`, `Get` (live from Zammad + enriched with TicketOwl metadata), `Create`, `Update`
- [x] **[4.4]** `internal/ticket/handler.go` — chi routes for all ticket endpoints
- [x] **[4.5]** Unit tests: service with mocked Zammad client and store
- [ ] **[4.6]** Integration tests: real PostgreSQL + mock Zammad server (testcontainers)

## Phase 5 — Links Domain

- [x] **[5.1]** `internal/link/link.go` — `IncidentLink`, `ArticleLink`, `PostMortemLink` types
- [x] **[5.2]** `internal/link/store.go` — CRUD for all three link tables
- [x] **[5.3]** `internal/link/service.go` — create/delete incident links (validates with NightOwl), article links (validates with BookOwl)
- [x] **[5.4]** `internal/link/handler.go` — link CRUD endpoints
- [x] **[5.5]** Unit + integration tests

## Phase 6 — SLA Domain

- [x] **[6.1]** `internal/sla/sla.go` — `Policy`, `State`, `StateLabel` types; `ComputeInput`, `ComputeResult`
- [x] **[6.2]** `internal/sla/service.go` — `ComputeState(ComputeInput)` pure function, `ApplyPolicy`, `CheckBreach`
- [x] **[6.3]** `internal/sla/store.go` — CRUD for `sla_policies`, read/write `sla_states`
- [x] **[6.4]** `internal/sla/handler.go` — SLA policy management and per-ticket SLA state endpoints
- [x] **[6.5]** Unit tests: `ComputeState` with all table-driven cases from `docs/08-sla.md`
- [ ] **[6.6]** Integration tests

## Phase 7 — Comments

- [x] **[7.1]** `internal/comment/comment.go` — `Comment`, `InternalNote`, `ThreadEntry` types
- [x] **[7.2]** `internal/comment/store.go` — CRUD for `internal_notes`
- [x] **[7.3]** `internal/comment/service.go` — `ListThread` (merges Zammad articles + internal notes sorted by time), `AddPublicReply`, `AddInternalNote`
- [x] **[7.4]** `internal/comment/handler.go`
- [x] **[7.5]** Unit + integration tests

## Phase 8 — Webhook Receiver

- [x] **[8.1]** `internal/webhook/zammad.go` — HMAC-SHA1 validation, parse payload, push to Redis stream
- [x] **[8.2]** `internal/webhook/nightowl.go` — parse NightOwl incident event, push to Redis stream
- [x] **[8.3]** `internal/webhook/handler.go` — chi routes for both
- [x] **[8.4]** Unit tests: HMAC validation (valid, tampered, wrong secret), payload parsing

## Phase 9 — Worker

- [x] **[9.1]** `internal/worker/worker.go` — main run loop, graceful shutdown with context
- [x] **[9.2]** `internal/worker/slapoller.go` — 60s tick, stale ticket query, Zammad fetch, `sla.ComputeState`, DB update, breach detection
- [x] **[9.3]** `internal/worker/eventprocessor.go` — `XREAD` from Redis stream, dispatch by event type
- [x] **[9.4]** `internal/worker/autoticket.go` — `MatchRule(rule, incident)`, `FindMatchingRules(rules, incident)`, create Zammad ticket + links on match
- [x] **[9.5]** `internal/worker/resolve.go` — incident.resolved → update Zammad ticket status
- [x] **[9.6]** `internal/notification/service.go` — `AlertSLABreach` → NightOwl `CreateAlert`
- [x] **[9.7]** Unit tests: `MatchRule` / `FindMatchingRules` table-driven; SLA poller logic
- [ ] **[9.8]** Integration tests: full event flow (testcontainers)

## Phase 10 — Runbook Suggestions & Post-Mortems

- [x] **[10.1]** `internal/ticket/suggestions.go` — extract tags and title from ticket, call BookOwl search
- [x] **[10.2]** Handler for `GET /api/v1/tickets/{id}/suggestions`
- [x] **[10.3]** `internal/ticket/postmortem.go` — build `CreatePostMortemRequest` from ticket + incident data, call BookOwl, store link
- [x] **[10.4]** Handler for `POST /api/v1/tickets/{id}/postmortem`
- [x] **[10.5]** Unit tests

## Phase 11 — Customer Portal

- [x] **[11.1]** `internal/customer/customer.go` — `Org`, `PortalTicket` types
- [x] **[11.2]** `internal/customer/store.go` — CRUD for `customer_orgs`, OIDC group lookup
- [x] **[11.3]** `internal/customer/service.go` — `ListMyTickets`, `GetMyTicket`, `AddReply`, `GetLinkedArticles`
- [x] **[11.4]** `internal/customer/handler.go` — portal endpoints with org-scoping middleware
- [x] **[11.5]** Auth middleware variant for `customer` role + org extraction
- [x] **[11.6]** Unit + integration tests (verify org A cannot see org B tickets)

## Phase 12 — Admin Endpoints

- [x] **[12.1]** `internal/admin/handler.go` — all `/api/v1/admin/` routes
- [x] **[12.2]** Zammad config CRUD + test-connection
- [x] **[12.3]** Integration keys (NightOwl, BookOwl)
- [x] **[12.4]** Customer org management
- [x] **[12.5]** Auto-ticket rule management
- [x] **[12.6]** Unit tests

## Phase 13 — Health & Observability

- [x] **[13.1]** `GET /healthz` — always 200
- [x] **[13.2]** `GET /readyz` — checks DB ping, Redis ping, Zammad reachability; 503 if any fail
- [x] **[13.3]** Prometheus counters: `ticketowl_ticket_requests_total`, `ticketowl_sla_breaches_total`, `ticketowl_zammad_request_duration_seconds`, `ticketowl_worker_poll_duration_seconds`
- [x] **[13.4]** OTel spans on all outbound client calls (Zammad, NightOwl, BookOwl)

## Phase 14 — Frontend

- [x] **[14.1]** NightOwl design tokens applied (dark mode default, `#0A2E24`, `#00e5a0`)
- [x] **[14.2]** TanStack Router authenticated shell with sidebar
- [x] **[14.3]** Typed API client `web/src/lib/api.ts` with TanStack Query hooks for all endpoints
- [x] **[14.4]** Login page (OIDC redirect)
- [x] **[14.5]** Ticket list — filter bar (status, priority, SLA state), sortable table, SLA badge
- [x] **[14.6]** Ticket detail — Zammad content pane, linked incidents panel, linked articles panel, SLA timer, internal notes, on-call badge, post-mortem button
- [x] **[14.7]** "Link incident" modal — search NightOwl incidents, select and link
- [x] **[14.8]** "Runbook suggestions" panel (auto-loads from BookOwl suggestions endpoint)
- [x] **[14.9]** Customer portal — ticket list, ticket detail (public view), reply form
- [x] **[14.10]** Admin pages — Zammad config, SLA policies, customer orgs, auto-ticket rules
- [x] **[14.11]** Dark/light mode toggle

## Phase 15 — Deployment

- [x] **[15.1]** Full Helm chart `charts/ticketowl/` (all templates from `docs/10-deployment.md`)
- [x] **[15.2]** `.github/workflows/ci.yml` — lint → test-unit → test-integration → helm-lint → build → helm-release
- [x] **[15.3]** Migration Job Helm hook
- [ ] **[15.4]** `wisbric-platform` umbrella chart updated with TicketOwl dependency

---

## Testing Strategy

### Unit tests (no external deps — run in CI without services)

Every service layer and pure function has unit tests. Use mocks/fakes for all external deps.

Key patterns:
- `internal/zammad/mock_test.go` — all ticket service tests use this
- `internal/nightowl/mock_test.go`, `internal/bookowl/mock_test.go`
- In-memory fakes for store interfaces in service tests
- `internal/sla/service_test.go` — table-driven `ComputeState` (all cases from `docs/08-sla.md`)
- `internal/worker/autoticket_test.go` — table-driven `MatchRule` / `FindMatchingRules`
- `internal/webhook/zammad_test.go` — HMAC validation + payload parsing

### Integration tests (`-tags integration`, require Docker via testcontainers)

- `internal/ticket/integration_test.go` — upsert ticket_meta → SLA policy assignment → state init
- `internal/sla/integration_test.go` — poller: insert ticket, advance time, verify breach
- `internal/worker/integration_test.go` — Redis event → handler side-effects
- `internal/customer/integration_test.go` — org scoping: org A cannot see org B tickets

### Test helpers — `internal/testhelpers/`

```
db.go       — SpinUpPostgres(t): testcontainers, run migrations, return pool + cleanup
redis.go    — SpinUpRedis(t): testcontainers, return client + cleanup
tenant.go   — CreateTestTenant(t, pool): insert tenant + create schema
fixtures.go — SampleSLAPolicy(), SampleTicketMeta(), SampleIncidentLink(), etc.
```

### Test file conventions per package

```
{domain}_test.go         — pure type/function tests
service_test.go          — service unit tests (mocked deps)
store_test.go            — store integration tests (testcontainers)
handler_test.go          — handler tests (httptest)
integration_test.go      — end-to-end (testcontainers, build tag: integration)
```

### Commands

```bash
make test                # unit tests only
make test-integration    # all tests (requires Docker)
go test ./internal/sla/... -v -run TestComputeState
go test ./... -race      # always with race detector
```

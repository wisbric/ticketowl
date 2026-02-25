# Getting Started with TicketOwl

This guide is for **Claude Code**. Follow these steps in order.

---

## 1. Prerequisites

The NightOwl and BookOwl repos should already exist alongside this one, as TicketOwl calls both of their APIs in development.

Confirm the ports aren't conflicting:

| Service       | Port  |
|---------------|-------|
| NightOwl API  | :8080 |
| BookOwl API   | :8081 |
| TicketOwl API | :8082 |
| NightOwl UI   | :3000 |
| BookOwl UI    | :3001 |
| TicketOwl UI  | :3002 |
| Zammad        | :3003 |

---

## 2. Read the Specs First

Before writing any code, read these docs in order:

```
CLAUDE.md
docs/01-requirements.md
docs/02-architecture.md
docs/03-data-model.md
docs/04-zammad-integration.md
docs/11-tasks.md
```

The remaining docs (`05` through `10`) are reference material — consult them when you reach the relevant phase.

---

## 3. Bootstrap Order

Work through `docs/11-tasks.md` phase by phase. The phases are ordered by dependency — do not skip ahead.

### Phase 0 first — get the skeleton compiling

Phase 0 must produce a Go binary that:
- Compiles cleanly (`go build ./...`)
- Passes `make lint`
- Runs `go run ./cmd/ticketowl` without panicking (even if it just prints "no mode" and exits)

Do not move to Phase 1 until Phase 0 is green.

### Phase 1 — get the DB running

After Phase 1:
```bash
docker compose up -d
make seed
```
This must succeed without errors. The `acme` tenant schema must exist with all tables.

### Phase 2 — Zammad client must have tests passing

After Phase 2:
```bash
make test
```
All Zammad client unit tests must pass using the mock server. No real Zammad needed.

### Phases 3–13 — build out and test each domain

Follow `docs/11-tasks.md` sequentially. Run `make test` and `make lint` after each phase.

### Phase 14 — frontend last

The frontend (Phase 14) is intentionally last. All API endpoints must be complete and tested before building the UI against them.

---

## 4. Development Loop

```bash
# Start dependencies
make up

# Seed the dev tenant (run once, or after make clean)
make seed

# Run API server (in one terminal)
make api

# Run worker (in another terminal)
make worker

# Run frontend (in a third terminal)
make web

# Before committing
make lint
make test
```

---

## 5. Key Dev Credentials

| Thing | Value |
|-------|-------|
| TicketOwl API URL | http://localhost:8082 |
| TicketOwl UI URL | http://localhost:3002/login |
| Dev API key | `to_dev_seed_key_do_not_use_in_production` |
| Local admin user | `admin` |
| Local admin password | `ticketowl-admin` |
| DB DSN | `postgres://ticketowl:ticketowl@localhost:5434/ticketowl?sslmode=disable` |
| Redis URL | `redis://localhost:6381` |
| Zammad URL | `http://localhost:3003` |
| NightOwl API key (for dev) | `ow_dev_seed_key_do_not_use_in_production` |
| BookOwl API key (for dev) | `bw_dev_seed_key_do_not_use_in_production` |

---

## 6. What Claude Code Should Not Do

- Do not write Zammad API calls without reading `docs/04-zammad-integration.md` first
- Do not bypass the tenant middleware for any DB query
- Do not store ticket content (subject, description, comments) in TicketOwl's DB
- Do not call real NightOwl or BookOwl APIs in tests — use mock servers
- Do not cache Zammad ticket content
- Do not skip writing tests for a phase — each phase's tests are part of the definition of done

---

## 7. Commit Convention

```
feat(ticket): list and detail endpoints [4.3]
fix(sla): pause accumulation off-by-one [6.2]
test(webhook): HMAC validation table cases [8.4]
chore(scaffold): go.mod, Makefile, Dockerfile [0.1-0.13]
```

Task IDs in brackets refer to `docs/11-tasks.md`.

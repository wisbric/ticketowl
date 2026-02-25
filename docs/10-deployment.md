# 10 — Deployment

## Overview

TicketOwl is deployed as part of the shared Wisbric platform Helm chart. Two Kubernetes Deployments from one image: `ticketowl-api` (mode=api) and `ticketowl-worker` (mode=worker).

---

## Docker Image

- Multi-stage build: `golang:1.25-alpine` builder → `gcr.io/distroless/static-debian12` runtime
- Binary: `/ticketowl`
- `CGO_ENABLED=0`, `-trimpath`, version from `git describe` embedded in ldflags
- `EXPOSE 8082`
- Migrations embedded via `go:embed` in the binary

---

## Environment Variables

Prefix: `TICKETOWL_`

| Variable | Default | Description |
|---|---|---|
| `TICKETOWL_MODE` | `api` | `api` \| `worker` \| `seed` \| `seed-demo` \| `migrate` |
| `TICKETOWL_PORT` | `8082` | HTTP listen port |
| `TICKETOWL_LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `TICKETOWL_LOG_FORMAT` | `json` | `json` \| `text` |
| `TICKETOWL_DB_URL` | — | PostgreSQL DSN |
| `TICKETOWL_REDIS_URL` | — | Redis URL |
| `TICKETOWL_OIDC_ISSUER` | — | Keycloak issuer URL |
| `TICKETOWL_OIDC_CLIENT_ID` | — | OIDC client ID |
| `TICKETOWL_ENCRYPTION_KEY` | — | 32-byte hex AES key for credential encryption |
| `TICKETOWL_NIGHTOWL_API_URL` | — | NightOwl base URL |
| `TICKETOWL_NIGHTOWL_API_KEY` | — | NightOwl service API key |
| `TICKETOWL_BOOKOWL_API_URL` | — | BookOwl base URL |
| `TICKETOWL_BOOKOWL_API_KEY` | — | BookOwl service API key |
| `TICKETOWL_OTEL_ENDPOINT` | — | OTLP gRPC endpoint (empty = disabled) |
| `TICKETOWL_WORKER_POLL_SECONDS` | `60` | SLA polling interval |
| `TICKETOWL_DEV_MODE` | `false` | Enables local admin and dev API key |

---

## Docker Compose (Development)

Ports are offset to avoid conflicts with NightOwl (5432/6379) and BookOwl (5433/6380):

```yaml
services:
  postgres:       # port 5434
  redis:          # port 6381
  zammad:         # port 3003 (requires its own postgres + redis)
  zammad-postgres:
  zammad-redis:
```

Note: Zammad requires its own dedicated PostgreSQL and Redis instances.

---

## Makefile Targets

```
make dev            — docker compose up + API server
make api            — API server only (deps must be running)
make worker         — worker only
make seed           — create "acme" dev tenant (idempotent)
make seed-demo      — seed demo tickets
make web            — frontend dev server (cd web && npm run dev)
make test           — unit tests (no Docker required)
make test-integration — all tests including integration (testcontainers)
make lint           — golangci-lint
make build          — compile binary to bin/ticketowl
make docker         — build ticketowl:dev image
make up / down / clean — docker compose lifecycle
make helm-lint      — lint the Helm chart
make helm-template  — dry-run template render
```

---

## Helm Chart — `charts/ticketowl/`

```
charts/ticketowl/
  Chart.yaml
  values.yaml
  templates/
    _helpers.tpl
    deployment-api.yaml
    deployment-worker.yaml
    service.yaml
    ingress.yaml
    configmap.yaml
    secret.yaml
    serviceaccount.yaml
    hpa.yaml
    pdb.yaml
    job-migrate.yaml    ← pre-install/pre-upgrade hook
```

### Key values.yaml structure

```yaml
image:
  repository: ghcr.io/wisbric/ticketowl
  tag: ""           # defaults to Chart.AppVersion

api:
  replicaCount: 2
  autoscaling:
    enabled: false

worker:
  replicaCount: 1   # single replica — no leader election in v1

service:
  type: ClusterIP
  port: 8082

ingress:
  enabled: false
  className: nginx

existingSecret: ""  # reference pre-created Secret, else chart creates one
secrets:
  dbUrl: ""
  redisUrl: ""
  oidcIssuer: ""
  oidcClientId: ""
  encryptionKey: ""
  nightowlApiKey: ""
  bookowlApiKey: ""

podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

### Worker deployment notes

- `strategy.rollingUpdate.maxSurge: 0` — prevents two workers running simultaneously during rollout (no leader election in v1)
- Worker liveness probe on port 9090 (separate from API)

### Migration Job

Runs as a Helm pre-install/pre-upgrade hook. Uses `helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded`. Calls binary with `--mode=migrate` which runs all pending migrations and exits.

---

## CI/CD — `.github/workflows/ci.yml`

```
lint → test-unit → test-integration → helm-lint → build+push → helm-release
```

- Unit tests: no external services required
- Integration tests: GitHub Actions service containers (postgres:16-alpine, redis:7-alpine)
- Build: only on `main` branch push
- Helm release: pushes chart to `oci://ghcr.io/wisbric/charts`

---

## Umbrella Chart — `wisbric-platform`

TicketOwl is a dependency in the platform umbrella chart alongside NightOwl, BookOwl, PostgreSQL, Redis, Keycloak, and Zammad:

```yaml
# wisbric-platform/Chart.yaml
dependencies:
  - name: nightowl
    repository: oci://ghcr.io/wisbric/charts
  - name: bookowl
    repository: oci://ghcr.io/wisbric/charts
  - name: ticketowl
    repository: oci://ghcr.io/wisbric/charts
  - name: zammad
    repository: https://zammad.github.io/zammad-helm
  - name: postgresql
    repository: https://charts.bitnami.com/bitnami
  - name: redis
    repository: https://charts.bitnami.com/bitnami
  - name: keycloak
    repository: https://charts.bitnami.com/bitnami
```

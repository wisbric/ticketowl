# 01 — Requirements

## Product Vision

TicketOwl is the support operations layer of the NightOwl platform. It gives MSPs a single pane of glass for customer tickets, wired directly into the incident and knowledge workflows their operations teams already live in.

The core insight: for an MSP running Kubernetes infrastructure, a customer ticket is almost never isolated — it has an incident history, a runbook, an on-call owner, and an SLA clock. TicketOwl makes all of that visible and actionable without forcing agents to jump between Zammad, NightOwl, and BookOwl manually.

---

## Users

**Internal Agent** — MSP engineer or support analyst. Handles tickets day-to-day. Needs incident context, runbook suggestions, SLA visibility, and the ability to escalate to on-call when a ticket breaches SLA.

**Internal Team Lead / Manager** — Monitors queue health, SLA compliance, and team workload.

**External Customer** — Contact at an organisation the MSP supports. Can view their own tickets, add comments, and see linked KB articles. Cannot see internal notes, incidents, or other customers' data.

**Platform Admin** — Configures tenant settings: Zammad connection, OIDC, SLA definitions, NightOwl/BookOwl API keys.

---

## Functional Requirements

### F1 — Ticket Management (Core)

| ID   | Requirement |
|------|-------------|
| F1.1 | List tickets with filtering by status, priority, assignee, customer org, and SLA state |
| F1.2 | View ticket detail: subject, description, status, priority, assignee, SLA timer, full comment thread |
| F1.3 | Create ticket manually (internal or on behalf of customer) |
| F1.4 | Update ticket status, priority, and assignee |
| F1.5 | Add internal note (not visible to customer) or public reply |
| F1.6 | Close and reopen tickets |
| F1.7 | Ticket search (full-text across subject, description, comments) |
| F1.8 | All ticket state is stored in and sourced from Zammad — TicketOwl does not duplicate ticket content |

### F2 — NightOwl Integration

| ID   | Requirement |
|------|-------------|
| F2.1 | Link a ticket to one or more NightOwl incidents |
| F2.2 | Display linked incident summary inline on ticket detail (severity, status, timeline, assignee) |
| F2.3 | Automatically create a Zammad ticket when a NightOwl incident is created (configurable per tenant) |
| F2.4 | Automatically update ticket status when linked incident is resolved |
| F2.5 | Trigger a NightOwl alert when a ticket breaches SLA (calls NightOwl alert ingest endpoint) |
| F2.6 | Show current on-call person for the ticket's service on the ticket detail view |

### F3 — BookOwl Integration

| ID   | Requirement |
|------|-------------|
| F3.1 | Surface relevant BookOwl articles and runbooks inline on ticket detail, based on ticket tags and service |
| F3.2 | Agent can attach a specific BookOwl article to a ticket (stored as a link) |
| F3.3 | On NightOwl incident resolution linked to a ticket, offer to create a BookOwl post-mortem draft |
| F3.4 | Linked post-mortem URL is shown on the ticket detail |

### F4 — Customer Portal

| ID   | Requirement |
|------|-------------|
| F4.1 | Customers authenticate via OIDC (same Keycloak, customer-scoped client) |
| F4.2 | Customer sees only tickets belonging to their organisation |
| F4.3 | Customer can view ticket status, priority, SLA remaining, and the public comment thread |
| F4.4 | Customer can add a public reply to their own ticket |
| F4.5 | Customer can view BookOwl KB articles linked to their ticket |
| F4.6 | Customer cannot see internal notes, incident links, on-call info, or other orgs' tickets |
| F4.7 | Customer cannot create tickets via the portal in v1 (email ingest via Zammad handles creation) |

### F5 — SLA Management

| ID   | Requirement |
|------|-------------|
| F5.1 | Tenant admin defines SLA policies: response and resolution times per priority level |
| F5.2 | Worker polls open tickets and computes SLA remaining from Zammad ticket data |
| F5.3 | SLA state (`on_track` / `warning` / `breached` / `met`) stored in TicketOwl DB and shown in UI |
| F5.4 | Warning threshold is configurable (default: 20% of SLA time remaining) |
| F5.5 | On breach: fire NightOwl alert and mark ticket in TicketOwl DB |
| F5.6 | SLA clock pauses when ticket is in a configured "pending customer" status |

### F6 — Zammad Sync

| ID   | Requirement |
|------|-------------|
| F6.1 | TicketOwl receives Zammad webhook events for ticket create, update, and comment |
| F6.2 | On receiving a Zammad event, TicketOwl updates SLA state and link metadata |
| F6.3 | TicketOwl never stores ticket content — always fetches live from Zammad API |
| F6.4 | Zammad connection details (URL, API token) are stored per tenant |
| F6.5 | Connection health surfaced in admin UI |

### F7 — Notifications

| ID   | Requirement |
|------|-------------|
| F7.1 | Email notifications for ticket updates handled entirely by Zammad |
| F7.2 | SLA breach notifications handled by NightOwl (triggered by F2.5) |
| F7.3 | TicketOwl does not send notifications directly in v1 |

### F8 — Admin & Configuration

| ID   | Requirement |
|------|-------------|
| F8.1 | Tenant admin UI: Zammad connection settings (URL, API token, test connection) |
| F8.2 | Tenant admin UI: SLA policy editor |
| F8.3 | Tenant admin UI: NightOwl and BookOwl API key configuration |
| F8.4 | Tenant admin UI: auto-ticket rules (which NightOwl alert groups trigger ticket creation) |
| F8.5 | Tenant admin UI: customer org management (map customer OIDC group → org) |
| F8.6 | Global admin: tenant provisioning |

---

## Non-Functional Requirements

| ID  | Requirement |
|-----|-------------|
| N1  | Multi-tenant: schema-per-tenant isolation, identical to NightOwl/BookOwl |
| N2  | All API responses < 500ms at p99 under normal load |
| N3  | Worker SLA polling cycle ≤ 60 seconds |
| N4  | Zammad API calls retried with exponential backoff (max 3 attempts) |
| N5  | OIDC token validation on every request; API keys validated via SHA-256 hash lookup |
| N6  | Audit log for all write operations |
| N7  | Prometheus metrics at `/metrics` |
| N8  | OpenTelemetry traces via OTLP gRPC |
| N9  | Helm chart for Kubernetes deployment |

---

## Out of Scope (v1)

- Native mobile app
- Customer ticket creation via portal (Zammad email ingest handles this)
- Direct Slack/Teams integration (NightOwl handles notification routing)
- Custom ticket fields or dynamic forms
- Time tracking / billing
- AI-generated ticket summaries or auto-categorisation
- Self-hosted Zammad provisioning

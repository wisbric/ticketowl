# 07 — Customer Portal

## Overview

The customer portal gives external contacts at MSP-managed organisations a scoped, read-mostly view of their own tickets. It is served from the same TicketOwl binary under the `/portal` route prefix.

---

## Authentication

Customers authenticate via OIDC using a Keycloak client configured for external users. The JWT includes:
- `ticketowl_role: customer`
- `tenant: <slug>` — which tenant they belong to
- `org_id: <uuid>` — which customer org within that tenant

The `org_id` is mapped from a Keycloak group to a `customer_orgs` row by the tenant admin.

A customer with no matching `customer_orgs` row is rejected with 403.

---

## Scoping

All portal endpoints enforce org scoping at the store layer. The `ListTicketsOptions.OrgID` field is always set from the authenticated user's `org_id` — it cannot be overridden by query parameters.

The Zammad `organization_id` on the ticket is used for filtering: `customer_orgs.zammad_org_id` is the bridge between TicketOwl's org concept and Zammad's organisation system.

---

## Portal Endpoints

### Ticket list — `GET /api/v1/portal/tickets`

Returns tickets belonging to the customer's org. Includes:
- `id`, `number`, `title`, `status`, `priority`
- `sla_state` (from TicketOwl `sla_states`)
- `created_at`, `updated_at`

Does **not** include: assignee details, linked incidents, internal notes.

### Ticket detail — `GET /api/v1/portal/tickets/{id}`

Returns the public view of a ticket. Includes:
- All fields from ticket list
- The public comment thread (Zammad articles where `internal = false`)
- Linked BookOwl articles (from `article_links`)
- SLA due times (response and resolution)

Does **not** include: internal notes, incident links, on-call info.

### Reply — `POST /api/v1/portal/tickets/{id}/reply`

Adds a public reply to the ticket. Proxied to Zammad as an article with `internal: false`, `sender: Customer`.

Customers can only reply to tickets belonging to their org.

### Linked articles — `GET /api/v1/portal/tickets/{id}/articles`

Returns the list of BookOwl articles linked to the ticket (same as agent view — articles are always public).

---

## What Customers Cannot See

- Internal notes (stored in TicketOwl `internal_notes` table)
- NightOwl incident links
- On-call person
- Other customers' tickets (enforced at store layer — no frontend trust)
- Ticket assignee details (by default — can be toggled per tenant in future)

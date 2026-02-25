# 03 — Data Model

## Overview

Schema-per-tenant isolation, identical to NightOwl and BookOwl. One global schema (`global`) and one schema per tenant (e.g., `tenant_acme`).

**TicketOwl does not store ticket content.** It stores only: configuration, link metadata, SLA state, internal notes, and audit log. All ticket subject/description/comment data comes live from Zammad.

---

## Global Schema

```sql
CREATE TABLE global.tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    suspended   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- SHA-256 hashes of API keys for service-to-service auth
CREATE TABLE global.api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES global.tenants(id) ON DELETE CASCADE,
    key_hash     TEXT NOT NULL UNIQUE,
    description  TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);
```

---

## Tenant Schema

All tables live in `tenant_{slug}`. `search_path` is set per-request by middleware.

### Configuration

```sql
CREATE TABLE zammad_config (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url            TEXT NOT NULL,
    api_token      TEXT NOT NULL,           -- encrypted AES-256-GCM
    webhook_secret TEXT NOT NULL,           -- encrypted AES-256-GCM
    pause_statuses TEXT[] NOT NULL DEFAULT ARRAY['pending customer'],
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by     UUID
);

CREATE TABLE integration_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service    TEXT NOT NULL UNIQUE,        -- 'nightowl' | 'bookowl'
    api_key    TEXT NOT NULL,               -- encrypted AES-256-GCM
    api_url    TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### SLA Policies

```sql
CREATE TABLE sla_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    priority            TEXT NOT NULL,      -- 'low' | 'normal' | 'high' | 'critical'
    response_minutes    INTEGER NOT NULL,
    resolution_minutes  INTEGER NOT NULL,
    warning_threshold   NUMERIC(4,3) NOT NULL DEFAULT 0.20,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (priority, is_default) DEFERRABLE
);
```

### Customer Organisations

```sql
CREATE TABLE customer_orgs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    oidc_group     TEXT NOT NULL UNIQUE,    -- Keycloak group → this org
    zammad_org_id  INTEGER,                 -- Zammad organisation ID for scoping
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Auto-Ticket Rules

```sql
CREATE TABLE auto_ticket_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    alert_group      TEXT,                  -- exact or prefix match (e.g. 'kubernetes-')
    min_severity     TEXT,                  -- 'low'|'medium'|'high'|'critical'
    default_priority TEXT NOT NULL DEFAULT 'normal',
    default_group    TEXT,                  -- Zammad group name
    title_template   TEXT NOT NULL DEFAULT '{{.AlertName}}: {{.Summary}}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Ticket Metadata

```sql
-- One row per Zammad ticket known to TicketOwl.
-- Zammad is source of truth for content; this is metadata only.
CREATE TABLE ticket_meta (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zammad_id     INTEGER NOT NULL UNIQUE,
    zammad_number TEXT NOT NULL UNIQUE,
    sla_policy_id UUID REFERENCES sla_policies(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON ticket_meta (zammad_id);
```

### SLA State

```sql
CREATE TABLE sla_states (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_meta_id          UUID NOT NULL UNIQUE REFERENCES ticket_meta(id) ON DELETE CASCADE,
    response_due_at         TIMESTAMPTZ,
    resolution_due_at       TIMESTAMPTZ,
    response_met_at         TIMESTAMPTZ,
    first_breach_alerted_at TIMESTAMPTZ,
    state                   TEXT NOT NULL DEFAULT 'on_track',
    paused                  BOOLEAN NOT NULL DEFAULT false,
    paused_at               TIMESTAMPTZ,
    accumulated_pause_secs  INTEGER NOT NULL DEFAULT 0,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Links

```sql
CREATE TABLE incident_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_meta_id UUID NOT NULL REFERENCES ticket_meta(id) ON DELETE CASCADE,
    incident_id    UUID NOT NULL,
    incident_slug  TEXT NOT NULL,
    linked_by      UUID NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ticket_meta_id, incident_id)
);

CREATE INDEX ON incident_links (incident_id);

CREATE TABLE article_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_meta_id UUID NOT NULL REFERENCES ticket_meta(id) ON DELETE CASCADE,
    article_id     UUID NOT NULL,
    article_slug   TEXT NOT NULL,
    article_title  TEXT NOT NULL,
    linked_by      UUID NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ticket_meta_id, article_id)
);

CREATE TABLE postmortem_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_meta_id UUID NOT NULL UNIQUE REFERENCES ticket_meta(id) ON DELETE CASCADE,
    postmortem_id  UUID NOT NULL,
    postmortem_url TEXT NOT NULL,
    created_by     UUID NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Internal Notes

```sql
-- Internal notes written in TicketOwl, never synced to Zammad.
-- Public comments are proxied from Zammad.
CREATE TABLE internal_notes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_meta_id UUID NOT NULL REFERENCES ticket_meta(id) ON DELETE CASCADE,
    author_id      UUID NOT NULL,
    author_name    TEXT NOT NULL,
    body           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON internal_notes (ticket_meta_id, created_at DESC);
```

### Audit Log

```sql
CREATE TABLE audit_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id      UUID,
    actor_name    TEXT,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON audit_log (resource_type, resource_id);
CREATE INDEX ON audit_log (created_at DESC);
```

---

## Migration Layout

```
migrations/
  global/
    0001_create_tenants.up.sql / .down.sql
    0002_create_api_keys.up.sql / .down.sql
  tenant/
    0001_create_zammad_config.up.sql / .down.sql
    0002_create_integration_keys.up.sql / .down.sql
    0003_create_sla_policies.up.sql / .down.sql
    0004_create_customer_orgs.up.sql / .down.sql
    0005_create_auto_ticket_rules.up.sql / .down.sql
    0006_create_ticket_meta.up.sql / .down.sql
    0007_create_sla_states.up.sql / .down.sql
    0008_create_links.up.sql / .down.sql
    0009_create_internal_notes.up.sql / .down.sql
    0010_create_audit_log.up.sql / .down.sql
```

Tenant migrations run when a new tenant is provisioned. Global migrations run once at startup.

---

## Encryption

`zammad_config.api_token`, `zammad_config.webhook_secret`, and `integration_keys.api_key` are encrypted at the application layer using AES-256-GCM before write, decrypted on read. Key provided via `TICKETOWL_ENCRYPTION_KEY` (32-byte hex). Same pattern as NightOwl.

CREATE TABLE zammad_config (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url            TEXT NOT NULL,
    api_token      TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    pause_statuses TEXT[] NOT NULL DEFAULT ARRAY['pending customer'],
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by     UUID
);

CREATE TABLE integration_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service    TEXT NOT NULL UNIQUE,
    api_key    TEXT NOT NULL,
    api_url    TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sla_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    priority            TEXT NOT NULL,
    response_minutes    INTEGER NOT NULL,
    resolution_minutes  INTEGER NOT NULL,
    warning_threshold   NUMERIC(4,3) NOT NULL DEFAULT 0.20,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (priority, is_default) DEFERRABLE
);

CREATE TABLE customer_orgs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    oidc_group     TEXT NOT NULL UNIQUE,
    zammad_org_id  INTEGER,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE auto_ticket_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    alert_group      TEXT,
    min_severity     TEXT,
    default_priority TEXT NOT NULL DEFAULT 'normal',
    default_group    TEXT,
    title_template   TEXT NOT NULL DEFAULT '{{.AlertName}}: {{.Summary}}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_meta (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zammad_id     INTEGER NOT NULL UNIQUE,
    zammad_number TEXT NOT NULL UNIQUE,
    sla_policy_id UUID REFERENCES sla_policies(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON ticket_meta (zammad_id);

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

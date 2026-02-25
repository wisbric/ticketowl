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

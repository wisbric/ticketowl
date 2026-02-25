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

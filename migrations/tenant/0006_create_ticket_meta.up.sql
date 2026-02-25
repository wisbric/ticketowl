CREATE TABLE ticket_meta (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zammad_id     INTEGER NOT NULL UNIQUE,
    zammad_number TEXT NOT NULL UNIQUE,
    sla_policy_id UUID REFERENCES sla_policies(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON ticket_meta (zammad_id);

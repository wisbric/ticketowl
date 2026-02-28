CREATE TABLE group_roster_mappings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zammad_group          TEXT NOT NULL,
    roster_id             TEXT NOT NULL,
    auto_assign           BOOLEAN NOT NULL DEFAULT true,
    escalate_to_secondary BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (zammad_group)
);

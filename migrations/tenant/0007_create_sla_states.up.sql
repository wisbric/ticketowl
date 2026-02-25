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

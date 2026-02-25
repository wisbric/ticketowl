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

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

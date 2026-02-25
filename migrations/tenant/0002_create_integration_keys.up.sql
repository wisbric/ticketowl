CREATE TABLE integration_keys (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service    TEXT NOT NULL UNIQUE,
    api_key    TEXT NOT NULL,
    api_url    TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

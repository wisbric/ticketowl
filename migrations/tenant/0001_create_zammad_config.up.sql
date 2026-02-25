CREATE TABLE zammad_config (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url            TEXT NOT NULL,
    api_token      TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    pause_statuses TEXT[] NOT NULL DEFAULT ARRAY['pending customer'],
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by     UUID
);

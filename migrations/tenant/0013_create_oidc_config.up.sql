CREATE TABLE oidc_config (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_url      TEXT NOT NULL,
    client_id       TEXT NOT NULL,
    client_secret   TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    tested_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE public.local_admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL UNIQUE REFERENCES global.tenants(id) ON DELETE CASCADE,
    username        TEXT NOT NULL DEFAULT 'admin',
    password_hash   TEXT NOT NULL,
    must_change     BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_local_admins_tenant ON public.local_admins(tenant_id);

-- name: GetTenantBySlug :one
SELECT id, slug, name, suspended, created_at, updated_at
FROM global.tenants
WHERE slug = $1;

-- name: GetTenant :one
SELECT id, slug, name, suspended, created_at, updated_at
FROM global.tenants
WHERE id = $1;

-- name: ListTenants :many
SELECT id, slug, name, suspended, created_at, updated_at
FROM global.tenants
ORDER BY name;

-- name: CreateTenant :one
INSERT INTO global.tenants (slug, name)
VALUES ($1, $2)
RETURNING id, slug, name, suspended, created_at, updated_at;

-- name: GetAPIKeyByHash :one
SELECT id, tenant_id, key_hash, description, created_at, last_used_at
FROM global.api_keys
WHERE key_hash = $1;

-- name: CreateAPIKey :one
INSERT INTO global.api_keys (tenant_id, key_hash, description)
VALUES ($1, $2, $3)
RETURNING id, tenant_id, key_hash, description, created_at, last_used_at;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE global.api_keys
SET last_used_at = now()
WHERE id = $1;

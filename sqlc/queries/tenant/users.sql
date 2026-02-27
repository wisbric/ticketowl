-- name: GetUserByExternalID :one
SELECT * FROM users WHERE external_id = $1;

-- name: UpsertUser :one
INSERT INTO users (external_id, email, display_name, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (external_id) DO UPDATE SET
    email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    updated_at = now()
RETURNING *;

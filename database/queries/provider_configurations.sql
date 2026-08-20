-- name: GetClientProviderProfile :one
SELECT p.client_id, p.demo_mode, p.version, p.created_by, p.updated_by, p.created_at, p.updated_at
FROM client_provider_profiles p
JOIN clients c ON c.id = p.client_id
WHERE p.client_id = sqlc.arg(client_id);

-- name: UpsertClientProviderProfile :one
INSERT INTO client_provider_profiles (client_id, demo_mode, created_by, updated_by)
VALUES (sqlc.arg(client_id), sqlc.arg(demo_mode), sqlc.arg(actor_id), sqlc.arg(actor_id))
ON CONFLICT (client_id) DO UPDATE
SET demo_mode = EXCLUDED.demo_mode,
    version = client_provider_profiles.version + 1,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
WHERE client_provider_profiles.version = sqlc.arg(version)
RETURNING *;

-- name: ListProviderConfigurationsByClient :many
SELECT pc.*
FROM provider_configurations pc
JOIN clients c ON c.id = pc.client_id
WHERE pc.client_id = sqlc.arg(client_id)
ORDER BY pc.provider;

-- name: GetProviderConfiguration :one
SELECT pc.*
FROM provider_configurations pc
JOIN clients c ON c.id = pc.client_id
WHERE pc.client_id = sqlc.arg(client_id)
  AND pc.provider = sqlc.arg(provider);

-- name: UpsertProviderConfiguration :one
INSERT INTO provider_configurations (
  client_id, provider, enabled, safe_config, secret_ciphertext, secret_nonce,
  configured_secret_fields, created_by, updated_by
)
VALUES (
  sqlc.arg(client_id), sqlc.arg(provider), sqlc.arg(enabled), sqlc.arg(safe_config),
  sqlc.arg(secret_ciphertext), sqlc.arg(secret_nonce), sqlc.arg(configured_secret_fields),
  sqlc.arg(actor_id), sqlc.arg(actor_id)
)
ON CONFLICT (client_id, provider) DO UPDATE
SET enabled = EXCLUDED.enabled,
    safe_config = EXCLUDED.safe_config,
    secret_ciphertext = EXCLUDED.secret_ciphertext,
    secret_nonce = EXCLUDED.secret_nonce,
    configured_secret_fields = EXCLUDED.configured_secret_fields,
    version = provider_configurations.version + 1,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
WHERE provider_configurations.version = sqlc.arg(version)
RETURNING *;

-- name: GetInternalUserByEmail :one
SELECT * FROM internal_users WHERE lower(email) = lower(sqlc.arg(email)) LIMIT 1;

-- name: GetInternalUserByID :one
SELECT * FROM internal_users WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetInternalUserByIDForUpdate :one
SELECT * FROM internal_users WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: CountInternalUsers :one
SELECT count(*) FROM internal_users;

-- name: CountAdminUsers :one
SELECT count(*) FROM internal_users WHERE role = 'ADMIN';

-- name: LockInternalUsersForAdminBootstrap :exec
LOCK TABLE internal_users IN SHARE ROW EXCLUSIVE MODE;

-- name: ListInternalUsers :many
SELECT * FROM internal_users
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreateInternalUser :one
INSERT INTO internal_users (email, display_name, password_hash, role, requires_password_change)
VALUES (lower(sqlc.arg(email)), sqlc.arg(display_name), sqlc.arg(password_hash), sqlc.arg(role), sqlc.arg(requires_password_change))
RETURNING *;

-- name: RecordFailedLogin :exec
UPDATE internal_users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE WHEN failed_login_attempts + 1 >= sqlc.arg(lock_threshold) THEN now() + sqlc.arg(lock_duration)::interval ELSE locked_until END,
    updated_at = now()
WHERE id = sqlc.arg(id);

-- name: RecordSuccessfulLogin :exec
UPDATE internal_users
SET failed_login_attempts = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: UpdateInternalUserPasswordVersioned :one
UPDATE internal_users
SET password_hash = sqlc.arg(password_hash), requires_password_change = sqlc.arg(requires_password_change),
    password_changed_at = now(), version = version + 1, updated_at = now()
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: UpdateInternalUserProfileVersioned :one
UPDATE internal_users
SET email = lower(sqlc.arg(email)), display_name = sqlc.arg(display_name), role = sqlc.arg(role),
    version = version + 1, updated_at = now()
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: SetInternalUserStatusVersioned :one
UPDATE internal_users
SET status = sqlc.arg(status), version = version + 1, updated_at = now()
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: LockInternalAdminUsers :many
SELECT id FROM internal_users WHERE role = 'ADMIN' FOR UPDATE;

-- name: CountOtherActiveAdmins :one
SELECT count(*) FROM internal_users
WHERE role = 'ADMIN' AND status = 'ACTIVE' AND id <> sqlc.arg(excluded_id);

-- name: CreateSession :one
INSERT INTO sessions (internal_user_id, token_hash, csrf_hash, ip_address, user_agent, expires_at)
VALUES (sqlc.arg(internal_user_id), sqlc.arg(token_hash), sqlc.arg(csrf_hash), sqlc.narg(ip_address), sqlc.arg(user_agent), sqlc.arg(expires_at))
RETURNING *;

-- name: GetActiveSessionByTokenHash :one
SELECT s.id AS session_id, s.internal_user_id, s.csrf_hash, s.expires_at, s.last_seen_at,
       u.email, u.display_name, u.role, u.status, u.requires_password_change, u.version,
       u.created_at AS user_created_at, u.updated_at AS user_updated_at, u.last_login_at
FROM sessions s
JOIN internal_users u ON u.id = s.internal_user_id
WHERE s.token_hash = sqlc.arg(token_hash)
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND u.status = 'ACTIVE'
LIMIT 1;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now() WHERE id = sqlc.arg(id) AND last_seen_at < now() - interval '5 minutes';

-- name: RevokeSession :execrows
UPDATE sessions SET revoked_at = now(), revoke_reason = sqlc.arg(reason)
WHERE id = sqlc.arg(id) AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :execrows
UPDATE sessions SET revoked_at = now(), revoke_reason = sqlc.arg(reason)
WHERE internal_user_id = sqlc.arg(internal_user_id) AND revoked_at IS NULL;

-- name: RevokeOtherUserSessions :execrows
UPDATE sessions SET revoked_at = now(), revoke_reason = sqlc.arg(reason)
WHERE internal_user_id = sqlc.arg(internal_user_id)
  AND id <> sqlc.arg(current_session_id)
  AND revoked_at IS NULL;

-- name: ListActiveUserSessions :many
SELECT id, internal_user_id, ip_address, user_agent, expires_at, last_seen_at, created_at
FROM sessions
WHERE internal_user_id = sqlc.arg(internal_user_id)
  AND revoked_at IS NULL
  AND expires_at > now()
ORDER BY last_seen_at DESC, created_at DESC;

-- name: RevokeUserSession :execrows
UPDATE sessions SET revoked_at = now(), revoke_reason = sqlc.arg(reason)
WHERE id = sqlc.arg(id)
  AND internal_user_id = sqlc.arg(internal_user_id)
  AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < now() - interval '7 days';

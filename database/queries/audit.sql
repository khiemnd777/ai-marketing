-- name: InsertAuditLog :one
INSERT INTO audit_logs (
  actor_internal_user_id, action, entity_type, entity_id, client_id, workspace_id,
  request_id, ip_address, user_agent, outcome, reason, before_data, after_data, metadata
) VALUES (
  sqlc.narg(actor_internal_user_id), sqlc.arg(action), sqlc.narg(entity_type), sqlc.narg(entity_id),
  sqlc.narg(client_id), sqlc.narg(workspace_id), sqlc.arg(request_id), sqlc.narg(ip_address),
  sqlc.arg(user_agent), sqlc.arg(outcome), sqlc.narg(reason), sqlc.narg(before_data),
  sqlc.narg(after_data), sqlc.arg(metadata)
)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE (sqlc.narg(workspace_id)::uuid IS NULL OR workspace_id = sqlc.narg(workspace_id))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

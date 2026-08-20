---
name: studio-database-change
description: Implement AI Product Marketing Studio persistence changes involving PostgreSQL 18 schema, Atlas migrations, sqlc queries, transactions, repositories, or River jobs. Use for any durable data-shape or query change, not for read-only database inspection alone.
---

# Studio Database Change

Keep the migration history, canonical schema, generated queries, application behavior, and tests synchronized.

## Inspect the persistence boundary

1. Read sqlc.yaml, atlas.hcl, the latest relevant migration, database/schema/schema.sql, and the owning query/service before editing.
2. Identify tenant keys, foreign keys, lifecycle/status values, optimistic version columns, immutable hashes, audit requirements, and jobs affected by the change.
3. Determine whether old rows require a safe backfill and whether rolling application versions can coexist during deployment.

## Apply the change

- Add an ordered Atlas versioned migration; never rewrite a migration that may have been applied.
- Keep database/schema/schema.sql at the migration head and refresh database/migrations/atlas.sum with the repository-pinned Atlas workflow.
- Update database/queries, regenerate services/api/internal/gen/db with the existing sqlc version, and never hand-edit generated files.
- Thread new query shapes through the owning service/repository and OpenAPI/UI only when they cross those boundaries.
- Require both client_id and workspace_id for business-record reads and writes. Test cross-workspace concealment, not only happy-path filtering.
- Use one transaction for state plus audit/job/idempotency/approval changes that must be atomic. Prefer explicit locks or optimistic versions when concurrent writers matter; add a concurrency test for the invariant.
- Preserve append-only records for audit, approvals, usage/cost, and provider attempts. Corrections append or compensate instead of erasing history.
- Use River for durable asynchronous work and PostgreSQL for coordination; do not introduce another state or queue system.

## Verify against PostgreSQL 18

- Validate Atlas checksums and replay every migration on a clean PostgreSQL 18 database.
- Run the relevant Testcontainers or integration workflow for constraints, transactions, locking, scoping, and River behavior.
- Run go test ./..., go test -race ./... when concurrency changed, and bun run openapi:check when the data crosses HTTP.
- Verify both upgrade behavior and a clean bootstrap. Do not claim success from sqlc generate or compilation alone.

Never point destructive migration, truncate, reset, or volume-deletion commands at a shared or unidentified database. Resolve the exact target and obtain confirmation when data loss is possible.

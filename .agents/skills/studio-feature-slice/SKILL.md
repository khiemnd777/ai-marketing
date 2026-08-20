---
name: studio-feature-slice
description: Implement or modify an AI Product Marketing Studio feature that crosses the Go API, OpenAPI contract, generated TypeScript client, and Next.js UI. Use for user-visible workflows or contract changes; prefer a narrower Studio skill for changes confined to one subsystem.
---

# Studio Feature Slice

Deliver the smallest production-complete vertical slice. Do not leave placeholder handlers, dead controls, fake success, hand-copied frontend DTOs, or unregistered modules.

## Establish the slice

1. Read `AGENTS.md`, the relevant section of `docs/IMPLEMENTATION_STATUS.md`, and only the ADRs that govern the requested behavior.
2. Trace the current path before editing: Next.js route/component → generated API call → `openapi/openapi.yaml` → Fiber route registration → transport → service → sqlc query or River job.
3. Identify the required role, `client_id` and `workspace_id` scope, entity version/hash, approval state, audit event, and side effects. Keep vertical-pack fields outside generic contracts.
4. State the intended observable behavior and the nearest regression tests. Ask for input only when a missing choice materially changes product behavior or authorization.

## Implement coherently

- Change `openapi/openapi.yaml` whenever data crosses the FE/API boundary, then regenerate `packages/api-client/src/generated/schema.ts`. Never hand-author a parallel frontend transport type.
- Keep HTTP parsing and response shaping in transport, business validation and authorization in services, scoped persistence in sqlc/repository code, and asynchronous orchestration in River workers.
- Register new routes, handlers, services, job kinds, and dependencies in the existing composition roots. A compiled but unreachable feature is incomplete.
- Enforce tenancy and RBAC on the server. UI visibility is an additional UX affordance, not the security boundary.
- Make mutations transactional when the domain state, audit event, job insertion, approval invalidation, or idempotency record must succeed together.
- Pass immutable entity versions and hashes to jobs. Re-check approval and current state immediately before consequential side effects.
- Update `docs/IMPLEMENTATION_STATUS.md` after a completed milestone. Add or revise an ADR only for a durable architectural decision or a provider-contract difference.

## Verify

- Run the narrow Go and/or Vitest tests while iterating.
- Run `bun run openapi:check` for contract work, then the affected lint, typecheck, test, and build commands.
- Add a PostgreSQL 18 integration test for transaction, tenancy, migration, locking, or River behavior that unit tests cannot prove.
- Browser-test user-visible flows through `/api/studio`, including failure and permission states; do not call paid providers.
- Before completion, run the applicable milestone gate, normally `make verify`, and report any check that could not run.

For a schema change, also use `$studio-database-change`. For UX-heavy, provider, video, approval, or release work, apply the corresponding narrower Studio skill alongside this one.

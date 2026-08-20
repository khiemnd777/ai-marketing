---
name: studio-feedback-triage
description: Diagnose AI Product Marketing Studio feedback, screenshots, broken flows, regressions, confusing UX, or inconsistent local behavior. Use to reproduce and locate the owning layer before fixing; do not use for a clearly specified new feature.
---

# Studio Feedback Triage

Turn feedback into a reproducible observation, an evidence-backed cause, and—when the user requested a change—a regression-tested fix.

## Reproduce before changing

1. Separate what the user observed from the expected behavior and from any proposed solution.
2. Capture the smallest useful context: route, role, client/workspace, entity/version/status, demo or live mode, viewport, action, and visible error. Never request or expose secrets, cookies, provider tokens, presigned URLs, or raw authorization headers.
3. Start from the user's local workflow (make start or the already-running stack). Inspect current container health and logs only as needed; redact provider payloads and credentials.
4. Reproduce through the same surface the user used. For UI feedback, inspect both the visible browser state and the network/API outcome. Do not silently replace UI reproduction with a direct API call.

## Locate the owner

Trace only as far as evidence requires:

- UI state, permission display, query invalidation, or responsive behavior → apps/web.
- Request/response mismatch → openapi/openapi.yaml, generated client, proxy, and Go transport.
- Validation, authorization, approval, or orchestration → Go service and app registration.
- Missing, duplicated, stale, or cross-tenant data → sqlc query, transaction, migration, or River job.
- Provider-only symptoms → typed adapter, sanitized trace, stored normalized state, and demo fixture.
- Rendering/media symptoms → object metadata, immutable manifest, renderer logs, ffprobe/QC result.

Always check common Studio regression axes: server-side RBAC; dual client/workspace scope; optimistic versions; approval/hash invalidation; stale TanStack Query keys; job idempotency/retries; demo-versus-live adapter selection; loading/error/empty states; keyboard, mobile, and reduced-motion behavior.

## Fix and prove

- For diagnosis-only requests, explain the cause and stop before implementing.
- For requested fixes, add the narrowest regression test that fails for the observed behavior, then change the owning layer without broad refactors.
- Preserve persisted user data and unrelated worktree changes. Do not reset databases, delete volumes, or replay paid actions to make reproduction easier.
- Re-run the exact failing path, the nearest automated tests, and any cross-boundary contract or browser checks.
- Report the original symptom, cause, changed behavior, verification evidence, and anything that still requires the user's environment.

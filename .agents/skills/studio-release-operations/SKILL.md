---
name: studio-release-operations
description: Verify and prepare AI Product Marketing Studio changes for local use, CI, milestone completion, deployment, or live-provider readiness. Use for Docker lifecycle, test gates, GitHub Actions, runbooks, release evidence, and operational handoff; not for ordinary feature implementation alone.
---

# Studio Release Operations

Match evidence to the release claim and keep local, CI, staging, and live-provider verification distinct.

## Select the gate

- Local lifecycle: use make start, make stop, and make restart; restart must rebuild application images, rerun Atlas and River migrations through Compose dependencies, and wait for healthy services.
- Code gate: run the nearest checks while iterating, then make verify for a completed milestone. Supply non-secret configuration-only values when production Compose interpolation requires them.
- Database gate: validate Atlas history and replay it on clean PostgreSQL 18, then run affected Testcontainers/integration workflows.
- UI gate: production build plus Playwright or browser QA through the real /api/studio proxy, including permission and failure states.
- Renderer gate: production image plus deterministic private-storage smoke and full media inspection.
- Live-provider gate: follow docs/runbooks/live-provider-certification.md; it is separate from code acceptance and requires explicit owner authorization and configured staging credentials.

## Required evidence

At minimum, consider:

- bun run openapi:check, lint, typecheck, unit tests, and production builds.
- Go format, vet, tests, and race suite; loopback tests may need execution outside a restrictive sandbox.
- Compose config, container health, API/renderer readiness, and actual host port bindings.
- Migration status, worker health, and preservation of named volumes across stop/start.
- No paid network calls in CI or demo mode, no secrets in logs/artifacts/diffs, and no .env files staged except examples.

Do not delete volumes, reset databases, rotate credentials, enable live mode, publish content, or activate spend as part of verification without explicit authorization. Diagnose failures rather than weakening or skipping gates.

## Handoff and repository hygiene

- Update docs/IMPLEMENTATION_STATUS.md with commands and observable evidence; update a runbook/ADR only when its maintained decision or procedure changed.
- Inspect git status and staged diffs, preserve unrelated changes, and keep generated artifacts synchronized.
- Commit or push only when authorized. Report the commit, branch, remaining external prerequisites, and whether local services were left running.
- Never equate a successful build with deployment readiness or a demo-provider pass with live-provider certification.

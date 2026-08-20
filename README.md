# AI Product Marketing Studio

AI Product Marketing Studio is a managed internal operations platform that turns approved product facts into campaign concepts, social copy, two-character conversational videos, Meta publishing jobs, paused Meta Ads campaigns, and performance recommendations.

Phase 1 ships one extensible vertical pack (`travel-luggage`), two end-to-end video formats (`INTERVIEW_REVIEW` and `PROBLEM_SOLUTION`), Vietnamese and English outputs, and a no-cost deterministic demo flow. Public registration, customer billing, entitlements, and autonomous spend changes are intentionally excluded.

## Architecture at a glance

- `apps/web`: Next.js 16 internal operations console.
- `services/api`: Go modular monolith, HTTP API, and River worker commands.
- `services/renderer`: isolated Node.js 24/Remotion renderer.
- `packages/contracts`: shared JSON Schemas and domain constants.
- `packages/api-client`: OpenAPI-generated TypeScript client.
- `packages/video-templates`: render compositions and validation.
- `database`: PostgreSQL schema, Atlas migrations, sqlc queries, and demo seeds.
- `verticals/travel-luggage`: data schema, claim rules, prompt rules, and validation rules.

See [the architecture overview](docs/architecture/overview.md), [implementation status](docs/IMPLEMENTATION_STATUS.md), and [ADRs](docs/adr/).

## Local development

Prerequisites on the host are Bun 1.3+, Docker, and Make. Runtime containers pin Node.js 24, Go 1.26, PostgreSQL 18, and the required media tools.

1. Copy `.env.example` to `.env.local` and provide random values for `SESSION_SECRET`, `ENCRYPTION_KEY`, and `RENDERER_SHARED_SECRET`. Provider credentials are optional in demo mode.
2. Start infrastructure with `docker compose -f infra/compose/dev.yml up -d postgres minio minio-init`.
3. Install JavaScript dependencies with `bun install`.
4. Apply migrations with `make migrate`.
5. Start the API/worker and web application with `make dev`.

The production build does not require live provider credentials. Provider routes fail with normalized configuration problems when their adapter is enabled but unconfigured.

## Operations and recovery

- [Coolify deployment](docs/deployment/coolify.md)
- [Failure diagnosis and recovery](docs/runbooks/failure-recovery.md)
- [PostgreSQL backup and restore verification](docs/runbooks/backup-restore.md)
- [Provider and application secret rotation](docs/runbooks/provider-secret-rotation.md)
- [Data retention](docs/data-retention.md) and [R2 lifecycle](docs/storage/r2-lifecycle.md)
- [Security review](docs/security-review.md) and [performance review](docs/performance-review.md)

## Safety invariants

- Product Truth is authoritative; locked values cannot be altered by AI or editor paths.
- Every business operation is scoped to a client and workspace.
- Scripts, scenes, final renders, publishing, campaign activation, budget increases, and recommendation actions require the prescribed human approvals.
- Paid generations are idempotent and are never retried automatically unless the provider documents safe semantics.
- Meta campaigns are created paused and cannot spend until a separately audited approval action.
- Automated tests force demo providers and cannot make paid network calls.

## Documentation sources

Provider adapters are implemented from official provider documentation and keep model IDs, API versions, pricing, and endpoints configurable. Deviations or ambiguities are recorded in ADRs and `docs/IMPLEMENTATION_STATUS.md`.

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

1. Copy `.env.example` to `.env.local` and provide random values for `SESSION_SECRET`, `ENCRYPTION_KEY`, and `RENDERER_INTERNAL_AUTH_SECRET`. Provider credentials, endpoints, models, versions, and demo/live mode are configured per client in **Settings → Providers** after sign-in; they are not application environment variables.
2. Run `make start`. The first run builds missing images, starts the core stack, and applies Atlas and River migrations through Compose dependencies.
3. Open `http://localhost:3300`. When no internal user with role `ADMIN` exists, the login screen switches to the one-time Admin bootstrap UI. After the first Admin is created, public bootstrap closes and the screen returns to normal internal login.

After creating a local client, connect its demo profile to the private MinIO bucket with `make configure-local-storage CLIENT_ID=<client-uuid>`. The command is development-only, refuses to replace non-local R2 profiles, and can safely refresh a local MinIO profile. It stores credentials through the same encrypted, audited, per-client service used by the Provider UI. Local profiles deliberately use separate endpoints: Docker services connect to `http://minio:9000`, while presigned browser requests use the loopback host port (by default `http://localhost:9100`).

Use `make stop` to stop and remove the local containers while preserving named volumes. Use `make restart` after code changes; it stops the stack, rebuilds application images, recreates the stack, reapplies idempotent Atlas/River migrations, and reruns the API, worker, renderer, and web services.

Default host ports avoid the commonly occupied `3000`, `8080`, `9001`, and `5432`: web `3300`, API `8180`, renderer `8190`, MinIO `9100`/`9101`, and PostgreSQL `55432`. Override the corresponding `STUDIO_*_PORT` values in `.env.local` when needed. Keep `APP_URL`, `API_URL`, `BROWSER_STORAGE_ORIGINS`, and the client-scoped provider callback URLs in the database aligned with any overrides. The browser storage allowlist is applied to CSP `connect-src`, `img-src`, and `media-src`; it contains origins only and never storage credentials or presigned URLs.

The production build does not require live provider credentials. Provider routes resolve configuration by `client_id` and fail with normalized configuration problems when that client's adapter is enabled but incomplete.

## Repository skills

Codex automatically discovers the versioned skills in `.agents/skills`. They can also be invoked explicitly with `$skill-name`:

| Skill | Use it for |
| --- | --- |
| `studio-feature-slice` | User-visible changes spanning OpenAPI, Go, generated clients, and Next.js. |
| `studio-feedback-triage` | Reproducing product feedback and locating the owning layer before a fix. |
| `studio-database-change` | PostgreSQL 18, Atlas, sqlc, transactions, scoping, and River persistence. |
| `studio-frontend-ux` | Accessible, responsive, permission-aware UI and browser QA. |
| `studio-provider-integration` | OpenAI, Seedance, transcription, storage, or Meta provider contracts and adapters. |
| `studio-video-pipeline` | Media upload, generation, QC, composition, and deterministic rendering. |
| `studio-truth-approval` | Product Truth, approval invalidation, publishing, paid actions, and recommendations. |
| `studio-release-operations` | Local Docker lifecycle, CI gates, release evidence, and live-readiness handoff. |

Skills are intentionally focused and may be combined for cross-cutting work. See the [OpenAI skill documentation](https://developers.openai.com/codex/skills) for discovery and invocation behavior.

## Operations and recovery

- [Coolify deployment](docs/deployment/coolify.md)
- [Failure diagnosis and recovery](docs/runbooks/failure-recovery.md)
- [PostgreSQL backup and restore verification](docs/runbooks/backup-restore.md)
- [Provider and application secret rotation](docs/runbooks/provider-secret-rotation.md)
- [Live provider readiness certification](docs/runbooks/live-provider-certification.md)
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

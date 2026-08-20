# AI Product Marketing Studio — Agent Rules

These rules apply to the entire repository.

## Product scope

- Phase 1 is an internal managed platform. Do not add public registration, billing, subscriptions, entitlements, customer self-service, or a public API.
- The only complete vertical pack is `travel-luggage`; keep vertical fields out of generic campaign, AI, video, publishing, ads, and analytics contracts.
- Complete video formats are `INTERVIEW_REVIEW` and `PROBLEM_SOLUTION`, in Vietnamese or English, 30 or 45 seconds, primarily 1080x1920 (9:16).
- Publishing is limited to Facebook Pages and Instagram Business. Paid distribution is limited to Meta Marketing and Insights APIs.

## Architecture

- Use Bun workspaces, Next.js 16 App Router, React 19, strict TypeScript, Tailwind CSS 4, shadcn-style components, TanStack Query, React Hook Form, and Zod.
- Use Go 1.26, Fiber v3, pgx/v5, sqlc, Atlas versioned migrations, PostgreSQL 18, and River in a modular monolith with separate API and worker commands.
- OpenAPI 3.1 is the cross-language contract source. Do not hand-copy backend DTOs into the frontend.
- The renderer is an isolated Node.js 24 service using exactly aligned Remotion 4 packages, Sharp, FFmpeg, and ffprobe.
- Provider adapters live behind typed interfaces. Model IDs, API versions, base URLs, pricing, and feature flags are configuration—not constants in business logic.
- PostgreSQL is the system of record and River is the job system. Do not introduce Redis, Kafka, Temporal, or other out-of-scope infrastructure.

## Security and data integrity

- Never expose or log secrets, session values, provider tokens, presigned URLs, raw authorization headers, or unredacted provider payloads.
- All business records must be scoped by both `client_id` and `workspace_id`; enforce scope in repositories and services, not only in UI filters.
- Use Argon2id, secure HttpOnly cookies, PostgreSQL-backed revocable sessions, CSRF protection, server-side role checks, and append-only audit events.
- Product Truth locked values are immutable through generation paths. Deterministic validation runs after AI generation and every human edit.
- Approval records bind an approver, entity version, and content hash. Any meaningful mutation invalidates the affected approval.
- Provider webhooks must be signature-checked when supported, deduplicated, and processed idempotently.
- Paid mutations require an internal idempotency key and explicit human action. Tests and demo mode must never call paid providers.
- Meta campaigns are created paused. AI cannot activate campaigns, increase budgets, publish content, or apply recommendations.
- Do not use unauthorized real-person likeness or voice cloning, fabricate testimonials, or render exact product facts into generative pixels.

## Engineering workflow

- Implement vertical slices and keep `docs/IMPLEMENTATION_STATUS.md` current after every milestone.
- Add or update an ADR for durable architectural decisions or official-provider contract differences.
- Database changes must update schema, migrations, sqlc queries, repository code, OpenAPI contracts, and tests together.
- Use table-driven Go tests, Vitest/Testing Library, Playwright, provider fixtures, and deterministic renderer tests. Paid network calls are forbidden in automated tests.
- Run the nearest checks while iterating and the full milestone gate before declaring a milestone complete.
- Do not leave production TODOs, dead controls, fake success responses, or placeholder provider implementations. Demo adapters are explicit, configurable implementations with persisted deterministic fixtures.
- Preserve unrelated local changes. Never commit `.env*` files other than examples.

## Standard commands

- `bun install --frozen-lockfile`
- `bun run lint`
- `bun run typecheck`
- `bun run test`
- `bun run build`
- `go test ./...`
- `go test -race ./...`
- `docker compose -f infra/compose/dev.yml config`
- `make verify`

# Architecture Overview

## System context

The studio is an internal, workspace-scoped modular monolith with four production process roles:

1. The Next.js web console renders internal workflows and uses the generated OpenAPI client.
2. The Go API process serves the Fiber HTTP boundary against one PostgreSQL database.
3. The Go worker process consumes River jobs and contains the media toolchain without expanding the API image.
4. The renderer accepts signed, versioned render manifests and produces inspected media outputs.

External systems are OpenAI Responses, BytePlus ModelArk/Seedance, Cloudflare R2, Meta Graph/Marketing/Insights, and a replaceable transcription service. All provider traffic originates server-side.

## Trust boundaries

- Browsers receive only application sessions, CSRF tokens, normalized data, and short-lived R2 operation URLs.
- Provider and Meta credentials are encrypted at rest and decrypted only inside the API/worker process.
- Worker jobs store normalized state and sanitized provider envelopes. Temporary provider URLs are copied into private R2 immediately.
- Renderer calls use a service secret and validate every referenced object hash before composition.

## Business boundaries

The Go application is grouped into auth, internal users, clients, workspaces, brands, products, Product Truth, vertical packs, media, campaigns, concepts, content, scripts, scenes, characters, AI, Seedance, transcription, quality, rendering, approvals, publishing, Meta Ads, analytics, usage, audit, notifications, and operations.

Modules expose concrete application services and repository interfaces only where substitution, transaction isolation, or provider testing requires them. Cross-module workflows use application orchestration and database transactions rather than shared mutable package state.

## State and asynchronous work

PostgreSQL is authoritative. River queues handle AI content, Seedance submit/status/download, transcription, quality checks, rendering, social publishing, Meta Ads, metrics synchronization, and maintenance. Job arguments identify immutable entity versions; consumers re-check current approval and hash state before side effects.

## Tenancy and authorization

Internal roles are `ADMIN`, `OPERATOR`, and `REVIEWER`. Every business aggregate carries `client_id` and `workspace_id`. Repositories require an explicit scope object, and authorization tests attempt cross-workspace reads and writes. Admin capability does not remove audit requirements.

## Product Truth and approvals

Product versions preserve editable catalog data while Product Truth facts and claims preserve independently approved evidence. Generators receive only effective approved facts and must echo locked values exactly. Deterministic validators compare structured content, dialogue, and rendered overlays against canonical values.

Approvals bind entity type, entity ID, version, hash, approver, timestamp, and notes. A content-changing write increments the version and appends an invalidation event. Side-effecting workflows require the exact approved version and hash.

## Demo and production providers

Demo mode selects deterministic adapters that persist the same normalized request, output, usage, and job records as real providers. It never reports a live provider success. Production adapters validate configuration lazily so installation, tests, and builds need no live credentials.

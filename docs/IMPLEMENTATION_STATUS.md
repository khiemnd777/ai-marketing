# Implementation Status

Last updated: 2026-08-21

## Current state

Milestones 0 through 6 and acceptance hardening gates 1 through 6 are complete for Phase 1 code and CI acceptance. On 2026-08-20, the product owner accepted Phase 1 without a credentialed staging run and chose to perform live-environment verification later. That operational follow-up is documented in the [live provider certification runbook](runbooks/live-provider-certification.md) and remains mandatory before enabling real providers or operator access in a production-like environment.

## Post-acceptance customer-centered UX

- Reorganized the authenticated shell around the hierarchy `Client → Workspace`, with role-aware client, workspace, and system navigation groups plus canonical nested frontend routes.
- Rebuilt the client portfolio with debounced URL-backed search, status filters, pagination, responsive scan-friendly rows, a dirty-state-protected creation drawer, and explicit lifecycle confirmation.
- Expanded client detail into a Customer Hub with overview, profile, workspace views, provider visibility for Admin, factual workspace readiness indicators, and canonical links into brand, Product Truth, media, character, campaign, Meta, and analytics workflows.
- Added Product Media inside Product Truth as a product-scoped view over the existing `media_assets` source of truth. Operators can upload directly with `product_id`, link an unassigned Library asset, or detach an unused asset without deleting it; Media Library gained product filtering and ownership labels.
- Added Brand Logo management as ordered, immutable references to the same Media Library. Brand pages support logo-only upload, review-aware selection, primary/alternate ordering and signed previews; Media Library shows Brand ownership/usage and can filter by Brand.
- Loaded each vertical pack's media requirements and exposed product readiness. Product approval now requires an approved, unexpired, upload-verified asset for every minimum category. Scene and generation paths accept only eligible media for the campaign product, recheck eligibility before paid submission, and the Scene Director uses named pickers instead of raw media UUIDs.
- Preserved legacy scoped routes while adding canonical client/workspace paths; those routing-only changes required no database or OpenAPI change.

## Phase 1 acceptance hardening

| Gate | Status | Validation evidence |
| --- | --- | --- |
| 1 — Release blockers and browser smoke | Complete | The OpenAPI fetch wrapper now preserves `Request` method/body/headers and attaches CSRF correctly; `/api/studio` uses a runtime proxy instead of a build-time rewrite; Atlas checksums validate and all 9 migrations replayed on clean PostgreSQL 18; the analytics recommendation route matches OpenAPI; production Next build and Playwright login → client → CSRF mutation → workspace → recommendation smoke passed. |
| 2 — Auth lifecycle and workspace context | Complete | Admin-managed reset/disable/reactivate revokes sessions and uses optimistic versions; forced password change blocks business routes; users can inspect/revoke their sessions; server-side auth gating prevents shell flash; global client/workspace scope persists in URL and local storage; navigation and primary actions follow Admin/Operator/Reviewer permissions. Production build and Playwright auth lifecycle passed. |
| 3 — Composer and Quality workflow | Complete | Added dedicated `/quality` and `/composer` workbenches. Quality owns transcript, deterministic checks, findings, human checklist and take selection; Composer owns ordering, edits, rendering, final review and campaign-output selection. Legacy `/final` remains compatible. Unit tests, production build and Playwright route smoke passed. |
| 4 — Media Library | Complete | Replaced the form uploader with an Uppy queue that persists selection/session state, resumes confirmed multipart parts, retries object-store writes, and verifies completion server-side. Added signed previews, search/type/status filters, folder/tags/rights/expiry editing, optimistic review/archive actions, role-aware controls, scoped upload validation, retry-safe completion, and multipart cleanup on failed session persistence. Go media/storage tests, OpenAPI generation, strict web checks, production build, and Playwright media route smoke passed. |
| 5 — QA and CI depth | Complete | CI runs the full production-proxy browser suite with API, worker, private MinIO, and the production renderer. Its no-cost Product Truth-to-analytics journey approves and selects four QC'd takes, produces and approves a real final MP4, connects demo Meta, publishes a post, creates an Ads campaign PAUSED by default, syncs metrics, and creates a recommendation. An isolated PostgreSQL 18 Testcontainers workflow, renderer plan golden coverage, and a separate production-container smoke validate HMAC, input/output checksums, 1080×1920 H.264/AAC output, thumbnail persistence, full FFmpeg decode, and idempotent reuse. OpenAI/Meta/Seedance failure matrices cover auth, rate limit, outage, moderation, timeout, protocol, and malformed output. |
| 6 — Accessibility, responsive UX, and live certification | Complete; owner verification deferred | Added automated axe WCAG A/AA checks, corrected contrast tokens, global keyboard focus and reduced-motion behavior, skip navigation, and a responsive 44px-target drawer with focus containment/restoration, Escape, backdrop, and overflow checks. The read-only no-spend certification harness validates TLS, API/database/web/renderer readiness, Admin session cleanup, the selected client's live mode, tenant identity, and all provider configurations; automated mock-server tests prove both its success path and per-client demo-mode fail-closed cleanup without credential leakage. A manual GitHub Actions gate targets a `staging` Environment and retains a sanitized report once that Environment is protected and configured. Read-only OpenAI model-access probes returned HTTP 200 for the configured generation and transcription models without creating a generation. The product owner explicitly deferred the credentialed staging and provider-workflow run for later manual verification; the retained checklist remains a prerequisite to live enablement, not Phase 1 code acceptance. |

## Milestones

| Milestone | Status | Validation evidence |
| --- | --- | --- |
| 0 — Foundation | Complete | Go race suite, JS typecheck/lint/tests/build, PostgreSQL 18 migrations, River migrations, Compose config, and production image builds passed. |
| 1 — Product data | Complete | PostgreSQL 18 migration, vertical schema validation, tenant-isolated CRUD, immutable brand/product/fact/claim versions, approval and lock enforcement, direct/multipart object uploads, River media metadata extraction, thumbnail generation, audit logging, HTTP isolation regression, race suite, production builds, and API/web/worker image builds passed. |
| 2 — AI planning | Complete | Versioned campaign briefs, strict OpenAI Responses schemas, deterministic demo provider, concept/content/script/scene generation, two-character selection, approval invalidation, cost estimates, provider traces, generated OpenAPI client, full workflow UI, and fresh-PostgreSQL integration test passed. |
| 3 — Video generation | Complete | Typed BytePlus ModelArk adapter, exact paid-submit idempotency, callback and polling reconciliation, private output persistence, transcription, automated QC, human checklist, edit metadata, preferred-take selection, generated API client, scene review UI, and PostgreSQL integration passed. |
| 4 — Final rendering | Complete | Versioned scene composer, signed/hash-validated renderer manifests, private input fetching, Remotion composition, ffmpeg/ffprobe validation, MP4/thumbnail/SRT/VTT outputs, object-store idempotency, final review/selection UI, PostgreSQL integration, production container startup, and a real 30-second render passed. |
| 5 — Meta distribution | Complete | Encrypted OAuth, Page/Instagram and Ads discovery, approval-bound publishing, PAUSED-by-default campaign creation, exact budget confirmations, spend guardrails, audited actions, insights sync, generated API contract, operator UI, fresh PostgreSQL migration, and end-to-end integration passed. |
| 6 — Analytics and hardening | Complete | Normalized cost ledger, scoped analytics, human-reviewed recommendations, admin operations console, maintenance jobs, OpenTelemetry/Prometheus/Grafana observability, alert rules, retention and recovery runbooks, backup/restore tooling, fresh-schema replay, full race/build suites, and four production image builds passed. |

## Verified assumptions

- Official OpenAI documentation lists `gpt-5.6-luna` as supporting the Responses API and Structured Outputs. Model and reasoning effort remain runtime configuration.
- Cloudflare R2 presigned URLs operate on the S3 endpoint and support GET, HEAD, PUT, and DELETE. Large resumable media uses the S3 multipart operations rather than browser form POST.
- Atlas versioned migrations use ordered SQL files plus `atlas.sum`; CI will validate migration integrity against PostgreSQL 18.

## Environment observations

- Host Bun: 1.3.1.
- Host Node: 23.11.0; production and CI will use Node 24 containers.
- Host Go: 1.25.0; production and CI pin Go 1.26.7. The host's auto-fetched 1.26.0 archive was incomplete, while the current 1.26.7 toolchain successfully ran sqlc.
- Atlas, sqlc, FFmpeg, and ffprobe are not installed on the host; repository tool containers will provide them.
- Live provider calls remain disabled in tests and demo mode.
- Provider endpoints, models, versions, pricing, credentials, and demo/live mode are database-backed per client. Runtime API and River paths resolve them by `client_id`; no tenant shares a provider profile. AES-256-GCM keeps credential values server-only, while the root encryption key and renderer service-auth secret remain infrastructure secrets.
- Remotion's official license page classifies automated video applications under its Company/Automators licensing. Production release therefore requires an active license when the organization is outside the free-license terms; all Remotion packages are pinned to 4.0.513.

## Validation log

### Managed-service pricing catalogue — 2026-08-22

- Added an internal `Bảng giá dịch vụ` route with three managed-service scopes: Khởi động at 5.29M VND/month, Tăng trưởng at 12.9M VND/month, and Mở rộng at 24.9M VND/month; each onboarding fee now equals its package's monthly fee. Promotional presentation shows Khởi động increasing from 2 to 4 campaign kits and 4 to 8 approved 30-second videos, while Tăng trưởng increases from 3 to 5 campaign kits and 10 to 15 approved videos. Mở rộng remains at 6 campaign kits and 20 videos. The customer-facing copy calls the paid channel `Quảng cáo Facebook & Instagram` and shows only `Không bao gồm` or `Hỗ trợ thiết lập`; PAUSED-by-default remains an internal safety behavior.
- Kept the catalogue commercial-only. It does not add customer registration, billing, subscriptions, entitlements, automatic renewals, or automatic overage/spend. Onboarding, add-ons, VAT, Meta media spend, third-party production costs, and human confirmation requirements are explicit.
- Added package invariant and rendered-page coverage, including ascending price/capacity, one recommended middle tier, supported language/format boundaries, accessible comparison-table semantics, and the absence of self-service purchase controls. No database, OpenAPI, provider, or ADR change was required.
- Passed all 50 Web tests, strict Web typecheck, ESLint, the Next.js production build, and `git diff --check`. Browser QA passed at 1440×1000 and 390×844: three-to-one-column responsive layout, locally scrollable comparison table without page overflow, active navigation, mobile drawer behavior, updated starter pricing/video scope, and zero console warnings/errors.

### Campaign workflow ownership and distribution branches — 2026-08-21

- Renamed the Quality step and workbench to `Duyệt take`, and removed final-render queries and review controls from that route. Final MP4 review and campaign-output selection now have one owner in `Dựng & duyệt final`, eliminating the previous Quality → Composer → Quality loop.
- Kept the existing OpenAPI progress keys and persisted completion rules, while presenting Publishing and optional Meta Ads as sibling channels inside `Bước 8 · Phân phối`. Either channel remains independently navigable and no longer visually implies that paid distribution depends on an organic post.
- Updated component and Playwright journey coverage so final approval remains on Composer and the distribution branch retains separate persisted completion states.
- Passed focused and complete Web tests (44), strict Web typecheck, ESLint, the Next.js production build, and the isolated Web container build. Recreated only the local Web container; API, worker, renderer, PostgreSQL, and persisted Studio data were left intact.

### Campaign progress navigation — 2026-08-21

- Initially replaced the campaign workflow pill links with one persisted progress stepper spanning Brief through the two distribution outcomes. The active route uses `aria-current="step"`, Meta Ads remains explicitly optional, and route position does not imply completion; the ownership correction above subsequently grouped the two outcomes as sibling channels.
- Added a client/workspace/campaign-scoped progress read endpoint. Checks now fail closed and require persisted state: two valid campaign characters; a locked approved concept; all 14 content variants approved; current script and every scene approved; one current approved selected take per scene; a current approved selected final render; an actually published social post; and, for optional Ads, a provider-created paused/active/archived campaign. Approval-backed steps require a matching non-invalidated approval and loading/error states render no checks.
- Synced the generated TypeScript client from OpenAPI and invalidated the progress query after brief, character, concept, content, script, scene, and AI-generation mutations; asynchronous video/render/publishing/Ads steps poll the single progress summary only while their current step is incomplete.
- Passed repository lint, strict typecheck, all 44 Web tests, all Go API tests, OpenAPI generation check, production builds, and `git diff --check`. The read-only PostgreSQL integration test passed against local data and verified cross-workspace lookup failure. `make restart` rebuilt the local images and left API, worker, renderer, Web, PostgreSQL, and MinIO healthy; the reported campaign currently has no characters, concept, content, script, or scenes, so every step correctly remains unchecked.

### Web container vertical-pack packaging — 2026-08-21

- Fixed the development Web image build after the guided-options work introduced a compile-time import from `verticals/travel-luggage/asset-requirements.json`. Host builds passed because the repository tree was present, but `infra/docker/web.Dockerfile` copied only the Web app, packages and OpenAPI into its isolated build stage.
- The Web Docker build stage now copies `verticals/` before running Next.js, keeping the vertical pack as the single source for media-category options. Added a packaging regression test that requires this copy to precede the containerized Next build.
- The exact failing `docker compose ... build web` command and the complete `make restart` workflow passed. The local API, renderer and PostgreSQL health checks are green, the Web application responds on port 3300, and all 41 Web tests plus lint and typecheck passed.

### Guided form options and entity pickers — 2026-08-21

- Replaced user-facing raw IDs with scoped, human-readable selectors in analytics campaign filters, final creative selection, Meta publishing create/edit flows, scene Product Truth references, and script character roles. Existing values remain visible during edits so legacy records are not silently rewritten.
- Replaced closed-vocabulary text fields with shared options for character source/profile attributes, client industry/market, workspace timezone, brand language, campaign market/country/currency, media categories, and travel-luggage attributes. Provider models plus concept/scene camera, environment and product placement use suggested presets while preserving custom configured values.
- Replaced raw travel-luggage JSON editing with a validated structured form and derived media categories from the vertical pack. The same product update endpoint and version remain in use, so Product Truth approval invalidation and locked-value safeguards are unchanged; no database or OpenAPI change was required.
- Added reusable option and multi-choice controls, legacy-value preservation, travel-luggage mapping/validation tests, and updated browser journeys to select the new client industry options. Passed `git diff --check`, OpenAPI generation drift, the full `make verify` gate, 40/40 web tests, all Go tests, workspace lint/typecheck/tests/build, and both development and production Compose validation using non-secret configuration-only values.

### Brand Logo and Media Library lifecycle — 2026-08-21

- Kept Media Library as the single file/lifecycle source and used the existing ordered `brand_versions.logo_asset_ids`; no schema migration or duplicate upload record was introduced. The first eligible ID is the primary renderer logo and later IDs are approved alternates.
- Added Brand-scoped PNG/WebP/JPEG uploads forced to `LOGO`/`BRAND_LOGO`, role-aware review and selection, primary-logo ordering, invalid-reference cleanup, unsaved-change protection, Media Library Brand filters and Brand/logo usage badges.
- Brand persistence now accepts only same-scope, approved, unexpired, verified and checksummed web images that are not product/campaign assets or owned by another Brand. Selected logos cannot be downgraded or soft-deleted, and meaningful mutations invalidate affected final-render approvals.
- Final-render idempotency now includes the Brand version plus primary-logo asset version/checksum. Render start and manifest construction independently fail closed when a configured primary logo becomes ineligible.
- Passed OpenAPI generation drift, strict web typecheck/lint, 26 web tests, the complete Go suite, targeted renderer tests, and the PostgreSQL 18 Testcontainers workflow covering eligible selection, in-use deletion protection and render rejection for an ineligible primary logo.

### Product Media and generation source clarity — 2026-08-21

- Kept Media Library as the single asset repository and implemented Product Media as contextual association through the existing `media_assets.product_id`; no schema migration or duplicate upload record was introduced.
- Added tenant-scoped list filtering, safe attach/detach endpoints with optimistic versions, in-use protection, product readiness contracts, approval preconditions, and Product Truth draft/invalidation behavior when associated media changes.
- Product detail now contains the readiness checklist, product-aware uploader, existing-asset linker, signed previews, media review actions, and a direct link back to Media Library. Media Library shows and filters product ownership, while Scene Director and take edits use human-readable product-media pickers.
- Scene version persistence, generation start, and the paid-submit worker independently reject wrong-product, unapproved, expired, deleted, unverified, or non-visual product references. The PostgreSQL workflow fixture now also covers safe attachment, cross-product rejection, in-use detach protection, and scene approval invalidation after a referenced-media change.
- OpenAPI client generation drift, targeted Go packages, strict web typecheck/lint, all 24 web tests, and the complete PostgreSQL 18 Testcontainers workflow passed. The workflow also exposed and corrected a stale `meta_ad_actions` worker alias (`x.client_id` → scoped `a.client_id`) before reaching a demo-only Meta action; no live or paid provider call was made.

### Edit completeness and password navigation — 2026-08-20

- Audited Studio-owned mutable resources and completed edit flows for clients, workspaces, the signed-in account, Admin-managed internal users, products, Product Truth facts and claims, workspace characters, unpublished social posts, and Meta Ads drafts that have not reached the provider. Provider-synchronized identities, immutable versions/audit records, published posts, and provider-created Ads remain intentionally read-only.
- Added optimistic-version API contracts and generated client types for account, internal-user, claim, character, social-post, and Meta Ads draft updates. All mutations retain server-side role and tenant checks; account and administrative user changes are audited, and demoting the last active Admin is rejected.
- Product, fact, and claim edits return the affected Product Truth and dependent campaign planning state to draft, invalidate active approvals with events, and reject queued `CREATE_PAUSED` Ads actions. Character edits invalidate affected script/scene approvals. Social-post and Ads draft edits invalidate their approval hashes; Ads edits never call Meta and are allowed only before a provider campaign exists.
- The password page now shows a clear Back action for voluntary changes and preserves the mandatory-reset guard by hiding Back when `requiresPasswordChange` is active; logout remains available in both modes.
- Passed OpenAPI generation drift, all workspace typechecks/lint/tests (16 web tests), the production workspace build, and `go test ./...`. PostgreSQL 18 prepared every new joined update statement successfully, including the Product Truth cascade and Character/Ads approval invalidation queries.

### Client-scoped provider configuration — 2026-08-20

- Replaced process-environment provider settings with PostgreSQL profiles keyed by `client_id`, Admin-only Database UI editing, optimistic versions, safe audit metadata, and AES-256-GCM credential storage. API and worker call paths now resolve OpenAI, Seedance, R2, Meta, and renderer settings from the owning client immediately before use.
- Added the ninth Atlas migration, sqlc queries/generated models, OpenAPI routes and client generation, per-client demo/live controls, secret-presence-only responses, explicit secret clearing, and a tenant-scoped live-readiness gate. The isolated renderer receives the selected client's R2 configuration only in its HMAC-signed internal request; provider environment fallbacks were removed.
- Passed Atlas validation, OpenAPI drift check, the full `make verify` gate, 14/14 web tests, all Go tests, renderer tests, certification mock tests, and a PostgreSQL 18 Testcontainers regression proving encrypted storage plus isolation between two clients. Rebuilt the local stack, applied the migration, confirmed every service healthy, and visually verified the no-client state and selected-client five-provider form in Microsoft Edge without entering or changing credentials.

### Local client storage bootstrap — 2026-08-21

- Added a development-only `make configure-local-storage CLIENT_ID=<client-uuid>` helper for clients created after the local stack starts. It refuses production and non-local R2 profiles, safely refreshes an existing `local-minio` profile, then persists the private MinIO settings through the existing encrypted, audited, client-scoped provider service instead of restoring an environment fallback.
- Local object storage now has separate internal and browser endpoints. API, worker, and renderer use Docker DNS (`http://minio:9000`) for server-side multipart, `HEAD`, `GET`, and `PUT` operations; only presigning uses the browser-reachable loopback endpoint (`http://localhost:9100`). This fixes the completion failure where a browser PUT succeeded but the API could not verify the object through a loopback-only host port. MinIO remains bound to IPv4/IPv6 loopback, and the web CSP allowlist covers only the browser endpoint. Corrected the Go JSON field names for presigned requests to match the OpenAPI `url`/`method`/`headers`/`expiresAt` contract and made the uploader tolerate a missing legacy header map.
- Hardened the post-upload lifecycle uncovered by the real storage probe: empty S3 user metadata is persisted as `{}` instead of JSON `null`; still images no longer seek past their only frame when rendering thumbnails; and the metadata worker computes SHA-256 while streaming the original object to disk, then persists the digest required by `readyForUse`. A real 1,993,523-byte PNG completed as `VERIFIED`, produced a 31 KiB JPEG thumbnail, recorded 1536×1024 dimensions and a SHA-256 digest, and reached `readyForUse=true`.

### Local lifecycle and first-Admin bootstrap — 2026-08-20

- Added loopback-only, overridable local ports that avoid the occupied host ports `3000`, `8080`, `9001`, and `5432`: web `3300`, API `8180`, renderer `8190`, MinIO `9100`/`9101`, and PostgreSQL `55432`. PostgreSQL 18 now mounts its named volume at `/var/lib/postgresql`, matching the official 18+ image layout.
- Added `make start`, `make stop`, and `make restart`. A real local lifecycle run confirmed that `restart` rebuilds the API, worker, renderer, web, and River migration images; recreates the stack; reapplies Atlas and River migrations; and waits for healthy services. `stop` preserves named volumes, and a subsequent `start` passed with all core services healthy.
- Replaced environment-variable/CLI Admin bootstrap with a one-time UI on `/login`. The public status check counts only users whose role is `ADMIN`; existing Operators/Reviewers do not close bootstrap, while an existing disabled Admin does. Creation is protected by PostgreSQL table locking, creates the Admin/session/CSRF/audit event atomically, and rejects concurrent or later attempts.
- Added OpenAPI-generated client coverage, unit tests for the login/bootstrap switch and form submission, a PostgreSQL 18 Testcontainers race/integrity workflow, and a Playwright setup project that bootstraps through the UI. Desktop and mobile browser QA confirmed the bootstrap copy, labeled inputs, validation feedback/focus, and responsive layout. No real local Admin was created during verification, so the running local stack still presents the bootstrap UI for the owner.
- Passed the full `make verify` gate—workspace lint/typecheck/tests, production web/Go builds, and both Compose configs—using non-secret configuration-only values for the required production variables. OpenAPI generation drift and the auth race suite passed; `make restart`, `make stop`, and `make start` all passed against Docker.

### Phase 1 acceptance hardening — 2026-08-20

- Replayed all Atlas migrations and River migrations on clean PostgreSQL 18, then passed the isolated Testcontainers workflow on the final tree.
- Passed all 5 production-proxy Playwright tests in one serial run: accessibility desktop/mobile, auth lifecycle and session revocation, the complete no-cost Product Truth-to-analytics campaign journey, and client/workspace/media/analytics/composer smoke. The campaign journey exercises four generated/QC'd takes, production final rendering, final approval/selection, Meta demo OAuth and publishing, Ads creation PAUSED by default, metrics sync, and recommendation creation.
- Passed OpenAPI generation drift, every workspace typecheck/lint/test/build, renderer golden tests, `go vet ./...`, and `go test -race ./...` (loopback-dependent provider tests ran outside the macOS sandbox).
- GitHub Actions run [32376552690](https://github.com/khiemnd777/ai-marketing/actions/runs/32376552690) passed all six acceptance jobs on commit `fb95ba8`; its browser job passed 5/5 tests in 2.4 minutes and its container job passed the independent production renderer smoke.
- Full browser acceptance exposed and closed two production-only worker defects hidden by the earlier route smoke: PostgreSQL 18 required an explicit boolean cast when persisting deterministic QC events, and River's one-minute default job timeout was shorter than a real final render. QC now has real-worker PostgreSQL integration coverage; final-render jobs have a 25-minute timeout that outlives the renderer's 20-minute HTTP timeout, and failure-state persistence uses a bounded cleanup context so the UI cannot remain indefinitely in `VALIDATING` or `RENDERING` after cancellation.
- The production-container gate now performs the required short renderer smoke on every commit: it generates a local 30-second fixture, renders and retrieves the output through private MinIO, validates exact output metadata/checksum/thumbnail, fully decodes the persisted MP4, and proves repeat-request reuse without a paid provider.
- Verified read-only OpenAI access to `gpt-5.6-luna` and `gpt-4o-mini-transcribe` through the Models API (HTTP 200 for each); no generation or transcription request was made.
- Ran the live readiness harness against the local demo environment and confirmed the expected fail-closed result for the selected client's database profile. No paid or live provider operation was executed.
- Audited the GitHub repository and found no Actions secrets, variables, environments, or deployments. Credentialed staging certification therefore remains an explicit external release prerequisite rather than inferred evidence.
- Recorded the product owner's decision to defer credentialed staging certification from Phase 1 acceptance and retain it as an owner-run prerequisite before live enablement. The durable checklist, secret-handling rules, no-spend gate, and six provider/workflow checks are maintained in `docs/runbooks/live-provider-certification.md` for future review.

### Milestone 0 — 2026-08-20

- `cd services/api && gofmt -w cmd internal && GOCACHE="$PWD/.gocache" go test ./...` — passed.
- `cd services/api && GOCACHE="$PWD/.gocache" go vet ./... && GOCACHE="$PWD/.gocache" go test -race ./...` — passed.
- `bun run typecheck`, `bun run test`, and `bun run lint` — passed across all workspaces.
- `bun run build` — passed; the web package uses Next's supported webpack builder because this macOS sandbox prohibits Turbopack's internal CSS worker from binding a localhost port.
- `docker compose -f infra/compose/dev.yml config --quiet` — passed.
- `atlas migrate apply` through `arigaio/atlas:1.2.0` against an isolated `postgres:18.4-alpine` container — 1 migration/18 statements passed.
- River's official migrator from the built API image against the same database — versions 1–7 passed.
- `docker build` for `infra/docker/api.Dockerfile`, `web.Dockerfile`, and `renderer.Dockerfile` — passed.
- Built API `/app/api --healthcheck` and renderer `/health/ready` — passed.

Deferred to later milestones: full provider-specific dashboards and alert tuning, backup/restore drills, and deployment smoke testing belong to Milestone 6.

### Milestone 1 — 2026-08-20

- Applied `20260820000200_product_data.sql` against isolated PostgreSQL 18: 37 statements passed and the Atlas schema snapshot was regenerated.
- Exercised the HTTP workflow for client, workspace, immutable brand/product versions, Product Truth approval and locking, and cross-workspace concealment; all assertions passed.
- `cd services/api && GOCACHE="$PWD/.gocache" go vet ./... && GOCACHE="$PWD/.gocache" go test -race ./...` — passed, including deterministic fact/claim validation, JSON Schema validation, object-key isolation, upload validation, and ffprobe decoding.
- `bun run openapi:check`, `bun run typecheck`, `bun run lint`, `bun run test`, and `bun run build` — passed across all workspaces.
- Built production API, web, and dedicated media-worker images. Runtime checks confirmed both `ffmpeg` and `ffprobe` are present in the non-root worker image.
- Upload completion and River job insertion share one PostgreSQL transaction. Fact and claim approval versions and their before/after audit events also share the approval transaction.

### Milestone 2 — 2026-08-20

- Applied `20260820000300_ai_planning.sql` and `20260820000310_campaign_characters.sql` against isolated PostgreSQL 18: 45 statements passed and the canonical schema snapshot was regenerated at the migration head.
- Implemented immutable campaign, concept, content, script, and scene versions with optimistic concurrency, approval hashes, approval events, and downstream invalidation on meaningful changes.
- Implemented a strict `LLMProvider` boundary. The live adapter uses the OpenAI Responses API with strict JSON Schemas; the deterministic demo provider covers both video formats, all 14 content variants, 30/45-second scripts, and exact-duration scene directions without paid calls.
- Implemented transactional generation jobs and provider request/output traces with input/output hashes, model, prompt version, token usage, estimated/actual cost, request IDs, and sanitized errors.
- Added the complete Campaign Builder UI: brief, concept comparison/edit/approve/reject/lock, content variants, script editor, two-character selection, Scene Director, estimates, and reload-safe job polling.
- `STUDIO_INTEGRATION_DATABASE_URL=... go test ./internal/planning -run TestDemoPlanningWorkflowIntegration -count=1 -v` against a fresh `postgres:18.4-alpine` database — passed. It asserted four successful jobs, two concepts, 14 variants, script approval, four scenes totaling exactly 30 seconds, the selected character pair, provider trace parity, and approval persistence.
- `cd services/api && GOCACHE="$PWD/.gocache" go test ./...` — passed.
- API provider/validator tests, web and generated-client type checks, ESLint, API-client tests, and web tests — passed. Live provider calls remained disabled.

### Milestone 3 — 2026-08-20

- Applied `20260820000400_video_generation.sql` against PostgreSQL 18: 17 statements passed. The canonical Atlas schema snapshot was regenerated at the M3 migration head.
- Implemented a typed BytePlus ModelArk v3 adapter for create, retrieve, and queued-task cancellation with text/image/video/audio references, configurable model and format, bounded HTTP I/O, normalized errors, sanitized request/response traces, and temporary-output URL isolation.
- Implemented immutable generation fingerprints across workspace, campaign, scene version/hash, prompt, reference assets/provider assets, model, resolution, ratio, duration, and audio. Identical active/successful work is reused; the chargeable create worker has one attempt and never auto-retries an ambiguous submission.
- Implemented callback-token authentication, callback replay hashes, monotonic state reconciliation, and polling fallback. Successful outputs transition through download and validation, are copied immediately to private R2-compatible storage, checksummed, probed, thumbnailed, and persisted as scoped media assets.
- Implemented OpenAI/demo transcription, normalized transcript comparison, decode/duration/resolution/audio checks, human visual review fields, immutable approval events, rejection notes, selection of an approved take, and non-destructive trim/mute/transition/replacement/product-attachment/subtitle-preview metadata.
- Expanded `TestDemoPlanningWorkflowIntegration` to assert generation idempotency, exactly one paid-submit job, provider submit/status synchronization, download handoff, edits, human approval, and preferred-take selection against PostgreSQL 18.
- `cd services/api && GOCACHE="$PWD/.gocache" go test ./...`, provider contract tests, generated-client/web type checks, lint, tests, and production web build passed. Live paid provider calls remained disabled.

### Milestone 4 — 2026-08-20

- Applied `20260820000500_final_rendering.sql` against PostgreSQL 18 and regenerated both `atlas.sum` and the canonical schema snapshot at the M4 migration head.
- Implemented immutable video-project versions, manifest hashes, render jobs, normalized video outputs, SRT/VTT output records, final approvals, selected campaign output, and downstream approval invalidation when composition settings change.
- Implemented the scene-based final composer for AI takes, replacement/product media, logos, headline/lower-third, structured price/discount/CTA/website/phone/QR/disclaimer overlays, Vietnamese/English burned captions, music gain, dialogue ducking, sound effects, transitions, 1080×1920 H.264/AAC MP4, and thumbnail output.
- The isolated Node.js 24 renderer verifies HMAC requests and every input SHA-256, serves only downloaded local inputs to Remotion, validates output with ffprobe, uploads private outputs with normalized metadata, cleans per-render temp directories, and reuses an existing output only when its render ID matches.
- Expanded the PostgreSQL workflow integration test to cover all selected approved takes, a single River render job under repeated idempotency keys, immutable final approval, final selection, and campaign approval.
- Built and started the production-like renderer image with pinned Chromium, ffmpeg, Noto fonts, Remotion 4.0.513, and a non-root runtime. A signed 30-second Vietnamese smoke manifest rendered through private MinIO to a 1080×1920, 30 fps, 30.059-second H.264/AAC MP4 plus thumbnail; ffprobe checks and a repeated `reused: true` request passed.

### Milestone 5 — 2026-08-20

- Added the direct Meta provider boundary and live Graph API adapter for OAuth, Page/Instagram discovery and publishing, Business/Ad Account/Pixel/Audience discovery, PAUSED campaign creation, status and budget changes, and campaign insights. Server-side `appsecret_proof`, bounded I/O, normalized errors, and a deterministic demo adapter prevent provider details or secrets from leaking into browser contracts.
- Added AES-256-GCM token storage with random nonces and workspace/account-bound associated data. OAuth state is one-time and hashed, reconnect/disconnect wipes old ciphertext, expiry is exposed as safe metadata, and workers refuse expired token or data-access windows.
- Added per-platform social posts with caption, immediate or River-scheduled delivery, idempotency, immutable content hashes, approval rechecks immediately before publishing, retryable/permanent failure classification, provider IDs/URLs, and audit coverage.
- Added Meta Ads account selection, Pixel and Audience linkage, objective, daily/lifetime budget modes, dates, location/age/gender/interest/custom/retargeting audience data, placements, HTTPS destination/UTM data, creative and thumbnail variants, and preview metadata.
- Ads are created on Meta only as `PAUSED`. Creation, activation/resume, and budget increases require human approval and exact amount confirmations. Workspace aggregate caps, campaign caps, maximum percentage increases, optimistic versions, action hashes, and one-attempt ambiguous-submit workers prevent autonomous spend changes; pause/archive remain auditable operator actions.
- Added campaign/action history and Insights synchronization plus the `/settings/meta`, campaign Publishing, and Meta Ads UI. The generated OpenAPI client, web typecheck, and ESLint passed.
- Refreshed `atlas.sum`, applied all 7 migrations/159 statements to a fresh `postgres:18.4-alpine` database, regenerated the canonical schema snapshot, and expanded the integration workflow through Meta OAuth discovery, publishing, PAUSED Ads creation, metrics sync, human activation, and rejection of a 30% budget increase over the configured 20% guardrail.

### Milestone 6 — 2026-08-20

- Applied the analytics and operations migration at the full migration head, refreshed `atlas.sum`, regenerated the canonical PostgreSQL schema, and replayed that snapshot into a second clean PostgreSQL 18 database, including the analytics materialized view and concurrent-refresh index.
- Added immutable usage and cost records with provider/model/reference attribution, generated and accepted units, idempotency, outcome tracking, reuse metadata, exchange-rate snapshots, and USD normalization. Planning, Seedance review, and final rendering now record usage through the shared ledger.
- Added tenant-scoped cost, video, social, Ads, daily, and creative analytics. Deterministic recommendations persist their input snapshot, model, output, rationale, reviewer, decision, and any separately approved action; recommendations never mutate spend or campaign state autonomously.
- Added an admin operations console with API/provider health, River queue depth and age, retry/discard/stuck visibility, controlled retry/cancel, maintenance mode, webhook and provider failure summaries, safe configuration metadata, costs, anomalies, storage, feature flags, and audit history. No raw credentials or provider payloads are returned.
- Added hourly retention and maintenance jobs for sessions, idempotency records, OAuth state, webhook bodies, River history, expiring Meta connections, cost anomalies, and concurrent analytics refresh. Added versioned retention, R2 lifecycle, failure recovery, secret rotation, Coolify deployment, backup/restore, security, and performance documentation.
- Added OpenTelemetry trace propagation across HTTP, River, provider clients, and renderer requests; Prometheus metrics and alerts cover request failures, provider latency/failures, River queue state/retries/age, PostgreSQL pool saturation, costs, storage, render/generation health, webhooks, and process resource telemetry. Grafana, Tempo, Loki, and Prometheus are wired in production Compose.
- The live PostgreSQL integration workflow passed through planning, generation, rendering persistence, Meta publishing/Ads, usage normalization, analytics, recommendation review, operations queries, and maintenance. `go vet`, the complete Go suite, the Go race suite, OpenAPI generation check, JavaScript typecheck/lint/tests, Next.js production build, Atlas validation, Prometheus rule/config validation, and both development and production Compose validation passed.
- Built production API, worker, web, and renderer images. API health, web route delivery, renderer readiness, renderer's real 30-second 1080×1920 output, and non-root media-worker runtime dependencies were verified.

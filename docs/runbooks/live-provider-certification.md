# Live provider readiness certification

Run this gate from the staging network after deploying the exact release candidate and before enabling operator access. It is deliberately read-only and cannot create OpenAI/Seedance generations, publish to Meta, create or activate Ads, upload media, or render a video.

## Deferred owner follow-up

On 2026-08-20, the product owner accepted Phase 1 code and CI validation without a credentialed staging run and chose to perform the live-environment checks later. This deferral is not a Phase 1 implementation blocker, but the checks below remain mandatory before enabling real providers or operator access in a production-like environment.

When the owner asks to review the remaining or deferred checks, inspect the current deployment and GitHub state again rather than relying on the observations recorded on 2026-08-20. Remind the owner to:

- deploy the exact release candidate to an isolated staging environment with `APP_ENV=production`, HTTPS API/web origins, a private bucket, and all migrations applied;
- configure restricted OpenAI, Seedance, R2, Meta, and renderer settings in the selected staging client's database-backed Provider UI, then explicitly switch only that client to live mode;
- create a protected GitHub Environment named `staging`, with required reviewers and release-branch restrictions;
- set `STAGING_CERT_API_URL`, `STAGING_CERT_WEB_URL`, and `STAGING_CERT_CLIENT_ID`, optionally set `STAGING_CERT_RENDERER_URL`, and add the dedicated `STAGING_CERT_ADMIN_EMAIL` and `STAGING_CERT_ADMIN_PASSWORD` Environment secrets;
- prepare isolated test assets/accounts, provider spend caps, callback allowlists, a rollback owner, and an explicitly authorized spend window before any chargeable provider check;
- run the no-spend readiness gate first, then perform the six credential/workflow checks below and retain the sanitized report, release digest, operator, UTC time, and non-secret provider account/project identifiers.

Never ask the owner to paste provider keys or passwords into chat, source files, shell history, logs, or CI artifacts. Use the staging platform's secret manager and GitHub Environment secrets.

## Preconditions

- Use an isolated staging database and private R2 bucket. Apply all Atlas and River migrations first.
- Set `APP_ENV=production` and HTTPS `APP_URL`. In **Settings → Providers**, select the staging client, enter its restricted credentials/settings, verify all five providers are complete, and explicitly switch that client's profile to live mode.
- Use a dedicated active Admin account with no forced password change. Supply its values only through the process environment; do not place them in shell history, repository files, logs, or CI artifacts.
- Configure provider spend caps, Meta test assets/ad account, R2 lifecycle/versioning, callback URLs, and provider console allowlists before this gate.

## Automated no-spend gate

```sh
STUDIO_CERT_API_URL=https://api.staging.example.com \
STUDIO_CERT_WEB_URL=https://staging.example.com \
STUDIO_CERT_RENDERER_URL=http://renderer:8090 \
STUDIO_CERT_CLIENT_ID='<staging client UUID>' \
STUDIO_CERT_ADMIN_EMAIL='<admin email>' \
STUDIO_CERT_ADMIN_PASSWORD='<admin password>' \
bun run certify:live
```

The command requires API/database readiness, the web login page, optional direct renderer readiness, a valid Admin session, that exact client's live mode, and safe configuration status for `OPENAI`, `SEEDANCE`, `R2`, `META`, and `RENDERER`. It verifies the API response carries the requested `clientId`. Output contains hostnames, the client UUID, and provider names only. The temporary session is revoked in `finally`, including when a later assertion fails.

`STUDIO_CERT_ALLOW_INSECURE_LOCAL=true` permits HTTP only for `localhost`/loopback targets. It must not be used for a remotely accessible staging environment. If the renderer is internal-only, run the command inside the deployment network or omit `STUDIO_CERT_RENDERER_URL`; the API still has to report renderer configuration as complete.

### GitHub Actions gate

The manual **Staging live readiness certification** workflow runs the same read-only gate and stores its sanitized JSON report for 30 days. Before the first run, create a protected GitHub Environment named `staging`, restrict it to the release branch, add required reviewers, and configure:

- Environment variables `STAGING_CERT_API_URL` and `STAGING_CERT_WEB_URL` as HTTPS origins, plus `STAGING_CERT_CLIENT_ID` for the isolated staging client being certified.
- Optional Environment variable `STAGING_CERT_RENDERER_URL` only when the GitHub runner can reach the renderer. Omit it for an internal-only renderer; API configuration is still checked.
- Environment secrets `STAGING_CERT_ADMIN_EMAIL` and `STAGING_CERT_ADMIN_PASSWORD` for a dedicated active Admin account.

The workflow does not receive OpenAI, Seedance, R2, Meta, database, or renderer secrets. Those remain only in the staging deployment. It checks out without persisted Git credentials, serializes certification runs, fails before network access when required Environment values are absent, and emits only the already-sanitized report. For a staging network that rejects GitHub-hosted runners, execute the CLI gate from an authorized host inside that network instead.

## Credential and workflow certification

The automated gate proves configuration is loaded, not that every external credential still has permission. In the provider consoles or an explicitly authorized staging window, record the operator, UTC time, release digest, provider account/project identifiers (never secrets), and these results:

1. OpenAI: read-only project/model access succeeds; one minimal generation is run only with an approved spend allowance and its trace/cost record is reconciled.
2. Seedance: model access and callback/polling are verified; create at most one approved short staging task, then confirm private R2 copy, provider URL removal, QC, and cost record.
3. R2: upload, HEAD, signed read, multipart resume, and cleanup pass in the staging bucket with no public object exposure.
4. Meta: OAuth scopes, Page/Instagram/ad-account/pixel discovery, test publishing, PAUSED Ads creation, exact activation/budget confirmation, insight sync, and disconnect/token deletion pass on test assets only.
5. Renderer: a signed manifest produces the expected 1080×1920 H.264/AAC output, captions/overlays remain in safe zones, and an identical request is reused.
6. Operations: request/trace/job IDs correlate; sanitized provider failures contain no token, raw payload, signed URL, or personal data; alerts and rollback owners acknowledge the release.

Do not certify with production customer data or a spend-capable production ad account. A failed credential/workflow check blocks live enablement but does not justify bypassing approval, idempotency, or PAUSED-by-default guardrails.

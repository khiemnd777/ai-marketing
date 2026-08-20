# Coolify production deployment

Deploy `infra/compose/prod.yml` as a Docker Compose resource behind the included Caddy service. PostgreSQL 18 and R2 are managed external dependencies in production; do not place production database or object data in ephemeral application volumes.

1. Build immutable API, worker, renderer, and web image tags in CI and set `API_IMAGE`, `WORKER_IMAGE`, `RENDERER_IMAGE`, and `WEB_IMAGE` to digests or release tags.
2. Configure the required infrastructure variables from `.env.example` in Coolify secrets. Use a TLS-verified `DATABASE_URL`, random 32-byte decoded key material for session/encryption settings, and a 32+ character renderer internal-auth secret. After deployment, enter restricted provider credentials separately for each client through **Settings → Providers**; do not add provider credentials to service environment variables. No secret uses a `NEXT_PUBLIC_` prefix.
3. Set `APP_DOMAIN`, `APP_URL=https://<domain>`, Meta redirect/callback URLs, and Caddy DNS/ports. Restrict River UI and Grafana with SSO/VPN or network policy in addition to their local authentication.
4. Deploy. The `migrate` one-shot Atlas service runs before the River migrator; API and worker start only after both complete. Never run two incompatible application versions across a breaking migration.
5. Verify `/v1/health/live`, `/v1/health/ready`, login/CSRF, provider status, renderer readiness, River queues, Prometheus targets, Tempo traces, a signed media read, and a demo-mode smoke workflow before enabling live providers. Then run the [live provider readiness certification](../runbooks/live-provider-certification.md) against the exact staging release candidate.
6. Configure daily PostgreSQL backup, monthly restore verification, R2 versioning/lifecycle, CPU/memory/disk/temp-storage alerts at the Coolify host level, and log/trace retention.

Rollback application images only when the database migration is backward-compatible. Otherwise enable maintenance, restore the matched database/R2 recovery set, and follow the backup runbook. Scale workers by queue pressure; preserve one maintenance worker and limit Seedance/Meta concurrency to provider quotas.

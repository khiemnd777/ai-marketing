# Security review

Review date: 2026-08-20. Scope: web, Go API/worker, renderer, PostgreSQL, R2 boundary, OpenAI/Seedance/Meta adapters, and deployment configuration.

## Implemented controls

- Internal Argon2id authentication, secure HttpOnly/SameSite cookies, CSRF, origin enforcement, login rate limiting, CSP/security headers, request IDs, role checks, client/workspace scope checks, and normalized problem responses.
- AES-256-GCM Meta tokens and per-client provider credentials with random nonces and scope-bound associated data; one-time hashed OAuth state; secret-presence-only provider responses; no browser-to-provider calls and no cross-client provider fallback.
- Product Truth validation after generation/edit, no raw HTML model rendering, bounded bodies/strings, private signed R2 requests, MIME/extension/size/probe checks, checksums, scoped object keys, and an explicit malware-scanning integration state before approval.
- HMAC-authenticated renderer manifests, SHA-256 verification for every input, isolated temporary directories, non-root containers, bounded network/body I/O, and normalized error logging.
- Idempotent webhooks and chargeable work, one-attempt ambiguous provider mutations, PAUSED Meta campaign creation, exact budget confirmation, workspace/campaign caps, human approvals, audit events, and recommendation non-execution.
- Encrypted backups/runbook, retention policy, maintenance mode, secret rotation procedure, Prometheus alerts, trace correlation, and admin-only operations controls.

## Release checks

Run dependency/vulnerability scanning in CI, scan built images, search the repository and image layers for credentials, verify `.env.local` is ignored, test cross-workspace concealment and expired/permission-denied providers, and confirm production uses TLS-verified PostgreSQL/R2 endpoints. Restrict database roles, Coolify, River UI, Grafana, backup storage, and provider consoles with least privilege and MFA.

## Accepted operational boundaries

This is a managed internal platform, not a public multi-tenant SaaS. Malware scanning is an integration state and must be connected to the organization's scanner before real third-party uploads are accepted. Host CPU/memory/disk/temp metrics and network policy are deployment-platform controls. Real-person assets require explicit consent records; synthetic content cannot be represented as a testimonial or used for unauthorized face/voice cloning.

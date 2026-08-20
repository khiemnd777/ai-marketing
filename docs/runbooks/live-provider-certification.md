# Live provider readiness certification

Run this gate from the staging network after deploying the exact release candidate and before enabling operator access. It is deliberately read-only and cannot create OpenAI/Seedance generations, publish to Meta, create or activate Ads, upload media, or render a video.

## Preconditions

- Use an isolated staging database and private R2 bucket. Apply all Atlas and River migrations first.
- Set `APP_ENV=production`, `DEMO_MODE=false`, HTTPS `APP_URL`, and restricted staging credentials for OpenAI, Seedance, R2, Meta, and the renderer.
- Use a dedicated active Admin account with no forced password change. Supply its values only through the process environment; do not place them in shell history, repository files, logs, or CI artifacts.
- Configure provider spend caps, Meta test assets/ad account, R2 lifecycle/versioning, callback URLs, and provider console allowlists before this gate.

## Automated no-spend gate

```sh
STUDIO_CERT_API_URL=https://api.staging.example.com \
STUDIO_CERT_WEB_URL=https://staging.example.com \
STUDIO_CERT_RENDERER_URL=http://renderer:8090 \
STUDIO_CERT_ADMIN_EMAIL='<admin email>' \
STUDIO_CERT_ADMIN_PASSWORD='<admin password>' \
bun run certify:live
```

The command requires API/database readiness, the web login page, optional direct renderer readiness, a valid Admin session, live mode, and safe configuration status for `openai`, `seedance`, `r2`, `meta`, and `renderer`. Output contains hostnames and provider names only. The temporary session is revoked in `finally`, including when a later assertion fails.

`STUDIO_CERT_ALLOW_INSECURE_LOCAL=true` permits HTTP only for `localhost`/loopback targets. It must not be used for a remotely accessible staging environment. If the renderer is internal-only, run the command inside the deployment network or omit `STUDIO_CERT_RENDERER_URL`; the API still has to report renderer configuration as complete.

## Credential and workflow certification

The automated gate proves configuration is loaded, not that every external credential still has permission. In the provider consoles or an explicitly authorized staging window, record the operator, UTC time, release digest, provider account/project identifiers (never secrets), and these results:

1. OpenAI: read-only project/model access succeeds; one minimal generation is run only with an approved spend allowance and its trace/cost record is reconciled.
2. Seedance: model access and callback/polling are verified; create at most one approved short staging task, then confirm private R2 copy, provider URL removal, QC, and cost record.
3. R2: upload, HEAD, signed read, multipart resume, and cleanup pass in the staging bucket with no public object exposure.
4. Meta: OAuth scopes, Page/Instagram/ad-account/pixel discovery, test publishing, PAUSED Ads creation, exact activation/budget confirmation, insight sync, and disconnect/token deletion pass on test assets only.
5. Renderer: a signed manifest produces the expected 1080×1920 H.264/AAC output, captions/overlays remain in safe zones, and an identical request is reused.
6. Operations: request/trace/job IDs correlate; sanitized provider failures contain no token, raw payload, signed URL, or personal data; alerts and rollback owners acknowledge the release.

Do not certify with production customer data or a spend-capable production ad account. A failed credential/workflow check blocks live enablement but does not justify bypassing approval, idempotency, or PAUSED-by-default guardrails.

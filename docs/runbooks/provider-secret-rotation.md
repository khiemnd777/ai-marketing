# Provider and application secret rotation

Store production secrets in the deployment platform secret store, never in Compose files, Git, browser-visible variables, logs, River arguments, or support tickets. Use at least two operators for production rotation and record an audit/incident reference.

## Low-impact provider keys

For `OPENAI_API_KEY`, `BYTEPLUS_MODELARK_API_KEY`, R2 credentials, `META_APP_SECRET`, webhook secrets, and `RENDERER_SHARED_SECRET`:

1. Create the replacement credential with least privilege while the old credential remains valid.
2. Update API, worker, and renderer services that consume it. `RENDERER_SHARED_SECRET` must change on worker and renderer in the same maintenance window.
3. Redeploy, verify provider status, execute only a non-paid or explicitly approved smoke request, and watch errors for 15 minutes.
4. Revoke the old credential and verify it fails outside production. Record provider key ID/fingerprint only—not the secret.

Changing `SEEDANCE_WEBHOOK_SECRET` requires updating the registered callback before revoking the old value. Changing R2 credentials requires validating multipart upload, signed GET, and renderer read/write paths.

## Session secret

Rotating `SESSION_SECRET` invalidates all active sessions. Enable maintenance mode, deploy API instances with the new value together, confirm login/CSRF behavior, and ask operators to sign in again.

## Meta-token encryption key

`ENCRYPTION_KEY` is AES-256-GCM key material and no old key is retained by the runtime. Use this safe reconnect procedure:

1. Enable maintenance mode and stop workers.
2. In the Meta settings page, disconnect every active workspace connection. Disconnect wipes token ciphertext and discovered account state.
3. Confirm `meta_connections` has no active row and no non-empty token ciphertext for disconnected rows.
4. Replace `ENCRYPTION_KEY` on API and worker together and redeploy.
5. Reconnect each workspace through OAuth, re-check Page/Instagram/Business/Ad Account/Pixel/Audience discovery, then restart workers and disable maintenance.

Never change `ENCRYPTION_KEY` while encrypted active connections remain; those tokens would become undecryptable. If the old key is suspected compromised, revoke Meta sessions/provider credentials before reconnecting.

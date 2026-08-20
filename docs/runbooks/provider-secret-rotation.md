# Provider and application secret rotation

Provider credentials are entered per client through **Settings → Providers**, encrypted with AES-256-GCM, and stored in PostgreSQL. Infrastructure secrets remain in the deployment platform secret store. Never put either class in Compose files, Git, browser-visible variables, logs, River arguments, or support tickets. Use at least two operators for production rotation and record an audit/incident reference.

## Low-impact provider keys

For a client's OpenAI/Seedance API keys, R2 credentials, Meta App Secret, and Seedance webhook secret:

1. Create the replacement credential with least privilege while the old credential remains valid.
2. As an Admin, select the exact client in **Settings → Providers**, replace only that provider's secret, and save. A blank secret preserves the current value; the explicit clear control removes it.
3. Verify that the selected client is `CONFIGURED`, execute only a non-paid or explicitly approved smoke request, and watch errors for 15 minutes. No API/worker/renderer redeploy is required.
4. Revoke the old credential and verify it fails outside production. Record provider key ID/fingerprint only—not the secret.

Changing a Seedance webhook secret requires updating the registered callback before revoking the old value. Changing R2 credentials requires validating multipart upload, signed GET, and renderer read/write paths for that client. Repeat the process separately for every affected client; never copy one tenant's credentials into another tenant's profile.

## Renderer internal authentication

`RENDERER_INTERNAL_AUTH_SECRET` is infrastructure authentication, not a client provider setting. Rotate it on API/worker and renderer in the same maintenance window, redeploy those services together, and verify a signed no-cost render.

## Session secret

Rotating `SESSION_SECRET` invalidates all active sessions. Enable maintenance mode, deploy API instances with the new value together, confirm login/CSRF behavior, and ask operators to sign in again.

## Database encryption key

`ENCRYPTION_KEY` protects both client provider secrets and Meta OAuth tokens. No old key is retained by the runtime. Use this safe re-entry procedure:

1. Enable maintenance mode and stop workers.
2. Inventory every client provider profile and obtain replacement/current credentials through the approved secret-management channel. Disconnect every active Meta workspace connection.
3. Clear every stored provider secret through **Settings → Providers** and confirm no active Meta token ciphertext or provider secret ciphertext remains that must survive the rotation.
4. Replace `ENCRYPTION_KEY` on API and worker together and redeploy.
5. Re-enter each client's provider credentials in its own profile, reconnect each workspace through OAuth, re-check discovery, then restart workers and disable maintenance.

Never change `ENCRYPTION_KEY` while encrypted active connections or provider credentials remain; they would become undecryptable. If the old key is suspected compromised, revoke Meta sessions and provider credentials before reconnecting or re-entering them.

# Failure diagnosis and recovery

Start at `/operations`: capture request ID, River job ID, entity ID, queue, attempt, normalized error code, provider model/API version, and current maintenance state. Never copy raw credentials, signed URLs, provider payloads, or personal data into incident notes.

## Retry decision

- Retry only after the cause is understood and the operation is idempotent or provider state has been reconciled.
- Seedance submit, Meta publish, and Meta Ads mutations use conservative semantics around ambiguous network outcomes. Look up the provider task/post/campaign using the stored normalized reference before retrying.
- Renderer retries are safe only when the immutable manifest and input checksums still match. R2 retries must reuse the scoped object key and verify checksum.
- Never use Retry to bypass human approval, budget caps, expired Meta access, moderation, permission denial, or Product Truth validation.

## Common incidents

### Queue stuck or discarded

Check database readiness, worker replicas, queue-specific concurrency, oldest job age, and the latest error. Cancel obsolete work. For retryable infrastructure failures, repair the dependency and use the admin Retry control once. A discarded chargeable one-attempt job requires provider reconciliation and a new operator decision.

### Seedance

For timeout, retain the provider task ID and let polling reconcile it. For moderation or permission errors, treat as permanent and revise approved inputs. For rate limiting, respect normalized retry timing; do not manually fan out jobs. Temporary provider URLs must never be treated as durable output—successful results must exist in private R2.

### Renderer or R2

Check renderer readiness, temporary disk, Chromium/ffmpeg process health, manifest HMAC, input SHA-256, R2 connectivity, and output validation. Failed temp directories are process-scoped and cleaned after the attempt; node-level disk alerts require host intervention. Do not mark a render successful without 1080×1920, 30 fps, duration, H.264 decode, and required audio checks.

### Meta publishing and Ads

Expired token/data-access windows require reconnect. Permission denial requires correcting Meta roles/scopes. Campaign create and activation are separate: create remains PAUSED, and activation/budget increases require exact human confirmation. Reconcile provider status before retrying an ambiguous action; never create a second campaign as a blind retry.

### Webhooks

Verify signature/token, delivery hash, event time, and normalized event state. Duplicate delivery is expected and must remain idempotent. A replay is safe only after the handler bug is fixed; do not alter the stored payload hash.

## Maintenance mode

Maintenance mode blocks authenticated non-GET mutations except its own admin endpoint. Enable it for restores, encryption-key rotation, or systemic integrity risk. Read endpoints, health, metrics, and diagnosis remain available. Record the reason and disable it only after a named operator validates readiness, queue state, costs, and provider status.

---
name: studio-provider-integration
description: Add, change, debug, or review an external API contract or typed adapter for OpenAI, Seedance/ModelArk, transcription, R2-compatible storage, or Meta. Use for provider HTTP/auth, webhooks, retries, traces, configuration, and demo parity; use the video skill for downstream pipeline-only changes.
---

# Studio Provider Integration

Keep external contracts outside business logic and make failures safe, observable, and reproducible without live spend.

## Confirm the contract

1. Identify the owning typed interface and its live, demo, and unavailable/configuration behavior.
2. Check current official provider documentation when endpoints, fields, model capabilities, authentication, webhook semantics, or API versions matter. Record durable differences in the relevant ADR.
3. Treat model IDs, API versions, endpoints, prices, timeouts, and feature flags as configuration.

## Implement the adapter

- Normalize provider requests, states, outputs, usage, and errors at the adapter boundary. Business services must not depend on raw provider envelopes.
- Bound HTTP bodies, deadlines, redirects, retries, and downloads. Classify configuration, authentication, moderation, invalid request, rate limit, timeout, outage, and ambiguous outcomes.
- Never log or return API keys, OAuth tokens, raw authorization headers, presigned URLs, temporary output URLs, or unredacted payloads. Persist only sanitized traces and normalized safe response fields.
- Verify webhook signatures when documented, deduplicate payloads, and process callbacks idempotently. When signatures are unavailable, follow the documented application-token and polling fallback decision rather than inventing trust.
- Derive idempotency from the complete immutable operation. Never automatically retry an ambiguous chargeable submission. Safe polling, download, normalization, and reconciliation may retry when proven idempotent.
- Copy provider-hosted media into private object storage immediately, verify expected hashes/metadata, and expose only short-lived application-controlled URLs.
- Append usage/cost ledger records for every provider attempt or reuse decision while keeping estimates distinct from reported cost.
- Keep deterministic demo adapters behaviorally equivalent at the normalized boundary; they must not claim live-provider success.

## Guard consequential operations

Publishing, ad activation/resume, budget increases, paid generation, and recommendation application require explicit human action, valid version/hash-bound approval, server-side authorization, and an internal idempotency key. Meta campaigns are created paused.

## Verify

- Use provider fixtures and failure matrices; automated tests must not contact paid services.
- Cover success, auth failure, rate limit, outage, timeout, malformed response, moderation where relevant, replay, and ambiguous-submit behavior.
- Run the nearest service/worker tests plus integration coverage for persisted normalized state, audit, usage, and idempotency.
- Run live read-only probes only when explicitly authorized and configured; never infer permission for a paid or mutating provider request.

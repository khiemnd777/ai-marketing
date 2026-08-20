# ADR-013: Client-scoped database provider configuration

- Status: Accepted
- Date: 2026-08-20

## Context

Provider configuration was loaded globally from process environment variables. That prevented Admin management through the product and made endpoints, models, pricing, credentials, and demo/live mode shared across every tenant.

## Decision

Store one provider profile per `client_id` in PostgreSQL and one record per client/provider. Safe settings are JSON; credential values are AES-256-GCM ciphertext with random nonces and client/provider-bound associated data. Browser responses expose safe settings and configured-field names only, never plaintext credentials. Admin writes use CSRF, role checks, optimistic versions, and value-free audit metadata.

API requests and River jobs resolve the profile from the business record's `client_id` immediately before constructing provider adapters. Provider callbacks carry and verify client scope. The renderer receives the selected client's storage settings in an HMAC-authenticated internal request and does not read provider configuration from its environment.

`ENCRYPTION_KEY` remains an infrastructure root key; storing it beside its ciphertext would defeat encryption. `RENDERER_INTERNAL_AUTH_SECRET` also remains infrastructure service authentication. Neither is provider configuration.

## Consequences

Clients can use different provider accounts, models, endpoints, storage buckets, prices, and demo/live modes without cross-tenant fallback. A missing or incomplete client profile fails closed. Secret rotation normally updates only the selected database profile and requires no deployment; root-key rotation requires coordinated re-entry because the runtime retains no old encryption key.

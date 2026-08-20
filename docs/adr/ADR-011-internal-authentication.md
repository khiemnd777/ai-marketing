# ADR-011: Internal Authentication

- Status: Accepted
- Date: 2026-08-20

## Context

Cloudflare Access may protect a deployment edge, but the application still needs auditable internal identity and role enforcement.

## Decision

Use admin-created `ADMIN`, `OPERATOR`, and `REVIEWER` accounts, Argon2id password hashes, opaque secure HttpOnly session cookies, PostgreSQL session storage, revocation, CSRF protection, rate-limited login, and server-side authorization. No public registration endpoint exists. Optional Cloudflare Access is defense in depth only.

## Consequences

Identity remains valid across deployment environments and security events are attributable. Operators must manage initial bootstrap and password-reset procedures.

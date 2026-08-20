# ADR-003: OpenAPI Contract Generation

- Status: Accepted
- Date: 2026-08-20

## Context

The Next.js UI and Go API must not drift or manually duplicate transport types.

## Decision

Maintain OpenAPI 3.1 in `openapi/openapi.yaml` as the HTTP source of truth. Generate the TypeScript client into `packages/api-client`, validate the document in CI, and test representative request/response fixtures against it. Go handlers implement the same schemas through explicit transport models and contract tests.

## Consequences

Contract changes are reviewed once and generated artifacts are reproducible. Internal domain types remain separate from public transport shapes.

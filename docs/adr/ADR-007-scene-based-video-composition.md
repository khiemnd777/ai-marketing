# ADR-007: Scene-Based Video Composition

- Status: Accepted
- Date: 2026-08-20

## Context

The product needs controlled conversational videos but not a general-purpose nonlinear editor.

## Decision

Model videos as ordered, versioned scenes with immutable approved generation outputs plus trim, audio, transition, and real-product overlay decisions. Exactly two approved characters are configured per conversational scene and one is the primary speaker. Reordering, replacement, and regeneration create new version hashes.

## Consequences

Operators receive the necessary controls while manifests remain deterministic and auditable. Complex timeline features stay out of Phase 1.

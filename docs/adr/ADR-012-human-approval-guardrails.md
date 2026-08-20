# ADR-012: Human Approval Guardrails

- Status: Accepted
- Date: 2026-08-20

## Context

Generative media, public publishing, and paid advertising have reputational and financial consequences that must not be delegated to autonomous model output.

## Decision

Persist version-and-hash-bound approvals for scripts, generated scenes, final renders, social publishing, ad activation, budget increases, and AI recommendations. Meaningful edits append invalidation events. Workers re-check approval validity immediately before side effects. UI controls explain the required approval and cannot collapse approval and execution into an ambiguous fake-success action.

## Consequences

Every consequential action is attributable and replay-resistant. Additional review steps add latency but are an explicit Phase 1 requirement.

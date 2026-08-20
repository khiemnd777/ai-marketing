# ADR-010: Usage and Cost Ledger

- Status: Accepted
- Date: 2026-08-20

## Context

Phase 1 has no customer billing but must understand provider cost, accepted output, regeneration factor, and campaign economics.

## Decision

Append one usage-ledger record per provider attempt or reuse decision, including model, scoped entities, units, generated and accepted seconds, estimated and reported cost, currency, exchange-rate snapshot, outcome, and reuse flag. Corrections append compensating records rather than rewriting history.

## Consequences

Operational cost and later billing migration have an auditable base. Estimates and provider-reported cost remain distinguishable.

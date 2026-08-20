# ADR-004: Product Truth Architecture

- Status: Accepted
- Date: 2026-08-20

## Context

Marketing generation must never silently change exact product names, prices, dimensions, warranty, discount codes, or other verified values.

## Decision

Store approved facts, claims, prohibited claims, evidence, effective dates, and locked values separately from editable product copy. Every AI input is compiled from the effective approved truth snapshot. Deterministic validators run after generation and human edits. Locked values must match canonical text or normalized typed values exactly.

## Consequences

AI output can be rejected with specific findings rather than trusted probabilistically. Product versions and truth versions remain independently auditable.

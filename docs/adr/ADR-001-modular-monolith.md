# ADR-001: Modular Monolith

- Status: Accepted
- Date: 2026-08-20

## Context

Phase 1 has many business capabilities but one internal operations team, one primary database, and tightly coordinated approval workflows. Splitting these transactions across independent services would add deployment and consistency cost before boundaries have operational evidence.

## Decision

Implement the Go backend as one modular monolith with explicit packages for each business capability. Ship separate API and River worker commands from the same module. Keep the Node renderer separate because it has a distinct runtime, media toolchain, scaling profile, and failure mode.

## Consequences

Cross-module workflows can use PostgreSQL transactions and shared observability. Package boundaries remain enforceable and can later become services if scaling or ownership measurements justify it.

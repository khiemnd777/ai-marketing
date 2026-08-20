# ADR-009: Direct Meta Integration

- Status: Accepted
- Date: 2026-08-20

## Context

Phase 1 needs Facebook Page and Instagram Business publishing, Meta Ads creation, and Insights synchronization. Browser automation and aggregators are outside scope.

## Decision

Integrate directly with the configurable Meta Graph API version. Encrypt tokens, record token metadata and permissions, normalize errors, schedule through River, and persist provider IDs. Create ad campaigns non-active by default. Treat activation, resume, budget increases, publishing, and recommendation actions as separate human-approved audited commands.

## Consequences

The studio owns provider contract maintenance and permission review. It gains full auditability and avoids browser-session risk.

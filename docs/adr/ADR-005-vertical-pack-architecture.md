# ADR-005: Vertical Pack Architecture

- Status: Accepted
- Date: 2026-08-20

## Context

Travel luggage is the initial product vertical, while campaign, video, publishing, ads, and analytics must support future verticals without schema rewrites.

## Decision

Define versioned vertical packs containing JSON Schema, claim rules, asset requirements, prompt rules, validation rules, and concept templates. Store validated product-specific data in scoped JSONB keyed by pack and schema version. Keep generic workflow contracts vertical-neutral.

## Consequences

Adding a vertical is a package and validation change instead of a cross-system column expansion. Database constraints enforce pack identity while application validators enforce pack schemas.

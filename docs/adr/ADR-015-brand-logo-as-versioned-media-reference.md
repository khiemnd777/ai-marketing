# ADR-015: Brand Logo as a Versioned Media Reference

- Status: Accepted
- Date: 2026-08-21

## Context

Brand profiles already persist ordered `logo_asset_ids` in immutable brand versions, while Media Library owns upload verification, checksums, review status, usage rights, expiry, signed previews, and deletion. Treating Brand logos as separate uploads or copying Product Media's mutable `product_id` attach model would create a second source of truth and lose the primary-logo order used by rendering.

## Decision

Keep `media_assets` as the only file and lifecycle record. A Brand version stores an ordered list of eligible Media Library asset IDs: the first item is the primary logo used by deterministic rendering and later items are approved alternates. Brand-scoped uploads create `LOGO` assets with category `BRAND_LOGO`; selecting, reordering, or removing a logo is persisted only when the operator saves a new Brand version.

Eligible logos must belong to the same client and workspace, must not belong to a product or campaign, must be an approved JPEG, PNG, or WebP image, must be unexpired, and must have a verified current upload with a SHA-256 checksum. Assets owned by another Brand cannot be selected. A current Brand logo cannot be deleted or moved away from `APPROVED` until it is removed from the Brand profile.

Final-render idempotency includes the Brand version and primary-logo asset version/checksum. The renderer independently rechecks logo eligibility and fails closed instead of silently omitting a configured logo. Meaningful logo or referenced-media mutations invalidate affected final-render approvals and clear selected final renders.

## Consequences

Brand and Media Library show the same asset status, rights, expiry, and preview without duplicating storage. Primary-logo order remains immutable and auditable through Brand versions. Operators must complete media processing and review before saving a logo selection, and must remove a selected logo from the Brand before rejecting, archiving, or deleting it. SVG remains out of scope until a sanitized rasterization path is defined.

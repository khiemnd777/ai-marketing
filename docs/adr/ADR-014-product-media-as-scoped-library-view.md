# ADR-014: Product Media as a Scoped Media Library View

- Status: Accepted
- Date: 2026-08-21

## Context

Product Truth needs product photography and footage for video generation, while the workspace already has a Media Library with upload verification, usage rights, expiry, review status, and object-storage versions. A second Product Media store would duplicate state and make it unclear which approval or file version the pipeline uses.

## Decision

Keep `media_assets` as the only media system of record. Product Media is a contextual product-detail view backed by `media_assets.product_id` and supports product-aware upload, safe association of an unassigned Library asset, and detach without deletion. Reassignment between products is never implicit.

Vertical packs define minimum media categories for product approval. Readiness counts only visual assets that are approved, unexpired, and backed by a verified current upload version. Scene references and product attachments must belong to the campaign product and satisfy the same eligibility rules; API and worker paths re-check them before a paid provider submission.

Media approval remains distinct from Product Truth fact and claim approval. A meaningful change to associated or referenced media returns affected product or scene state to review and invalidates downstream approvals where applicable.

## Consequences

Users see the same asset, status, rights, and version in Product Truth and Media Library, with no duplicate lifecycle. Product pages can explain readiness in product terms, while the Library remains the place for workspace-wide organization. Generation fails closed if an asset becomes ineligible after scene approval.

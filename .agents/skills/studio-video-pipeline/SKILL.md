---
name: studio-video-pipeline
description: Implement or diagnose AI Product Marketing Studio media upload, normalized generation state, transcription, QC, take selection, scene composition, Remotion rendering, or final-video workflows. Use for pipeline behavior across River, storage, and renderer; pair with the provider skill only when the external API contract changes.
---

# Studio Video Pipeline

Preserve provenance and deterministic verification from source media through the selected final output.

## Load the relevant decisions

Read only the paths involved, plus the applicable documents:

- Seedance submission/callback/polling: docs/adr/ADR-006-seedance-provider-adapter.md.
- Scene composition and renderer boundary: docs/adr/ADR-007-scene-based-video-composition.md and docs/adr/ADR-008-remotion-renderer.md.
- Object lifecycle: docs/storage/r2-lifecycle.md.

## Preserve the pipeline invariants

- Every job identifies immutable scoped entity versions and hashes. Repeated requests reuse safe existing work through a complete fingerprint.
- Paid generation submission has one automatic attempt; reconciliation, polling, private download, transcription, QC, and rendering can retry only when idempotent.
- Provider output URLs never reach the browser or remain authoritative. Copy to private storage, compute SHA-256, inspect with ffprobe, and persist normalized media metadata.
- Store dialogue/product facts as structured data. Do not ask a generative video model to rasterize exact names, prices, dimensions, discount codes, warranty, CTA, QR, or legal text; render canonical values as deterministic overlays or real product media.
- Deterministic validation runs after AI generation and human edits. Human visual/audio review remains explicit before take or final approval.
- A meaningful trim, mute, replacement, subtitle, overlay, ordering, transition, or audio change creates a new version/hash and invalidates affected approvals.
- Renderer requests are signed and versioned, every input hash is verified, temporary files are isolated and cleaned, and outputs are fully decoded/inspected before persistence.
- Keep all Remotion packages on exactly the same pinned version and preserve the Node.js 24 renderer isolation.

## Verify proportionally

- Add deterministic provider fixtures and Go tests for state transitions, callback replay, idempotency, QC, and approval invalidation.
- Add renderer golden/unit tests for manifest planning, auth, input checksums, and reuse.
- For renderer-affecting changes, run a real no-cost smoke through private MinIO and validate resolution, ratio, fps, codecs, duration, audio, thumbnail, checksum, full decode, and repeated-request reuse.
- Cover the supported 30/45-second Vietnamese/English workflow and 9:16 output where the change can affect timing, captions, or layout.
- Never use a live paid generation call in automated verification.

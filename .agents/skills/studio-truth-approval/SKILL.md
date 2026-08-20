---
name: studio-truth-approval
description: Implement or review AI Product Marketing Studio changes involving Product Truth, claims, generated copy, human approvals, approval invalidation, publishing, ads spend, or AI recommendations. Use whenever incorrect facts or autonomous consequential actions are plausible risks.
---

# Studio Truth and Approval

Make factual and consequential workflows fail closed with evidence, attribution, and replay resistance.

## Build from canonical truth

- Read docs/adr/ADR-004-product-truth-architecture.md and docs/adr/ADR-012-human-approval-guardrails.md for changes in this domain.
- Compile AI inputs only from effective approved facts, claims, prohibited claims, and locked values for the scoped product/workspace.
- Treat locked values as immutable through generation and human editing. Compare canonical structured values deterministically; do not rely on a second model to decide factual correctness.
- Keep generic campaign, AI, video, publishing, ads, and analytics contracts free of travel-luggage fields. Resolve vertical behavior through the vertical pack.
- Never fabricate testimonials or imply real-person likeness/voice authorization. Keep exact product facts out of generative pixels.

## Bind approval to the exact content

- Approval records bind entity type/ID, version, content hash, approver, timestamp, and notes.
- Classify every mutation as meaningful or metadata-only. Meaningful content, scene, media, overlay, targeting, budget, or recommendation changes increment the version/hash and append an invalidation event.
- Store approval, audit, invalidation, and dependent job insertion in the same transaction when partial success would create a false authorization state.
- Workers and side-effecting handlers re-read the current version/hash and approval immediately before execution. A queued approval snapshot is not sufficient.
- Rejection and review notes remain attributable; do not overwrite history to simulate a clean state.

## Keep humans in control

- AI may propose but cannot publish, activate/resume Meta campaigns, increase budgets, or apply recommendations.
- Paid mutations require an explicit user action, exact amount/impact confirmation where applicable, server-side role checks, valid approval, audit, and idempotency.
- Create Meta campaigns paused. Separate approval from execution in both API and UI.
- Demo/test flows never call paid providers or claim a live action occurred.

## Test the negative paths

Cover stale versions, changed hashes, invalidated approvals, cross-workspace access, disabled/unauthorized actors, repeated idempotency keys, concurrent attempts, and worker-time approval changes. The test must prove the forbidden side effect did not occur, not merely that an error was returned.

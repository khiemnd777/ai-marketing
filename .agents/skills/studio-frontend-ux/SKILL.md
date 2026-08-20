---
name: studio-frontend-ux
description: Implement or review AI Product Marketing Studio frontend and UX changes in apps/web, including forms, workflow pages, responsive behavior, accessibility, permissions, and browser QA. Use for UI-focused work; pair with the feature-slice skill when API contracts also change.
---

# Studio Frontend UX

Produce an operable internal workflow, not just a visually complete screen.

## Fit the existing frontend

- Use Next.js 16 App Router, React 19, strict TypeScript, Tailwind CSS 4, existing shadcn-style primitives, TanStack Query, React Hook Form, and Zod.
- Consume the generated OpenAPI client through apps/web/src/lib/api.ts and the runtime /api/studio proxy. Do not create parallel DTOs or bypass CSRF/session behavior.
- Reuse layout, tokens, controls, and interaction patterns already present. Do not redesign unrelated screens or introduce a second component system.
- Keep query keys scoped by client/workspace and entity identity. After mutations, update or invalidate every affected view; handle optimistic-version conflicts explicitly.

## Design the complete experience

- Provide loading, empty, error with retry, disabled/pending, success, and stale/conflict states where applicable.
- Reflect roles in navigation and controls, but rely on server authorization. Explain why a consequential action is unavailable rather than presenting a dead control.
- Use real labels, semantic headings, keyboard order, visible focus, accessible validation messages, and status/alert semantics. Move focus to the first invalid field or new error when the workflow needs it.
- Maintain at least 44px touch targets for primary mobile controls, usable layouts at narrow and desktop widths, no essential hover-only behavior, and reduced-motion compatibility.
- Separate approval from execution. Never show success before the API confirms persisted state, and never collapse publish, spend, activation, or budget change into an ambiguous action.
- Preserve unsaved work intentionally through autosave, dirty-state protection, or explicit confirmation according to the existing workflow.

## Verify behavior

- Add Vitest/Testing Library coverage for validation, permissions, state transitions, and query invalidation—not wording snapshots alone.
- Build the production Next.js app to catch server/client boundary issues.
- Browser-test the real local route through /api/studio at desktop and mobile widths. Exercise keyboard focus plus one meaningful failure or conflict state.
- Use deterministic demo data and never trigger live publishing, ads, or paid provider calls during UI QA.

If the request includes schema or transport changes, apply $studio-feature-slice and $studio-database-change as appropriate.

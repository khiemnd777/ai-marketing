# ADR-006: Seedance Provider Adapter

- Status: Accepted
- Date: 2026-08-20

## Context

BytePlus ModelArk video contracts, model identifiers, supported reference inputs, and cancellation behavior can vary by deployed model and API version. Generation is chargeable and outputs may initially live at temporary URLs.

## Decision

Place Seedance behind a typed provider interface with configurable base URL, model, and API version. Normalize provider states, sanitize stored envelopes, verify webhooks when supported, poll as fallback, and copy successful outputs to private R2 immediately. Derive an idempotency key from the complete immutable generation specification and never automatically retry an ambiguous chargeable submission.

The Phase 1 adapter is pinned by configuration to the current ModelArk v3 task contract:

- create: `POST /api/v3/contents/generations/tasks`;
- retrieve: `GET /api/v3/contents/generations/tasks/{task_id}`;
- cancel queued task: `DELETE /api/v3/contents/generations/tasks/{task_id}`;
- model default: `dreamina-seedance-2-0-260128`.

The typed content union supports prompt text and image, video, and audio references. Reference counts are validated before submission. Provider task statuses are mapped to the studio's larger lifecycle, and the temporary `content.video_url` is never returned to the browser or retained after the output is copied to private object storage.

ModelArk's documented callback payload is the same task representation as retrieval, but its public contract does not document a signing header. The callback therefore requires an application-generated high-entropy token embedded in `callback_url`, compares it in constant time, stores a payload hash for replay protection, and treats authenticated callbacks as an optimization. Polling remains the correctness fallback. A future documented signature scheme can be added inside the same adapter boundary.

Primary references checked on 2026-08-20:

- [Create video generation task](https://docs.byteplus.com/en/docs/modelark/1520757)
- [Retrieve video generation task](https://docs.byteplus.com/en/docs/ModelArk/1521309)
- [Cancel video generation task](https://docs.byteplus.com/en/docs/ModelArk/1521720)
- [Video generation tutorial and callback behavior](https://docs.byteplus.com/en/docs/ModelArk/2298881?redirect=1)

## Consequences

Business workflows do not depend on provider payload shapes or temporary URLs. Provider capability differences surface as typed configuration or unsupported-operation errors rather than optimistic success. Locally queued and provider-queued work can be cancelled; ModelArk running work cannot. Safe polling and output-copy jobs retry, while the chargeable create request has exactly one automatic attempt.

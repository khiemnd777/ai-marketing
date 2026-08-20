# Performance and capacity review

Review date: 2026-08-20.

- PostgreSQL scope/time and state indexes cover workspace lists, provider polling, campaign analytics, costs, recommendations, Meta expiry, and River queues. Analytics scans are bounded to 366 days, derived rates use raw counts, and a concurrently refreshed daily materialized view supports future long-window reads without blocking writers.
- API pool usage is exported. Request timeout/body limits are bounded; list endpoints paginate or constrain output; object bytes bypass the API through signed R2 operations.
- River isolates AI, Seedance submit/status/download, transcription, QC, rendering, publishing, Meta Ads, metrics, and maintenance queues. Expensive/chargeable queues have lower concurrency; job age, retries, attempts, and discarded states alert independently.
- Renderer work is isolated from API/worker, uses per-render cleanup, validates existing immutable output for reuse, and has a 20-minute client deadline. Scale renderer replicas only after confirming temporary disk, memory, Chromium process limits, and object-store bandwidth.
- Operations and analytics endpoints are polled at 10–30 seconds, not per animation frame. Frontend queries are keyed by scope/filter and mutations invalidate only relevant caches.

Before increasing production quotas, run representative 30/45-second render soak tests, PostgreSQL query-plan sampling on production-shaped volumes, and provider rate-limit tests in an authorized sandbox. Capacity triggers are: p95 API latency over 500 ms, database pool saturation over 80%, runnable job age over 15 minutes, attempt factor over 2, renderer temp disk over 70%, or daily provider cost outside the approved operating envelope.

# ADR-002: PostgreSQL and River Without Redis

- Status: Accepted
- Date: 2026-08-20

## Context

Jobs must survive restarts, coordinate with domain records, and support diagnosis, retry, and cancellation. The platform already requires PostgreSQL 18.

## Decision

Use PostgreSQL as the system of record and River for asynchronous jobs. Do not introduce Redis. Use dedicated River queues and immutable version identifiers in job arguments.

## Consequences

Domain state and job state share transactional infrastructure and backup procedures. Queue load must be monitored through database pool, queue depth, and job age metrics; a future cache requires measured evidence and a separate ADR.

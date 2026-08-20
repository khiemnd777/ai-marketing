# Data retention policy

| Data | Default retention | Disposal behavior |
| --- | ---: | --- |
| Sessions and OAuth state | Until expiry; consumed Meta OAuth state for 24 hours | Hourly maintenance deletes rows |
| Idempotency keys | Until expiry | Hourly maintenance deletes rows |
| Sanitized webhook bodies | 90 days | Body is cleared; delivery hash, signature result, timestamps, and error remain |
| Completed/cancelled River jobs | 30 days | Hourly maintenance deletes terminal rows |
| Discarded River jobs | 90 days | Retained longer for incident diagnosis, then deleted |
| Provider request/output traces | 180 days online | Export normalized trace metadata if longer evidence is required; never retain unsanitized secrets |
| Usage ledger, costs, approvals, Ads metrics | 24 months online | Aggregate/export before controlled deletion |
| Audit logs | 24 months minimum | Immutable export to restricted archive before any deletion |
| Media assets and generated outputs | Active campaign lifetime plus 90 days after approved deletion/archive | Soft-delete first, then object lifecycle deletes only after backup/recovery windows |
| Character consent records | Asset lifetime plus 7 years | Restricted archive; legal hold overrides deletion |
| PostgreSQL backups | 30 daily and 12 month-end | KMS-encrypted deletion by backup policy |

Client deletion or legal-hold requests override defaults and require an approved ticket. Object deletion must verify there is no active media version, selected generation, render, post, Ad creative, consent obligation, legal hold, or retained database backup reference. Metrics and recommendations may retain non-identifying aggregates after source deletion. All retention jobs run in UTC and are auditable through operations logs.

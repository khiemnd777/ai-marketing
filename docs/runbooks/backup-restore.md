# PostgreSQL backup and restore verification

## Recovery objectives

- Default production target: daily encrypted database backups, 30 daily copies, 12 month-end copies, RPO at most 24 hours, and RTO at most 4 hours.
- The database backup and R2 object versioning/lifecycle are one recovery set. A database row can reference an object created before the database snapshot, so object versions must outlive the oldest retained database backup.
- Audit logs require a two-year retention export before database pruning. The application maintenance job does not delete audit logs.

## Create and verify a backup

Run from a restricted backup runner with PostgreSQL client tools. `DATABASE_URL` should use a role with read access, TLS verification, and no shell history exposure.

```sh
DATABASE_URL='postgresql://...' BACKUP_DIR=/secure/studio-backups ./infra/backup/backup-postgres.sh
```

The script creates a custom-format dump with mode `0600`, checks that `pg_restore` can read its catalog, and writes a SHA-256 sidecar. Encrypt the resulting files with the infrastructure KMS before remote transfer. Alert if the job, encryption, checksum, or upload fails.

## Monthly restore drill

1. Provision a new isolated PostgreSQL 18 database whose name contains `restore` or `verify`. Block application traffic and provider network access.
2. Verify the backup checksum using `sha256sum -c` or `shasum -a 256 -c`.
3. Run the guarded verification script:

```sh
RESTORE_VERIFY_DATABASE_URL='postgresql://.../studio_restore_verify?sslmode=require' \
BACKUP_FILE=/secure/studio-backups/studio-YYYYMMDDTHHMMSSZ.dump \
ALLOW_DESTRUCTIVE_RESTORE=verify-only \
./infra/backup/restore-verify.sh
```

4. Apply no migrations before the check. Confirm the restored Atlas revision, row-count reasonableness, foreign keys, check constraints, and the newest `audit_logs.occurred_at` timestamp.
5. Start the API against the isolated database with outbound provider traffic denied. Through the Provider UI, keep the restored test client in demo mode and point its R2 profile to the restored test copy; then start the worker. Check `/v1/health/ready`, `/v1/operations/overview`, signed media reads, and one no-cost demo workflow.
6. Record backup timestamp, recovery duration, row counts, object sample results, operator, and any remediation. Destroy the isolated database after evidence is retained.

## Production recovery

Declare an incident, enable maintenance mode, stop worker and renderer processes, restore PostgreSQL and the matching R2 recovery point, validate as above, rotate credentials that may have been exposed, then start migrations, API, worker, renderer, and web in that order. Reconcile ambiguous one-attempt Seedance and Meta jobs with provider state before retrying. Disable maintenance only after health, queue, cost, and sample-object checks pass.

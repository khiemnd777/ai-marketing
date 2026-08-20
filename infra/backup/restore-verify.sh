#!/bin/sh
set -eu

umask 077
: "${RESTORE_VERIFY_DATABASE_URL:?RESTORE_VERIFY_DATABASE_URL is required}"
: "${BACKUP_FILE:?BACKUP_FILE is required}"

if [ "${ALLOW_DESTRUCTIVE_RESTORE:-}" != "verify-only" ]; then
  printf '%s\n' 'Refusing restore: set ALLOW_DESTRUCTIVE_RESTORE=verify-only for an isolated verification database.' >&2
  exit 2
fi
if [ ! -f "$BACKUP_FILE" ]; then
  printf 'Backup does not exist: %s\n' "$BACKUP_FILE" >&2
  exit 2
fi

target_database=$(psql "$RESTORE_VERIFY_DATABASE_URL" -v ON_ERROR_STOP=1 -Atqc 'select current_database()')
case "$target_database" in
  *restore*|*verify*) ;;
  *)
    printf '%s\n' 'Refusing restore: target database name must contain restore or verify.' >&2
    exit 2
    ;;
esac

pg_restore --list "$BACKUP_FILE" >/dev/null
pg_restore --dbname="$RESTORE_VERIFY_DATABASE_URL" --clean --if-exists --no-owner --no-privileges --exit-on-error "$BACKUP_FILE"

psql "$RESTORE_VERIFY_DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE missing text;
BEGIN
  SELECT string_agg(name, ', ') INTO missing
  FROM unnest(ARRAY['internal_users','clients','workspaces','products','campaigns','media_assets','usage_ledger','cost_records','audit_logs']) AS name
  WHERE to_regclass('public.' || name) IS NULL;
  IF missing IS NOT NULL THEN
    RAISE EXCEPTION 'restore missing required tables: %', missing;
  END IF;
END $$;
SELECT 'foreign_keys', count(*) FROM pg_constraint WHERE contype='f';
SELECT 'check_constraints', count(*) FROM pg_constraint WHERE contype='c';
SELECT 'workspaces', count(*) FROM workspaces;
SELECT 'campaigns', count(*) FROM campaigns;
SELECT 'audit_logs', count(*) FROM audit_logs;
SQL

printf 'Restore verification passed for isolated database: %s\n' "$target_database"

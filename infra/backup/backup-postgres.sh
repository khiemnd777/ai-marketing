#!/bin/sh
set -eu

umask 077
: "${DATABASE_URL:?DATABASE_URL is required}"

backup_dir=${BACKUP_DIR:-./backups}
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
backup_file="$backup_dir/studio-$timestamp.dump"
checksum_file="$backup_file.sha256"

mkdir -p "$backup_dir"
pg_dump --dbname="$DATABASE_URL" --format=custom --compress=9 --no-owner --no-privileges --file="$backup_file"
pg_restore --list "$backup_file" >/dev/null

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$backup_file" >"$checksum_file"
else
  shasum -a 256 "$backup_file" >"$checksum_file"
fi

printf 'Backup verified: %s\nChecksum: %s\n' "$backup_file" "$checksum_file"

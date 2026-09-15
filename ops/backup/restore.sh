#!/usr/bin/env bash
# Restores one backup from Cloudflare R2 into a brand-new database named
# "<production-db>_restore_<timestamp>". NEVER touches the production
# database — see cutover.sh for the separate, human-confirmed step that
# promotes a restored database to production.
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=./common.sh
source ./common.sh

backup_id="${1:-}"
[ -n "$backup_id" ] || fatal "Usage: restore.sh <backup-id>  (e.g. restore.sh 2026-09-15T200000Z)"
require_valid_backup_id "$backup_id"

require_postgres_vars
require_r2_vars
setup_pg_env
check_postgres_ready
setup_rclone_config

make_tmpdir tmpdir
remote_prefix="$(r2_prefix_for_backup_id "$backup_id")"

log_info "Downloading manifest for backup ${backup_id}..."
manifest_file="${tmpdir}/manifest.json"
rclone copyto "$(r2_remote_path "${remote_prefix}/manifest.json")" "$manifest_file" 2>/dev/null \
    || fatal "manifest.json not found for backup id '${backup_id}'; this backup does not exist or is incomplete"

jq -e . "$manifest_file" >/dev/null 2>&1 || fatal "manifest.json is not valid JSON; refusing to restore"

m_version="$(jq -r '.version // empty' "$manifest_file")"
m_status="$(jq -r '.status // empty' "$manifest_file")"
m_format="$(jq -r '.format // empty' "$manifest_file")"
m_backup_id="$(jq -r '.backup_id // empty' "$manifest_file")"
m_sha256="$(jq -r '.sha256 // empty' "$manifest_file")"

[ "$m_version" = "1" ] || fatal "Unsupported manifest version '${m_version:-<missing>}'; refusing to guess the format"
[ "$m_status" = "complete" ] || fatal "Backup status is '${m_status:-<missing>}', not 'complete'; refusing to restore"
[ "$m_format" = "pg_dump-custom" ] || fatal "Unsupported backup format '${m_format:-<missing>}'; refusing to restore"
[ "$m_backup_id" = "$backup_id" ] || fatal "Manifest backup_id '${m_backup_id}' does not match requested '${backup_id}'"
[ -n "$m_sha256" ] || fatal "Manifest is missing a sha256 checksum; refusing to restore"

log_info "Downloading dump and checksum..."
dump_file="${tmpdir}/database.dump"
checksum_file="${tmpdir}/database.dump.sha256"
rclone copyto "$(r2_remote_path "${remote_prefix}/database.dump")" "$dump_file" 2>/dev/null \
    || fatal "Failed to download database.dump for backup '${backup_id}'"
rclone copyto "$(r2_remote_path "${remote_prefix}/database.dump.sha256")" "$checksum_file" 2>/dev/null \
    || fatal "Failed to download database.dump.sha256 for backup '${backup_id}'"

log_info "Verifying checksum..."
(cd "$tmpdir" && sha256sum -c database.dump.sha256 >/dev/null) \
    || fatal "Checksum verification FAILED for backup '${backup_id}'; the dump may be corrupted or tampered with. Aborting."

downloaded_hash="$(awk '{print $1}' "$checksum_file")"
[ "$downloaded_hash" = "$m_sha256" ] || fatal "Checksum in manifest.json does not match database.dump.sha256; aborting."

log_info "Verifying archive structure with pg_restore --list..."
pg_restore --list "$dump_file" >/dev/null \
    || fatal "pg_restore --list could not read the downloaded dump; archive appears corrupt. Aborting."

restore_db="${PGDATABASE}_restore_$(backup_id_to_db_suffix "$backup_id")"
require_restore_db_name "$restore_db" "$PGDATABASE"

if db_exists "$restore_db"; then
    fatal "Restore database '${restore_db}' already exists. Refusing to overwrite it; drop it manually first if this restore attempt should be redone."
fi

log_info "Creating restore database '${restore_db}'..."
createdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -O "$PGUSER" "$restore_db" \
    || fatal "createdb failed for '${restore_db}'"

restore_failed=0
log_info "Running pg_restore into '${restore_db}'..."
if ! pg_restore --exit-on-error --no-owner --no-acl --dbname="$restore_db" "$dump_file"; then
    restore_failed=1
fi

if [ "$restore_failed" -eq 0 ]; then
    log_info "Running post-restore sanity checks..."
    check_sql() { psql -v ON_ERROR_STOP=1 -d "$restore_db" -tAc "$1"; }

    [ "$(check_sql 'SELECT 1' | tr -d '[:space:]')" = "1" ] || restore_failed=1
    [ "$(check_sql "SELECT to_regclass('public.links') IS NOT NULL" | tr -d '[:space:]')" = "t" ] || restore_failed=1
    [ "$(check_sql "SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm'" | tr -d '[:space:]')" = "1" ] || restore_failed=1
    [ "$(check_sql "SELECT 1 FROM pg_constraint WHERE conrelid = 'public.links'::regclass AND contype = 'p'" | tr -d '[:space:]')" = "1" ] || restore_failed=1
    [ "$(check_sql "SELECT 1 FROM pg_constraint WHERE conrelid = 'public.links'::regclass AND contype = 'u'" | tr -d '[:space:]')" = "1" ] || restore_failed=1
    [ "$(check_sql "SELECT 1 FROM pg_trigger WHERE tgrelid = 'public.links'::regclass AND tgname = 'update_links_updated_at'" | tr -d '[:space:]')" = "1" ] || restore_failed=1

    if [ "$restore_failed" -ne 0 ]; then
        log_warn "One or more post-restore sanity checks failed."
    fi
fi

if [ "$restore_failed" -ne 0 ]; then
    log_warn "Restore failed; dropping partial restore database '${restore_db}'..."
    require_restore_db_name "$restore_db" "$PGDATABASE"
    dropdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" --if-exists "$restore_db" || true
    fatal "Restore of backup '${backup_id}' FAILED. Production database was not touched."
fi

row_count="$(psql -v ON_ERROR_STOP=1 -d "$restore_db" -tAc 'SELECT COUNT(*) FROM links' | tr -d '[:space:]')"

cat >&2 <<EOF

Restore completed successfully.

Backup:
${backup_id}

Restored database:
${restore_db}

Rows in links:
${row_count}

Production database has NOT been modified.
Next step:
  ./linkhub-backup cutover ${restore_db}
EOF

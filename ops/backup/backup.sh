#!/usr/bin/env bash
# Creates one PostgreSQL logical backup (pg_dump custom format), verifies it
# locally, and uploads it to Cloudflare R2 as a complete, atomically-committed
# backup set: database.dump, database.dump.sha256, manifest.json (in that
# order — manifest.json is the commit marker, see README.md).
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=./common.sh
source ./common.sh

require_postgres_vars
require_r2_vars
setup_pg_env
check_postgres_ready

make_tmpdir tmpdir
dump_file="${tmpdir}/database.dump"
checksum_file="${tmpdir}/database.dump.sha256"
manifest_file="${tmpdir}/manifest.json"

backup_id="$(new_backup_id)"
created_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
log_info "Starting backup ${backup_id} for database '${PGDATABASE}'"

log_info "Running pg_dump (custom format)..."
pg_dump --format=custom --no-owner --no-acl --file="$dump_file" \
    || fatal "pg_dump failed; aborting without uploading anything"

log_info "Verifying archive structure with pg_restore --list..."
pg_restore --list "$dump_file" >/dev/null \
    || fatal "pg_restore --list could not read the dump; aborting without uploading anything"

size_bytes="$(file_size_bytes "$dump_file")"
[ "$size_bytes" -gt 0 ] || fatal "Dump file is empty; aborting without uploading anything"

log_info "Computing SHA-256 checksum..."
(cd "$tmpdir" && sha256sum database.dump >database.dump.sha256)
sha256_hex="$(awk '{print $1}' "$checksum_file")"

server_version="$(psql -tAc "SHOW server_version" | tr -d '[:space:]')"
client_version="$(pg_dump --version | awk '{print $NF}')"

jq -n \
    --argjson version 1 \
    --arg backup_id "$backup_id" \
    --arg created_at "$created_at" \
    --arg database "$PGDATABASE" \
    --arg format "pg_dump-custom" \
    --arg server_version "$server_version" \
    --arg client_version "$client_version" \
    --argjson size_bytes "$size_bytes" \
    --arg sha256 "$sha256_hex" \
    --arg status "complete" \
    '{
        version: $version,
        backup_id: $backup_id,
        created_at: $created_at,
        database: $database,
        format: $format,
        postgres_server_version: $server_version,
        postgres_client_version: $client_version,
        size_bytes: $size_bytes,
        sha256: $sha256,
        status: $status
    }' >"$manifest_file"

setup_rclone_config
remote_prefix="$(r2_prefix_for_backup_id "$backup_id")"
remote_dump="$(r2_remote_path "${remote_prefix}/database.dump")"
remote_checksum="$(r2_remote_path "${remote_prefix}/database.dump.sha256")"
remote_manifest="$(r2_remote_path "${remote_prefix}/manifest.json")"

log_info "Uploading dump to R2 (${remote_prefix}/database.dump)..."
rclone copyto "$dump_file" "$remote_dump" \
    || fatal "Upload of database.dump failed; backup is INCOMPLETE (no manifest was written)"

log_info "Uploading checksum..."
rclone copyto "$checksum_file" "$remote_checksum" \
    || fatal "Upload of database.dump.sha256 failed; backup is INCOMPLETE (no manifest was written)"

log_info "Uploading manifest (commit marker)..."
rclone copyto "$manifest_file" "$remote_manifest" \
    || fatal "Upload of manifest.json failed; backup is INCOMPLETE"

log_info "Verifying uploaded object sizes..."
remote_size="$(rclone lsjson "$remote_dump" --files-only 2>/dev/null | jq -r 'if length==1 then .[0].Size else empty end')"
if [ -z "$remote_size" ] || [ "$remote_size" != "$size_bytes" ]; then
    fatal "Remote dump size verification failed (local=${size_bytes}, remote=${remote_size:-missing}); backup must be considered INCOMPLETE"
fi
for obj in "$remote_checksum" "$remote_manifest"; do
    rclone lsjson "$obj" --files-only 2>/dev/null | jq -e 'length == 1' >/dev/null \
        || fatal "Remote object missing after upload: ${obj}; backup must be considered INCOMPLETE"
done

log_info "Backup complete."
log_info "  backup_id : ${backup_id}"
log_info "  database  : ${PGDATABASE}"
log_info "  size      : $(human_size "$size_bytes") (${size_bytes} bytes)"
log_info "  sha256    : ${sha256_hex}"
log_info "  r2 path   : ${remote_prefix}/"

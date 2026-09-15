#!/usr/bin/env bash
# Lists valid, complete backups stored in Cloudflare R2, newest first.
#
# A backup counts as valid/complete only if its manifest.json exists, is
# parseable, has version=1 and status="complete". Partial backups (dump
# and/or checksum uploaded but no manifest) are silently excluded from the
# normal listing, matching the manifest-as-commit-marker design in backup.sh.
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=./common.sh
source ./common.sh

require_r2_vars
setup_rclone_config

base_remote="$(r2_remote_path "$BACKUP_PREFIX")"

log_info "Scanning ${base_remote} for backups..."
manifest_paths="$(rclone lsjson "$base_remote" --recursive --files-only 2>/dev/null \
    | jq -r '.[] | select(.Path | endswith("/manifest.json")) | .Path')"

if [ -z "$manifest_paths" ]; then
    echo "No backups found under ${BACKUP_PREFIX}/."
    exit 0
fi

rows_file="$(mktemp)"
register_cleanup "$rows_file"

while IFS= read -r path; do
    [ -z "$path" ] && continue
    manifest_json="$(rclone cat "$(r2_remote_path "${BACKUP_PREFIX}/${path}")" 2>/dev/null || true)"
    if [ -z "$manifest_json" ] || ! jq -e . >/dev/null 2>&1 <<<"$manifest_json"; then
        log_warn "Skipping unreadable manifest: ${path}"
        continue
    fi
    version="$(jq -r '.version // empty' <<<"$manifest_json")"
    status="$(jq -r '.status // empty' <<<"$manifest_json")"
    if [ "$version" != "1" ] || [ "$status" != "complete" ]; then
        log_warn "Skipping incomplete/unsupported manifest: ${path} (version=${version:-?}, status=${status:-?})"
        continue
    fi
    backup_id="$(jq -r '.backup_id // empty' <<<"$manifest_json")"
    created_at="$(jq -r '.created_at // empty' <<<"$manifest_json")"
    size_bytes="$(jq -r '.size_bytes // 0' <<<"$manifest_json")"
    database="$(jq -r '.database // empty' <<<"$manifest_json")"
    if ! valid_backup_id "$backup_id"; then
        log_warn "Skipping manifest with invalid backup_id: ${path}"
        continue
    fi
    printf '%s\t%s\t%s\t%s\n' "$backup_id" "$created_at" "$size_bytes" "$database" >>"$rows_file"
done <<<"$manifest_paths"

if [ ! -s "$rows_file" ]; then
    echo "No valid completed backups found under ${BACKUP_PREFIX}/."
    exit 0
fi

printf '%-22s %-22s %-10s %s\n' "BACKUP ID" "CREATED AT" "SIZE" "DATABASE"
sort -t "$(printf '\t')" -k1,1r "$rows_file" | while IFS=$'\t' read -r backup_id created_at size_bytes database; do
    printf '%-22s %-22s %-10s %s\n' "$backup_id" "$created_at" "$(human_size "$size_bytes")" "$database"
done

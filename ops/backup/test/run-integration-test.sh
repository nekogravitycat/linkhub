#!/usr/bin/env bash
# End-to-end integration test for the backup/restore/cutover/rollback
# toolset, run against a throwaway PostgreSQL and a MinIO instance standing
# in for Cloudflare R2 (both defined in docker-compose.test.yml). Never
# touches real production data or a real R2 bucket.
#
# Covers the acceptance drill from ops/backup/README.md ("Manual acceptance
# drill"): backup -> mutate -> restore (isolation check) -> cutover ->
# rollback, plus the fail-closed paths (checksum tampering, wrong cutover
# confirmation, duplicate restore, unknown backup id).
#
# Usage: ./run-integration-test.sh
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

export MSYS_NO_PATHCONV=1
COMPOSE=(docker compose -f docker-compose.test.yml)

pass_count=0
fail_count=0

check() {
    local desc="$1"
    shift
    if "$@"; then
        printf 'PASS: %s\n' "$desc"
        pass_count=$((pass_count + 1))
    else
        printf 'FAIL: %s\n' "$desc"
        fail_count=$((fail_count + 1))
    fi
}

run_backup_tool() {
    "${COMPOSE[@]}" exec -T backup /app/entrypoint.sh "$@"
}

db_slugs() {
    # Usage: db_slugs DBNAME
    "${COMPOSE[@]}" exec -T test-db psql -U linkhub -d "$1" -tAc "SELECT slug FROM links ORDER BY slug"
}

valid_backup_id_like() { [[ "$1" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{6}Z$ ]]; }
starts_with() { case "$1" in "$2"*) return 0 ;; *) return 1 ;; esac; }
no_restore_db_for() {
    local suffix
    suffix="$(printf '%s' "$1" | sed -E 's/-//g; s/T/_/; s/Z$//')"
    ! "${COMPOSE[@]}" exec -T test-db psql -U linkhub -d linkhub -tAc \
        "SELECT 1 FROM pg_database WHERE datname = 'linkhub_restore_${suffix}'" \
        | grep -q 1
}

cleanup() {
    "${COMPOSE[@]}" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "== Building backup image =="
docker build -t linkhub-backup:test -f ../Dockerfile ..

echo "== Starting throwaway test-db + MinIO stack =="
"${COMPOSE[@]}" up -d

echo "== Creating test bucket =="
"${COMPOSE[@]}" exec -T test-minio sh -c \
    "mc alias set local http://localhost:9000 test-access-key test-secret-key >/dev/null && mc mb -p local/test-backups >/dev/null"

echo "== Seeding schema and data (backup-test-a) =="
"${COMPOSE[@]}" exec -T test-db psql -U linkhub -d linkhub -v ON_ERROR_STOP=1 -f - <schema.sql >/dev/null
"${COMPOSE[@]}" exec -T test-db psql -U linkhub -d linkhub -v ON_ERROR_STOP=1 \
    -c "INSERT INTO links (slug, url) VALUES ('backup-test-a', 'https://example.com/a');" >/dev/null

echo "== backup: taking Backup A =="
backup_output="$(run_backup_tool backup 2>&1)"
backup_exit=$?
echo "$backup_output"
check "backup exits 0" [ "$backup_exit" -eq 0 ]
backup_id_a="$(printf '%s' "$backup_output" | sed -n 's/.*backup_id : //p' | tr -d '[:space:]')"
check "backup id A looks like a timestamp" valid_backup_id_like "$backup_id_a"

echo "== list: Backup A should appear =="
list_output="$(run_backup_tool list 2>&1)"
check "list shows backup A" grep -q "$backup_id_a" <<<"$list_output"

echo "== Mutating production data: remove A, add B =="
"${COMPOSE[@]}" exec -T test-db psql -U linkhub -d linkhub -v ON_ERROR_STOP=1 \
    -c "DELETE FROM links WHERE slug = 'backup-test-a'; INSERT INTO links (slug, url) VALUES ('backup-test-b', 'https://example.com/b');" >/dev/null

echo "== restore: restoring Backup A into a new database =="
restore_output="$(run_backup_tool restore "$backup_id_a" 2>&1)"
restore_exit=$?
echo "$restore_output"
check "restore exits 0" [ "$restore_exit" -eq 0 ]
restore_db="$(printf '%s' "$restore_output" | sed -n '/^Restored database:/{n;p}' | tr -d '[:space:]')"
check "restore db name follows naming convention" starts_with "$restore_db" "linkhub_restore_"

echo "== Verifying restore did not touch production =="
prod_slugs="$(db_slugs linkhub)"
restore_slugs="$(db_slugs "$restore_db")"
check "production still has backup-test-b only" [ "$prod_slugs" = "backup-test-b" ]
check "restore db has backup-test-a only" [ "$restore_slugs" = "backup-test-a" ]

echo "== restore: duplicate restore of the same backup id must fail =="
if run_backup_tool restore "$backup_id_a" >/tmp/dup_restore.log 2>&1; then
    dup_exit=0
else
    dup_exit=$?
fi
check "duplicate restore fails (exit != 0)" [ "$dup_exit" -ne 0 ]
check "duplicate restore error mentions 'already exists'" grep -qi "already exists" /tmp/dup_restore.log

echo "== restore: unknown backup id must fail =="
if run_backup_tool restore "1999-01-01T000000Z" >/tmp/unknown_restore.log 2>&1; then
    unknown_exit=0
else
    unknown_exit=$?
fi
check "restore of unknown backup id fails" [ "$unknown_exit" -ne 0 ]

echo "== cutover: wrong confirmation text must abort without changes =="
if echo "not-the-db-name" | run_backup_tool cutover "$restore_db" >/tmp/wrong_confirm.log 2>&1; then
    wrong_confirm_exit=0
else
    wrong_confirm_exit=$?
fi
check "cutover with wrong confirmation fails" [ "$wrong_confirm_exit" -ne 0 ]
check "production unchanged after aborted cutover" [ "$(db_slugs linkhub)" = "backup-test-b" ]

echo "== cutover: correct confirmation promotes the restored database =="
cutover_output="$(echo "linkhub" | run_backup_tool cutover "$restore_db" 2>&1)"
cutover_exit=$?
echo "$cutover_output"
check "cutover exits 0" [ "$cutover_exit" -eq 0 ]
check "production now has backup-test-a" [ "$(db_slugs linkhub)" = "backup-test-a" ]
pre_restore_db="$(printf '%s' "$cutover_output" | sed -n '/^Previous production database preserved as:/{n;p}' | tr -d '[:space:]')"
check "pre-restore db name follows naming convention" starts_with "$pre_restore_db" "linkhub_pre_restore_"

echo "== rollback: correct confirmation reverts the cutover =="
rollback_output="$(echo "linkhub" | run_backup_tool rollback "$pre_restore_db" 2>&1)"
rollback_exit=$?
echo "$rollback_output"
check "rollback exits 0" [ "$rollback_exit" -eq 0 ]
check "production reverted to backup-test-b" [ "$(db_slugs linkhub)" = "backup-test-b" ]

echo "== backup: tampering with an uploaded dump must be caught on restore =="
backup_output_2="$(run_backup_tool backup 2>&1)"
backup_id_b="$(printf '%s' "$backup_output_2" | sed -n 's/.*backup_id : //p' | tr -d '[:space:]')"
"${COMPOSE[@]}" exec -T backup bash -c '
    source /app/common.sh
    setup_rclone_config
    echo corrupted >/tmp/corrupt.dump
    prefix="$(r2_prefix_for_backup_id "'"$backup_id_b"'")"
    rclone copyto /tmp/corrupt.dump "$(r2_remote_path "${prefix}/database.dump")"
'
if run_backup_tool restore "$backup_id_b" >/tmp/tampered.log 2>&1; then
    tampered_exit=0
else
    tampered_exit=$?
fi
check "restore of tampered backup fails" [ "$tampered_exit" -ne 0 ]
check "tampered restore error mentions checksum" grep -qi "checksum" /tmp/tampered.log
check "no restore db created for tampered backup" no_restore_db_for "$backup_id_b"

echo
echo "================================================================"
echo " Results: ${pass_count} passed, ${fail_count} failed"
echo "================================================================"
[ "$fail_count" -eq 0 ]

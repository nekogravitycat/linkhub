#!/usr/bin/env bash
# Shared helpers for the LinkHub PostgreSQL backup/restore toolset.
# Sourced by backup.sh, list.sh, restore.sh, cutover.sh and rollback.sh.
set -Eeuo pipefail

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

log_info() { printf '[%s] INFO  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
log_warn() { printf '[%s] WARN  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
fatal() {
    printf '[%s] FATAL %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2
    exit 1
}

# ---------------------------------------------------------------------------
# Cleanup registry: any file/dir registered here is removed on script exit,
# whether the script succeeds, fails, or is interrupted.
# ---------------------------------------------------------------------------

_CLEANUP_PATHS=()
_cleanup_all() {
    # Never let cleanup itself change the script's exit status: an empty
    # _CLEANUP_PATHS array combined with a false final test would otherwise
    # leak a bogus exit code into the trap (and thus into the process).
    local p
    for p in "${_CLEANUP_PATHS[@]}"; do
        [ -n "$p" ] && rm -rf -- "$p"
    done
    return 0
}
trap _cleanup_all EXIT

register_cleanup() { _CLEANUP_PATHS+=("$1"); }

make_tmpdir() {
    # Usage: make_tmpdir OUTVAR
    local -n __outvar="$1"
    __outvar=$(mktemp -d)
    register_cleanup "$__outvar"
}

# ---------------------------------------------------------------------------
# Environment validation
# ---------------------------------------------------------------------------

require_vars() {
    local missing=() name
    for name in "$@"; do
        if [ -z "${!name:-}" ]; then
            missing+=("$name")
        fi
    done
    if [ "${#missing[@]}" -gt 0 ]; then
        fatal "Missing required environment variable(s): ${missing[*]}"
    fi
}

require_postgres_vars() {
    require_vars POSTGRES_ADDR POSTGRES_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
}

require_r2_vars() {
    require_vars R2_ENDPOINT R2_BUCKET R2_ACCESS_KEY_ID R2_SECRET_ACCESS_KEY BACKUP_PREFIX
}

# ---------------------------------------------------------------------------
# PostgreSQL connection setup (libpq standard environment variables)
# ---------------------------------------------------------------------------

setup_pg_env() {
    export PGHOST="$POSTGRES_ADDR"
    export PGPORT="$POSTGRES_PORT"
    export PGUSER="$POSTGRES_USER"
    export PGPASSWORD="$POSTGRES_PASSWORD"
    export PGDATABASE="$POSTGRES_DB"
}

check_postgres_ready() {
    pg_isready -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -q \
        || fatal "PostgreSQL is not reachable at ${PGHOST}:${PGPORT} (pg_isready failed)"
    psql -v ON_ERROR_STOP=1 -tAc "SELECT 1" >/dev/null \
        || fatal "PostgreSQL connectivity check failed (SELECT 1)"
}

# psql against the 'postgres' maintenance database, for statements that must
# not run against the database they might be renaming/dropping.
psql_maint() {
    psql -v ON_ERROR_STOP=1 -d postgres "$@"
}

db_exists() {
    local name="$1" out
    out=$(psql_maint -tAc "SELECT 1 FROM pg_database WHERE datname = '${name}'")
    [ "$(printf '%s' "$out" | tr -d '[:space:]')" = "1" ]
}

terminate_backend_connections() {
    # Usage: terminate_backend_connections DBNAME
    local name="$1"
    psql_maint -tAc "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = '${name}' AND pid <> pg_backend_pid();
    " >/dev/null
}

# ---------------------------------------------------------------------------
# Identifier / name validation
#
# These are the last line of defense against SQL injection and destructive
# operations against the wrong database. Every function fails closed: on any
# doubt, return non-zero rather than guessing.
# ---------------------------------------------------------------------------

SYSTEM_DATABASES=("postgres" "template0" "template1")

is_system_database() {
    local name="$1" sysdb
    for sysdb in "${SYSTEM_DATABASES[@]}"; do
        [ "$name" = "$sysdb" ] && return 0
    done
    return 1
}

# A bare PostgreSQL identifier: letters/digits/underscore, must not start
# with a digit, max 63 bytes (PostgreSQL's NAMEDATALEN limit).
valid_identifier() {
    local name="$1"
    [[ "$name" =~ ^[A-Za-z_][A-Za-z0-9_]{0,62}$ ]]
}

require_valid_identifier() {
    local name="$1" label="${2:-identifier}"
    valid_identifier "$name" || fatal "Invalid ${label}: '${name}' is not a safe PostgreSQL identifier"
    is_system_database "$name" && fatal "Refusing to operate on system database '${name}'"
    return 0
}

# Restore databases are always named "<prod-db>_restore_<YYYYMMDD>_<HHMMSS>".
require_restore_db_name() {
    local name="$1" prod_db="$2"
    require_valid_identifier "$name" "restore database name"
    [[ "$name" =~ ^${prod_db}_restore_[0-9]{8}_[0-9]{6}$ ]] \
        || fatal "Refusing to operate: '${name}' does not match the expected '${prod_db}_restore_<timestamp>' pattern"
}

# Pre-restore snapshots are always named "<prod-db>_pre_restore_<YYYYMMDD>_<HHMMSS>".
require_pre_restore_db_name() {
    local name="$1" prod_db="$2"
    require_valid_identifier "$name" "pre-restore database name"
    [[ "$name" =~ ^${prod_db}_pre_restore_[0-9]{8}_[0-9]{6}$ ]] \
        || fatal "Refusing to operate: '${name}' does not match the expected '${prod_db}_pre_restore_<timestamp>' pattern"
}

# ---------------------------------------------------------------------------
# Backup ID helpers
#
# Backup IDs are UTC timestamps of the form 2026-09-15T200000Z. They are used
# both as the R2 object path leaf and (transformed) as part of restore
# database names.
# ---------------------------------------------------------------------------

new_backup_id() {
    date -u +%Y-%m-%dT%H%M%SZ
}

valid_backup_id() {
    local id="$1"
    [[ "$id" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{6}Z$ ]]
}

require_valid_backup_id() {
    local id="$1"
    valid_backup_id "$id" || fatal "Invalid backup id: '${id}' (expected format YYYY-MM-DDTHHMMSSZ)"
}

# Transforms a backup id "2026-09-15T200000Z" into the restore-db-safe
# suffix "20260915_200000".
backup_id_to_db_suffix() {
    local id="$1"
    require_valid_backup_id "$id"
    printf '%s' "$id" | sed -E 's/-//g; s/T/_/; s/Z$//'
}

# R2 object prefix for a given backup id, e.g. "postgres/2026/09/2026-09-15T200000Z"
r2_prefix_for_backup_id() {
    local id="$1"
    require_valid_backup_id "$id"
    local year="${id:0:4}" month="${id:5:2}"
    printf '%s/%s/%s/%s' "$BACKUP_PREFIX" "$year" "$month" "$id"
}

# ---------------------------------------------------------------------------
# rclone / R2 setup
#
# Credentials are written to a temp file with 0600 permissions and cleaned
# up on exit; they are never passed as CLI arguments (which would leak via
# process listings) and never echoed to logs.
# ---------------------------------------------------------------------------

R2_REMOTE_NAME="r2"

setup_rclone_config() {
    require_r2_vars
    local cfg
    cfg=$(mktemp)
    register_cleanup "$cfg"
    umask 077
    cat >"$cfg" <<EOF
[${R2_REMOTE_NAME}]
type = s3
provider = Cloudflare
access_key_id = ${R2_ACCESS_KEY_ID}
secret_access_key = ${R2_SECRET_ACCESS_KEY}
endpoint = ${R2_ENDPOINT}
acl = private
no_check_bucket = true
EOF
    chmod 600 "$cfg"
    export RCLONE_CONFIG="$cfg"
}

r2_remote_path() {
    # Usage: r2_remote_path SUFFIX
    printf '%s:%s/%s' "$R2_REMOTE_NAME" "$R2_BUCKET" "$1"
}

# ---------------------------------------------------------------------------
# Misc
# ---------------------------------------------------------------------------

file_size_bytes() {
    stat -c%s "$1"
}

human_size() {
    local bytes="$1"
    awk -v b="$bytes" 'BEGIN {
        split("B KiB MiB GiB TiB", units, " ")
        u = 1
        v = b
        while (v >= 1024 && u < 5) { v /= 1024; u++ }
        printf "%.1f %s", v, units[u]
    }'
}

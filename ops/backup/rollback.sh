#!/usr/bin/env bash
# Reverts a previous cutover by renaming a preserved
# "<prod>_pre_restore_<timestamp>" database back to production.
#
# This does NOT re-download anything from R2 -- it only operates on
# databases already present on this PostgreSQL server. Must be run via the
# `linkhub-backup rollback` wrapper, which stops/restarts the API/Nginx
# containers; this script only ever touches PostgreSQL.
#
# The database being replaced (the cutover's result) is not dropped: it is
# renamed to "<prod>_failed_restore_<timestamp>" for later inspection.
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=./common.sh
source ./common.sh

pre_restore_db="${1:-}"
[ -n "$pre_restore_db" ] || fatal "Usage: rollback.sh <pre-restore-db-name>"

require_postgres_vars
setup_pg_env
check_postgres_ready

prod_db="$PGDATABASE"
require_pre_restore_db_name "$pre_restore_db" "$prod_db"
require_valid_identifier "$prod_db" "production database name"

db_exists "$pre_restore_db" || fatal "Pre-restore database '${pre_restore_db}' does not exist"
db_exists "$prod_db" || fatal "Production database '${prod_db}' does not exist"
[ "$pre_restore_db" != "$prod_db" ] || fatal "Pre-restore database and production database names must differ"

timestamp="$(date -u +%Y%m%d_%H%M%S)"
failed_db="${prod_db}_failed_restore_${timestamp}"
require_valid_identifier "$failed_db" "failed-restore database name"
[[ "$failed_db" =~ ^${prod_db}_failed_restore_[0-9]{8}_[0-9]{6}$ ]] || fatal "Internal error generating failed-restore name"

cat >&2 <<EOF

=====================================================================
 ROLLBACK
=====================================================================
 Production database (current, will be preserved) :  ${prod_db}
 Pre-restore database (will become production)      :  ${pre_restore_db}

 This will:
   1. Terminate all active connections to '${prod_db}' and '${pre_restore_db}'
   2. Rename '${prod_db}'          -> '${failed_db}'       (kept, not dropped)
   3. Rename '${pre_restore_db}' -> '${prod_db}'

 The API and Nginx containers must already be stopped.
=====================================================================

EOF

read -r -p "Type the production database name (${prod_db}) to confirm rollback: " confirm
[ "$confirm" = "$prod_db" ] || fatal "Confirmation text did not match '${prod_db}'. Aborting; nothing was changed."

log_info "Terminating active connections to '${prod_db}' and '${pre_restore_db}'..."
terminate_backend_connections "$prod_db"
terminate_backend_connections "$pre_restore_db"

log_info "Renaming '${prod_db}' -> '${failed_db}'..."
psql_maint -tAc "ALTER DATABASE \"${prod_db}\" RENAME TO \"${failed_db}\";" \
    || fatal "Failed to rename '${prod_db}' to '${failed_db}'. Production database name is UNCHANGED; nothing was renamed. Investigate active connections and retry."

log_info "Renaming '${pre_restore_db}' -> '${prod_db}'..."
if ! psql_maint -tAc "ALTER DATABASE \"${pre_restore_db}\" RENAME TO \"${prod_db}\";"; then
    log_warn "Failed to rename '${pre_restore_db}' to '${prod_db}'. Attempting to restore original state..."
    if psql_maint -tAc "ALTER DATABASE \"${failed_db}\" RENAME TO \"${prod_db}\";"; then
        fatal "Rollback aborted safely: '${prod_db}' has been restored to its pre-rollback name. '${pre_restore_db}' was left untouched."
    else
        fatal "CRITICAL: rollback partially failed and automatic recovery ALSO failed. Production database currently does not exist under its expected name. Manually rename '${failed_db}' back to '${prod_db}' immediately."
    fi
fi

cat >&2 <<EOF

Rollback completed successfully.

Production database '${prod_db}' has been reverted to the pre-restore state.
The database that was replaced has been preserved as:
  ${failed_db}

It is not deleted automatically. Inspect it, then drop it manually once you
are sure it is no longer needed (see README.md for the manual cleanup
procedure).

EOF

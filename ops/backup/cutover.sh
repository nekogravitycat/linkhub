#!/usr/bin/env bash
# Promotes a restored database ("<prod>_restore_<timestamp>") to production.
#
# This is a destructive, human-confirmed operation. It must be run via the
# `linkhub-backup cutover` wrapper, which stops the API/Nginx containers
# before invoking this script and restarts them afterwards -- this script
# only ever touches PostgreSQL, never Docker.
#
# The previous production database is never dropped: it is renamed to
# "<prod>_pre_restore_<timestamp>" so rollback.sh can restore it quickly.
set -Eeuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=./common.sh
source ./common.sh

restore_db="${1:-}"
[ -n "$restore_db" ] || fatal "Usage: cutover.sh <restore-db-name>"

require_postgres_vars
setup_pg_env
check_postgres_ready

prod_db="$PGDATABASE"
require_restore_db_name "$restore_db" "$prod_db"
require_valid_identifier "$prod_db" "production database name"

db_exists "$restore_db" || fatal "Restore database '${restore_db}' does not exist"
db_exists "$prod_db" || fatal "Production database '${prod_db}' does not exist"
[ "$restore_db" != "$prod_db" ] || fatal "Restore database and production database names must differ"

timestamp="$(date -u +%Y%m%d_%H%M%S)"
pre_restore_db="${prod_db}_pre_restore_${timestamp}"
require_pre_restore_db_name "$pre_restore_db" "$prod_db"

cat >&2 <<EOF

=====================================================================
 PRODUCTION CUTOVER
=====================================================================
 Production database :  ${prod_db}
 Restore database     :  ${restore_db}

 This will:
   1. Terminate all active connections to '${prod_db}' and '${restore_db}'
   2. Rename '${prod_db}'      -> '${pre_restore_db}'   (kept, not dropped)
   3. Rename '${restore_db}' -> '${prod_db}'

 The API and Nginx containers must already be stopped. The previous
 production database is preserved for rollback (see rollback.sh) and
 is NOT deleted by this script.
=====================================================================

EOF

read -r -p "Type the production database name (${prod_db}) to confirm cutover: " confirm
[ "$confirm" = "$prod_db" ] || fatal "Confirmation text did not match '${prod_db}'. Aborting; nothing was changed."

log_info "Terminating active connections to '${prod_db}' and '${restore_db}'..."
terminate_backend_connections "$prod_db"
terminate_backend_connections "$restore_db"

log_info "Renaming '${prod_db}' -> '${pre_restore_db}'..."
psql_maint -tAc "ALTER DATABASE \"${prod_db}\" RENAME TO \"${pre_restore_db}\";" \
    || fatal "Failed to rename '${prod_db}' to '${pre_restore_db}'. Production database name is UNCHANGED; nothing was renamed. Investigate active connections and retry."

log_info "Renaming '${restore_db}' -> '${prod_db}'..."
if ! psql_maint -tAc "ALTER DATABASE \"${restore_db}\" RENAME TO \"${prod_db}\";"; then
    log_warn "Failed to rename '${restore_db}' to '${prod_db}'. Attempting to restore original state..."
    if psql_maint -tAc "ALTER DATABASE \"${pre_restore_db}\" RENAME TO \"${prod_db}\";"; then
        fatal "Cutover aborted safely: '${prod_db}' has been restored to its original name. '${restore_db}' was left untouched."
    else
        fatal "CRITICAL: cutover partially failed and automatic recovery ALSO failed. Production database currently does not exist under its expected name. Manually rename '${pre_restore_db}' back to '${prod_db}' immediately."
    fi
fi

cat >&2 <<EOF

Cutover completed successfully.

Production database '${prod_db}' now serves the restored data.
Previous production database preserved as:
  ${pre_restore_db}

If you need to undo this cutover:
  ./linkhub-backup rollback ${pre_restore_db}

EOF

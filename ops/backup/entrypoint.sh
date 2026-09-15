#!/usr/bin/env bash
set -Eeuo pipefail

cmd="${1:-}"
shift || true

case "$cmd" in
backup) exec /app/backup.sh "$@" ;;
list) exec /app/list.sh "$@" ;;
restore) exec /app/restore.sh "$@" ;;
cutover) exec /app/cutover.sh "$@" ;;
rollback) exec /app/rollback.sh "$@" ;;
*)
    cat >&2 <<'EOF'
Usage: <backup|list|restore|cutover|rollback> [args]

  backup                        Create and upload a new backup
  list                          List valid completed backups in R2
  restore <backup-id>           Restore a backup into a new "*_restore_*" database
  cutover <restore-db-name>     Promote a restored database to production (destructive, confirmed)
  rollback <pre-restore-db-name> Revert a previous cutover (destructive, confirmed)
EOF
    exit 1
    ;;
esac

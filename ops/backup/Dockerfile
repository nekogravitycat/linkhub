# Same major.minor version as the production PostgreSQL server (see
# backend/compose.yml) so that pg_dump/pg_restore/psql are guaranteed
# wire-compatible. Bump both together, deliberately.
FROM postgres:18.2-alpine3.23

RUN apk add --no-cache bash jq rclone coreutils

WORKDIR /app

COPY common.sh backup.sh list.sh restore.sh cutover.sh rollback.sh entrypoint.sh ./
RUN chmod +x ./*.sh

ENTRYPOINT ["/app/entrypoint.sh"]

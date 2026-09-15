# LinkHub PostgreSQL Backup & Restore

Operational tooling for daily PostgreSQL logical backups to Cloudflare R2,
safe restore into an isolated database, a human-confirmed production
cutover, rollback, and full disaster recovery.

This is deliberately simple: one `pg_dump` (custom format) per day, no WAL
archiving, no PITR, no replication. LinkHub can tolerate ~24h of data loss
and restore downtime is not a concern. See the top-level design doc for the
full rationale.

**Core safety rules** (do not weaken these):

- `restore` never touches the production database -- it only ever creates a
  new `*_restore_*` database.
- `cutover` and `rollback` are separate, destructive, human-confirmed steps.
  Both require typing the full production database name.
- The previous production database is never dropped automatically, by
  either `cutover` or `rollback` -- only renamed, so it can always be
  recovered.
- A backup without a `manifest.json` is never considered valid.
- Any checksum mismatch, unknown manifest version, or invalid database name
  aborts immediately (fail closed).

## Contents

- [Setup](#setup)
- [Cloudflare R2 setup](#cloudflare-r2-setup)
- [Day-to-day usage](#day-to-day-usage)
- [systemd daily schedule](#systemd-daily-schedule)
- [Manual cleanup of restore/pre_restore/failed_restore databases](#manual-cleanup)
- [Manual acceptance drill](#manual-acceptance-drill)
- [Disaster Recovery Runbook](#disaster-recovery-runbook)

## Setup

1. Copy `backend/.env.backup.example` to `backend/.env.backup` and fill in
   your Cloudflare R2 credentials (see below). This file must never be
   committed -- it's already covered by `.gitignore`.
2. `backend/.env` must already exist (see `backend/.env.example`) with your
   PostgreSQL credentials -- the backup tooling reuses it.
3. From the repository root, everything runs through the wrapper:

   ```bash
   ./ops/backup/linkhub-backup backup
   ./ops/backup/linkhub-backup list
   ./ops/backup/linkhub-backup restore <backup-id>
   ./ops/backup/linkhub-backup cutover <restore-db-name>
   ./ops/backup/linkhub-backup rollback <pre-restore-db-name>
   ```

   The wrapper only calls Docker Compose; it does not run on Windows without
   a bash environment (Git Bash/WSL) or a Linux host -- this matches the
   Linux + systemd production target described below.

Under the hood this builds and runs a one-shot container
(`docker compose -f backend/compose.yml --profile ops run --rm backup ...`)
on the same Docker network as the `database` service. It is never part of
the normal `docker compose up -d` stack, never publishes a port, and never
talks to PostgreSQL except through the existing internal `database:5432`
address.

## Cloudflare R2 setup

1. Create a **private** R2 bucket, e.g. `linkhub-backups`. Do not enable
   public access.
2. Create an API token scoped to **Object Read & Write** on that bucket
   only -- not a Global API Key, not an account-admin token.
3. Fill `backend/.env.backup`:
   - `R2_ENDPOINT` -- `https://<account-id>.r2.cloudflarestorage.com`
   - `R2_BUCKET` -- the bucket name
   - `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` -- the scoped token
   - `BACKUP_PREFIX` -- object prefix, default `postgres`
4. Add a **Lifecycle Rule** on the bucket to delete objects after
   `BACKUP_RETENTION_DAYS` (default 30). This version of the tooling does
   **not** delete old backups itself -- retention is entirely enforced by
   this R2-side rule.
5. Recommended: enable **Object Lock (Bucket Lock)** with a 7-day
   retention period on the bucket. This makes the last 7 days of backups
   impossible to delete or overwrite, even with valid credentials, which
   limits the blast radius of a compromised R2 token or an operator
   mistake.

   **Important:** Object Lock is a hard guarantee. Once an object is locked,
   it genuinely cannot be deleted before its retention period expires --
   not by you, not by support, not by anyone. Understand this before
   enabling it; it is intentionally difficult to undo.

Objects are stored as:

```
<BACKUP_PREFIX>/<YYYY>/<MM>/<YYYY-MM-DDTHHMMSSZ>/database.dump
<BACKUP_PREFIX>/<YYYY>/<MM>/<YYYY-MM-DDTHHMMSSZ>/database.dump.sha256
<BACKUP_PREFIX>/<YYYY>/<MM>/<YYYY-MM-DDTHHMMSSZ>/manifest.json
```

`manifest.json` is uploaded last and is the single source of truth for
"this backup is complete" -- `list.sh` and `restore.sh` both ignore any
backup id whose manifest is missing, unparseable, or has
`status != "complete"`.

## Day-to-day usage

**Take a backup:**

```bash
./ops/backup/linkhub-backup backup
```

**List backups:**

```bash
./ops/backup/linkhub-backup list
```

**Restore a backup for inspection** (never touches production):

```bash
./ops/backup/linkhub-backup restore 2026-09-15T200000Z
# -> creates database "linkhub_restore_20260915_200000"
```

**Promote a restored database to production** (destructive, requires typing
the production database name):

```bash
./ops/backup/linkhub-backup cutover linkhub_restore_20260915_200000
```

This stops the `api` and `nginx` containers, prompts for confirmation,
terminates active connections, renames the current production database to
`<prod>_pre_restore_<timestamp>` (kept, not dropped), renames the restore
database to the production name, then restarts `api` and `nginx`.

**Undo a cutover:**

```bash
./ops/backup/linkhub-backup rollback linkhub_pre_restore_20260915_203012
```

Same shape as cutover, in reverse: the database being replaced is preserved
as `<prod>_failed_restore_<timestamp>`, never dropped automatically.

## systemd daily schedule

Install on the production host (adjust the path first):

```bash
sudo cp ops/backup/systemd/linkhub-backup.service /etc/systemd/system/
sudo cp ops/backup/systemd/linkhub-backup.timer /etc/systemd/system/
sudo $EDITOR /etc/systemd/system/linkhub-backup.service   # set WorkingDirectory
sudo systemctl daemon-reload
sudo systemctl enable --now linkhub-backup.timer
```

Check schedule and history:

```bash
systemctl list-timers linkhub-backup.timer
journalctl -u linkhub-backup.service
```

`Persistent=true` means a backup that was missed because the host was off
at 04:00 runs as soon as the host is back up. There is no notification
integration in this version -- systemd exit status and the journal are the
only signal. If you need Slack/email/etc alerting later, treat it as a
separate enhancement layered on top of `systemctl status`/journal, not a
change to the scripts themselves.

## Manual cleanup

`*_restore_*`, `*_pre_restore_*` and `*_failed_restore_*` databases are
**never** deleted automatically by any script, including the daily backup
job. Clean them up by hand once you're sure they're no longer needed:

```bash
docker compose -f backend/compose.yml exec database psql -U <user> -l
docker compose -f backend/compose.yml exec database dropdb -U <user> <name>
```

Only ever drop a database whose name matches
`<prod>_restore_*` / `<prod>_pre_restore_*` / `<prod>_failed_restore_*` --
never the production database itself.

## Manual acceptance drill

Before relying on this in production, run the drill described in the
design doc once against real (or realistic) data, and ideally rerun
`ops/backup/test/run-integration-test.sh` (see below) after any change to
the scripts:

1. Insert a known row, e.g. `slug=backup-test-a`.
2. `linkhub-backup backup` -> note Backup ID A.
3. Delete `backup-test-a`, insert `backup-test-b`.
4. `linkhub-backup restore <A>` -> confirm the restore database has
   `backup-test-a` and *not* `backup-test-b`, while production still has
   the opposite (proves restore did not touch production).
5. `linkhub-backup cutover <restore-db>` -> confirm the app now serves
   `backup-test-a`.
6. `linkhub-backup rollback <pre-restore-db>` -> confirm the app is back to
   serving `backup-test-b`.

`ops/backup/test/run-integration-test.sh` automates exactly this drill
(plus the fail-closed paths: checksum tampering, wrong cutover
confirmation, duplicate restore, unknown backup id) against a throwaway
PostgreSQL and a local MinIO instance standing in for R2 -- it never
touches real production data or a real R2 bucket:

```bash
cd ops/backup/test
./run-integration-test.sh
```

## Disaster Recovery Runbook

Scenario: the original VPS, its Docker volumes, and the PostgreSQL data
directory are all gone. All you have is the GitHub repository, the
Cloudflare R2 backups, and your secrets (PostgreSQL credentials, R2
credentials) recovered from wherever you store secrets separately from the
server (password manager, sealed note, etc). If you need anything beyond
that to recover, treat it as a gap in this runbook and fix the runbook.

1. **Provision a new host** and install Git, Docker, and the Docker Compose
   plugin.

2. **Clone the repository:**

   ```bash
   git clone <this-repo-url> linkhub
   cd linkhub/backend
   ```

3. **Recreate secrets:**

   ```bash
   cp .env.example .env
   $EDITOR .env               # POSTGRES_USER/PASSWORD/DB, ALLOW_ORIGINS, REDIRECT_DOMAIN, ...
   cp .env.backup.example .env.backup
   $EDITOR .env.backup         # R2_ENDPOINT/BUCKET/ACCESS_KEY_ID/SECRET_ACCESS_KEY
   ```

4. **Start PostgreSQL only** (not the API yet -- there is no data to serve):

   ```bash
   docker compose up -d database
   ```

   Wait for it to become healthy: `docker compose ps`.

5. **List available backups:**

   ```bash
   ../ops/backup/linkhub-backup list
   ```

6. **Restore the most recent (or a specific) backup:**

   ```bash
   ../ops/backup/linkhub-backup restore <backup-id>
   ```

   This creates `<POSTGRES_DB>_restore_<timestamp>`. Note the name it
   prints.

7. **Promote it to production.** Since `api`/`nginx` aren't running yet,
   you can do this directly against the database, but using the same
   `cutover` path keeps the procedure identical to the normal one and is
   recommended:

   ```bash
   ../ops/backup/linkhub-backup cutover <restore-db-name>
   ```

   Type the production database name when prompted.

8. **Start the full stack:**

   ```bash
   docker compose up -d
   ```

9. **Verify:**
   - `curl http://localhost:8001/...` against a known API route (adjust to
     whatever health/list endpoint the API exposes).
   - `curl -I http://localhost:8002/<known-slug>` and confirm the expected
     redirect.
   - Spot-check row counts and a known slug directly:

     ```bash
     docker compose exec database psql -U <user> -d <db> -c "SELECT COUNT(*) FROM links;"
     docker compose exec database psql -U <user> -d <db> -c "SELECT * FROM links WHERE slug = '<known-slug>';"
     ```

10. **Point DNS/reverse proxy at the new host** (outside the scope of this
    repository).

If any of the above step requires information that isn't in this repo, in
R2, or in your secrets store, that's a disaster-recovery gap -- fix the gap,
not just this one recovery.

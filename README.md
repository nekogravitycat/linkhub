# LinkHub

LinkHub is a modern, high-performance URL shortener service featuring a Go (Golang) backend and a Vue 3 admin dashboard.

## Features

- **High Performance**: Backend written in Go for speed and efficiency.
- **Modern Admin UI**: Built with Vue 3, TypeScript, and Tailwind CSS.
- **Dockerized Backend**: Easy deployment using Docker Compose.
- **PostgreSQL**: Reliable data storage.
- **Analytics**: (Planned/In-progress) Track link usage.

## Architecture

- **Backend**: Go (Gin), PostgreSQL, Nginx.
- **Frontend**: Vue 3, Vite, Pinia, Tailwind CSS.

## Deployment

### Dependencies

- Docker & Docker Compose
- Node.js (for building the frontend)

### Backend Service

The backend services (API, Database, Nginx) are containerized.

1.  Navigate to the `backend` directory:

    ```bash
    cd backend
    ```

2.  Start the services:
    ```bash
    docker compose up -d
    ```

This will expose:

- **API**: `http://localhost:8001`
- **Redirection Service**: `http://localhost:8002`

### Frontend Admin Dashboard

To build and deploy the frontend:

1.  Navigate to the `frontend-admin` directory:

    ```bash
    cd frontend-admin
    ```

2.  Install dependencies:

    ```bash
    npm install
    ```

3.  Build the project:

    ```bash
    npm run build
    ```

4.  Serve the `dist` directory using your preferred web server (e.g., Nginx, Apache, Caddy, or a simple static file server).

    Example using `serve`:

    ```bash
    npx serve -s dist
    ```

## Backup & Restore

LinkHub ships with a daily PostgreSQL backup/restore toolset that uploads
encrypted-in-transit, private backups to Cloudflare R2, and supports safe
restore, production cutover, and rollback -- all from the CLI, with no
changes required to the API or frontend.

Full details (R2 setup, systemd scheduling, disaster recovery runbook) live
in [`ops/backup/README.md`](ops/backup/README.md). The quick version:

1.  Make sure `backend/.env` exists (copy `backend/.env.example` if not).

2.  Create a **private** Cloudflare R2 bucket (e.g. `linkhub-backups`) and an
    API token scoped to **Object Read & Write** on that bucket only -- not a
    Global API Key.

3.  Copy the backup env template and fill in your R2 credentials:

    ```bash
    cd backend
    cp .env.backup.example .env.backup
    ```

    ```dotenv
    R2_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
    R2_BUCKET=linkhub-backups
    R2_ACCESS_KEY_ID=<your-access-key-id>
    R2_SECRET_ACCESS_KEY=<your-secret-access-key>
    BACKUP_PREFIX=postgres
    BACKUP_RETENTION_DAYS=30
    ```

    `backend/.env.backup` is git-ignored and must never be committed.

4.  (Recommended) In the Cloudflare dashboard, add a **Lifecycle Rule** on
    the bucket to expire objects after `BACKUP_RETENTION_DAYS`, and consider
    enabling **Object Lock** with a 7-day retention window so recent
    backups can't be deleted or overwritten even by mistake.

5.  From the repository root, take your first backup:

    ```bash
    ./ops/backup/linkhub-backup backup
    ```

6.  List backups, restore one for inspection, or promote/roll back a
    restore:

    ```bash
    ./ops/backup/linkhub-backup list
    ./ops/backup/linkhub-backup restore <backup-id>
    ./ops/backup/linkhub-backup cutover <restore-db-name>
    ./ops/backup/linkhub-backup rollback <pre-restore-db-name>
    ```

    `restore` only ever creates a new, separate database -- it never
    touches production. `cutover` and `rollback` are destructive and each
    require typing the production database name to confirm.

7.  Schedule daily backups on the production host with systemd (see
    `ops/backup/systemd/` and the full instructions in
    `ops/backup/README.md`):

    ```bash
    sudo cp ops/backup/systemd/linkhub-backup.{service,timer} /etc/systemd/system/
    sudo $EDITOR /etc/systemd/system/linkhub-backup.service   # set WorkingDirectory
    sudo systemctl daemon-reload
    sudo systemctl enable --now linkhub-backup.timer
    ```

If you ever need to rebuild LinkHub from scratch (lost server, lost disk),
`ops/backup/README.md` has a full step-by-step Disaster Recovery runbook
that only assumes you have this repository, your R2 backups, and your
secrets.

## Security & Access Control

### API (`http://localhost:8001`)

The backend API **does not have built-in authentication**. Anyone with access to port 8001 can create, edit, or delete links.

> ⚠️ **You MUST implement an access control layer** if you expose the API to the internet.

**Recommended Solutions:**

- **Cloudflare Access / Zero Trust**: Put the API domain behind Cloudflare Access.
- **OAuth2 Proxy**: Run an OAuth2 proxy (Google, GitHub, login) in front of the API container.
- **Basic Auth**: Configure Basic Auth in Nginx or Traefik.
- **VPN / Private Network**: Only access the API via a secure tunnel (Tailscale, WireGuard).

### Frontend Admin

The `frontend-admin` is a static Single Page Application (SPA).

- **Exposure**: If you have correctly secured the API (as described above), it is **technically safe** to expose the frontend static files to the public internet, as all sensitive operations require API access.
- **Recommendation**: Despite being technically safe, it is **still not recommended** to expose the admin dashboard publically. It is best practice to keep the admin UI behind the same access control layer as your API to prevent confusion and reduce the attack surface.

## Configuration

Detailed configuration for development is available in the `README.md` files within the `backend/` and `frontend-admin/` subdirectories.

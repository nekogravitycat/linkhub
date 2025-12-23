# LinkHub

LinkHub is a modern, high-performance URL shortener service featuring a Go (Golang) backend and a Vue 3 admin dashboard.

## Features

- **High Performance**: Backend written in Go for speed and efficiency.
- **Modern Admin UI**: Built with Vue 3, TypeScript, and Tailwind CSS.
- **Dockerized Backend**: Easy deployment using Docker Compose.
- **PostgreSQL**: Reliable data storage.
- **Analytics**: (Planned/In-progress) Track link usage.

## Architecture

- **Backend**: Go (Gin/Chi or similar standard library based), PostgreSQL, Nginx.
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

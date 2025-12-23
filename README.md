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

> [!WARNING] > **Security Warning**: The `frontend-admin` application currently has **NO AUTHENTICATION OR PROTECTION**. It allows full control over the link database.
>
> **DO NOT EXPOSE THIS TO THE PUBLIC INTERNET.**
>
> Deploy this only behind a trusted VPN, secure internal network, or protected by an external authentication layer (e.g., Basic Auth, OAuth2 Proxy, Cloudflare Access).

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

## Configuration

Detailed configuration for development is available in the `README.md` files within the `backend/` and `frontend-admin/` subdirectories.

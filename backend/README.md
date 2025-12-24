# LinkHub Backend

LinkHub is a powerful and scalable link shortener service. This repository contains the backend service written in Go, which handles link management, redirection, and statistics.

## Architecture

The project follows a **Clean Architecture** pattern to ensure separation of concerns and maintainability.

- **Handler Layer** (`internal/links/http`): Handles HTTP requests, validation, and responses. Uses **Gin** framework.
- **Service Layer** (`internal/links/service.go`): Contains business logic.
- **Repository Layer** (`internal/links/repository.go`): Handles database interactions using **pgx/v5** and **Squirrel** query builder.

### Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin)
- **Database**: PostgreSQL
- **Driver**: [pgx/v5](https://github.com/jackc/pgx) (with pgxpool)
- **Reverse Proxy**: Nginx
- **Containerization**: Docker & Docker Compose

## Project Structure

```
├── cmd/
│   └── server/          # Entry point of the application
├── internal/
│   ├── api/             # Router setup and global middleware
│   ├── config/          # Configuration loading
│   ├── database/        # Database connection setup
│   └── links/           # Link domain (Handler, Service, Repository, DTOs)
├── nginx/               # Nginx configuration templates
├── database/            # Database initialization scripts and schema
├── tests/               # Integration tests
└── docker-compose.yml   # Docker Compose definition
```

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- [Go 1.25+](https://go.dev/) (only for local development without Docker)

### Running with Docker (Recommended)

1. **Clone the repository:**

   ```bash
   git clone <repository-url>
   cd linkhub/backend
   ```

2. **Configure Environment:**
   Copy the example environment file:

   ```bash
   cp .env.example .env
   ```

   Modify `.env` if necessary. By default, it works out-of-the-box with Docker Compose.

3. **Start the Services:**

   ```bash
   docker-compose up -d --build
   ```

   This will start:

   - **PostgreSQL Database**
   - **Backend Service**
   - **Nginx Reverse Proxy**

4. **Access the Application:**
   - **API**: [http://localhost:8001](http://localhost:8001)
   - **Short Link Redirection**: [http://localhost:8002](http://localhost:8002) (e.g., `http://localhost:8002/my-slug`)

### Local Development

If you want to run the Go application locally (outside Docker) while keeping the database in Docker:

1. **Start the Database:**

   ```bash
   docker-compose up -d database
   ```

2. **Set Environment Variables:**
   Ensure your `.env` connects to the exposed Postgres port (usually 5432). You might need to set `POSTGRES_ADDR=localhost` if it's running in Docker mapped to localhost.

3. **Run the Application:**

   ```bash
   go run cmd/server/main.go
   ```

4. **Run Tests:**
   The project includes integration tests.
   ```bash
   go test ./tests/...
   ```

## Nginx Configuration

Nginx acts as the entry point and handles routing based on ports (or domains in production).

- **Port 8001**: Proxies requests to the backend API.
- **Port 8002**: Handles redirection. It rewrites `/{slug}` to `/redirect/{slug}` and forwards it to the backend.

## Database

The database schema is automatically initialized using the scripts in the `database/` directory when the Postgres container starts for the first time.

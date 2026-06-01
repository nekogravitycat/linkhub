# linkhub stress test

A self-contained load-testing tool for the linkhub backend. It measures the
**throughput (req/s)** and **latency distribution** of each API operation:

| scenario   | endpoint                  | what it measures                          |
|------------|---------------------------|-------------------------------------------|
| `create`   | `POST /links`             | link creation (unique slug + valid URL)   |
| `get`      | `GET /links/:slug`        | single-link lookup                        |
| `list`     | `GET /links`              | listing with pagination/sort/keyword      |
| `update`   | `PATCH /links/:slug`      | link edition                              |
| `redirect` | `GET /redirect/:slug`     | link shortening / redirect (302)          |
| `delete`   | `DELETE /links/:slug`     | link deletion                             |
| `mixed`    | weighted blend            | realistic traffic (read-heavy)            |

Pure Go, **standard library only** — no install step, no external dependencies.

## Requirements

- Go 1.25+
- A running linkhub backend reachable at `--url` (and its PostgreSQL).

The tool talks only HTTP; it never touches the database directly.

## Quick start

Start the backend directly (bypasses nginx rate limits, so you measure the Go
server itself). From `backend/`:

```bash
docker compose up -d database
APP_ENV=development PPROF_ADDR=localhost:6060 go run cmd/server/main.go
```

Then, from `tools/stress/`:

```bash
# Run every scenario for 15s at concurrency 50, seeding 2000 links first
go run . --scenario all -c 50 -d 15s --seed 2000

# Single operation, fixed request count
go run . --scenario create -n 50000 -c 100

# Just redirects (the "link shortening" hot path)
go run . --scenario redirect -c 200 -d 30s --seed 5000

# Open-loop: hold a target rate and watch latency
go run . --scenario mixed --rate 500 -d 60s --seed 10000
```

## Flags

| flag                | default                 | meaning                                                            |
|---------------------|-------------------------|--------------------------------------------------------------------|
| `--url`             | `http://localhost:8001` | backend base URL (point at nginx ports to include its limits)      |
| `--scenario`        | `all`                   | `all` or one of the scenarios above                                |
| `-c, --concurrency` | `50`                    | number of concurrent workers                                       |
| `-d, --duration`    | `30s`                   | run length per scenario (ignored when `--requests` set)            |
| `-n, --requests`    | `0`                     | total requests per scenario (`0` = use duration)                   |
| `--rate`            | `0`                     | target req/s, open-loop (`0` = closed-loop, as fast as possible)   |
| `--seed`            | `10000`                 | links to pre-create for read/update/redirect/delete               |
| `--warmup`          | `2s`                    | discard metrics for this window (duration mode only)               |
| `--timeout`         | `10s`                   | per-request timeout                                                |
| `--conns`           | `0`                     | max idle conns/host (`0` = 2×concurrency)                          |
| `--keepalive`       | `true`                  | HTTP keep-alive (set `false` to measure cold-connection cost)      |
| `--redirect-domain` | `localhost:8003`        | excluded from generated URLs (match backend `REDIRECT_DOMAIN`)     |
| `--out`             | _(none)_                | write per-scenario results as JSON                                 |
| `--csv`             | _(none)_                | write per-scenario results as CSV                                  |
| `--no-cleanup`      | `false`                 | leave seeded links in the database                                 |

## How it works

- **Seeding.** Scenarios that read existing links (`get`, `list`, `update`,
  `redirect`, `mixed`) need data first, so the tool concurrently creates
  `--seed` links before running. `delete` consumes its own dedicated pool.
- **Unique slugs.** Every run uses a unique prefix (`s<timestamp>`); slugs are
  `<prefix>-<counter>` in base36, so they're always unique and within the
  32-char / `^[a-zA-Z0-9-_]+$` constraints — no 409s from collisions.
- **Redirects.** The client does **not** follow redirects; a `302` is the
  success signal for the `redirect` scenario.
- **`delete` is pool-bounded.** It performs exactly as many deletes as there are
  pre-seeded links (`--seed`, or `--requests` if set), ignoring `--duration`,
  so every delete targets a real row and there are no spurious 404s.
- **`update` keeps links active** (`is_active: true`) so they remain valid for
  the `redirect` scenario when running `all`.
- **Metrics** are collected per-worker (lock-free on the hot path) into a
  log-bucketed latency histogram, then merged. Warmup-window requests are
  excluded. Throughput = successful requests ÷ measured window.

### "all" mode ordering

Scenarios run as: `create → get → list → redirect → update → mixed → delete`.
`redirect`/`get` run while the seeded pool is pristine, and `delete` runs last
because it consumes its own pool.

## Reading the output

```
redirect  |  15.0s | 142,330 req | 9,488.6 req/s | err 0 (0.00%)
  latency  min 0.3ms  mean 5.1ms  p50 4.2ms  p90 9ms  p95 12ms  p99 28ms  p99.9 61ms  max 140ms
  status   302:142330
```

When more than one scenario runs, a comparison table (req/s, p50, p99, err%) is
printed at the end. Use `--out results.json` to track numbers across runs.

## Profiling the backend (pprof)

The backend exposes Go pprof on a separate port when `PPROF_ADDR` is set (off by
default; dev only). With the backend started as in *Quick start*:

```bash
# 30s CPU profile while a load run is in progress
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# heap / goroutines
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Caveats

- **nginx rate limits.** The production path is fronted by nginx, which limits
  the API to ~2 req/s and redirects to ~10 req/s. The default `--url`
  (`http://localhost:8001`, the Go server directly) bypasses these so you see
  raw backend throughput. Point `--url` at the nginx-exposed ports to measure
  the production path including its limits — throughput will be capped
  accordingly (this is expected).
- **Leftover data.** Links created by `create`/`mixed` are *not* auto-deleted
  (their count is unbounded). The run prints its prefix; purge with
  `DELETE FROM links WHERE slug LIKE '<prefix>%';` or `TRUNCATE links;`.
  Seeded read-pool links are cleaned up automatically unless `--no-cleanup`.
- Run against a **disposable / local** database, not production data.

# Jobflow

A distributed async job processing backend built in Go. Supports scheduled job dispatch, real-time status streaming, and multi-consumer parallel processing.

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP Framework | Gin |
| ORM | GORM |
| Database | PostgreSQL |
| Queue | RabbitMQ (Delayed Message Plugin) |
| Cache / Pub-Sub | Redis |
| Auth | JWT (access + refresh) + Bcrypt |
| Real-time | Server-Sent Events (SSE) |
| Containers | Docker + Docker Compose |

## Architecture

```
Client
  │
  ▼
┌─────────────────────────────────────────┐
│               Server (Gin)              │
│  Auth · Jobs CRUD · SSE endpoint        │
│  Rate limiter (token bucket)            │
└────────────────┬────────────────────────┘
                 │ PublishDelayed(delay)
                 ▼
┌─────────────────────────────────────────┐
│    RabbitMQ  —  jobs.delayed exchange   │
│    (x-delayed-message plugin)           │
│    holds message until scheduled_at     │
└────────────────┬────────────────────────┘
                 │ delivers after delay
                 ▼
┌─────────────────────────────────────────┐
│         Worker  (5 consumers)           │
│  each consumer owns its own channel     │
│  sendEmail · reportGeneration           │
│  resizeImage · exportCSV                │
│                                         │
│  on status change → Redis PUBLISH       │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│  Redis Pub/Sub subscriber (goroutine)   │
│  fan-out to connected SSE clients       │
└─────────────────────────────────────────┘
```

## Key Design Decisions

**Scheduled jobs via RabbitMQ Delayed Message Plugin**
Jobs with a future `scheduled_at` are published immediately to a `x-delayed-message` exchange with an `x-delay` header in milliseconds. The broker holds the message internally and routes it to the `jobs` queue once the delay expires — no polling, no scheduler process.

**5 parallel consumers, each with its own AMQP channel**
`Qos(1, 0, false)` per consumer ensures fair dispatch. Using a separate channel per consumer avoids race conditions on `Ack`/`Nack` that would occur on a shared channel.

**Token bucket rate limiter (drop, not queue)**
Requests that exceed capacity are rejected immediately with `429`. A queue-based approach would pile goroutines in memory under sustained load.

**SSE over WebSocket**
Job status updates are one-directional (server → client). SSE is simpler, works over HTTP/1.1, and needs no upgrade handshake. Redis pub/sub decouples the worker from connected clients.

**JWT + JTI revocation in Redis**
Access tokens carry a `jti` claim stored in Redis with the token's TTL. On logout, the JTI is deleted — tokens are effectively invalidated without a DB hit on every request.

## API

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register a new user |
| POST | `/auth/login` | Login, returns access + refresh token |
| POST | `/auth/refresh` | Rotate refresh token |
| POST | `/auth/logout` | Revoke current session |

### Jobs
| Method | Path | Description |
|--------|------|-------------|
| POST | `/jobs` | Create a single job |
| POST | `/jobs/bulk` | Create multiple jobs |
| GET | `/jobs` | List all jobs for the authenticated user |
| GET | `/jobs/:id` | Get a job by ID |
| PUT | `/jobs/:id` | Update a job |
| DELETE | `/jobs/:id` | Delete a job |

### SSE
| Method | Path | Description |
|--------|------|-------------|
| GET | `/sse/jobs` | Stream real-time job status updates |

### Health
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Check Postgres, Redis, RabbitMQ status |

## Job Types

| Type | Behaviour |
|------|-----------|
| `send_email` | 2s simulated processing |
| `report_generation` | 3s simulated processing |
| `resize_image` | 2s simulated processing |
| `export_csv` | Always fails (intentional — demonstrates error path) |

## Running Locally

**Prerequisites:** Docker, Docker Compose

```bash
# 1. Clone and enter the repo
git clone https://github.com/saravana-devx/jobflow
cd jobflow

# 2. Copy env config
cp .env.example .env

# 3. Build and start all services
docker compose up --build

# Server →  http://localhost:8080
# RabbitMQ management → http://localhost:15672  (guest / guest)
```

> First run builds a custom RabbitMQ image with the
> `rabbitmq_delayed_message_exchange` plugin pre-installed.

## Project Structure

```
cmd/
  server/       → HTTP server entry point
  worker/       → Job consumer entry point
internal/
  auth/         → JWT auth, token store, JTI revocation
  bootstrap/    → Dependency wiring
  config/       → Env-based config
  cron/         → Refresh token cleanup + job reconciler (dual-write recovery)
  database/     → PostgreSQL connection
  email/        → Email module scaffold (provider integration TODO)
  health/       → Health check handler
  jobs/         → Job model, service, handler, repository
  middleware/   → Auth + rate limit middleware
  rabbitmq/     → Connection, publisher, consumer, queue declarations
  ratelimit/    → Token bucket implementation
  redis/        → Redis client, pub/sub
  routes/       → Route registration
  sse/          → SSE client manager, handler, subscriber
  worker/       → Job handlers, worker repository
pkg/
  logger/       → Leveled logging wrapper (swap-in point for slog/zerolog)
```

## Known Limitations / Future Work

**No dead-letter queue.** `internal/rabbitmq/consume.go` correctly `Nack`s a
message with `requeue=false` once a job exhausts `MaxRetries`, but none of the
queues in `internal/rabbitmq/queues.go` set `DLXName` (the field already exists
on `QueueConfig`). With no dead-letter exchange bound, RabbitMQ simply drops
these messages — a job that fails every retry disappears with nothing but a log
line, and there's no way to inspect or replay it. To close this gap: configure
`DLXName` on `QueueJobs`, declare a `jobs.dlq` queue bound to that exchange, and
add monitoring/an admin endpoint to inspect and optionally replay dead-lettered
jobs.

**Job handlers are simulated.** `internal/worker/handler.go`'s `handleSendEmail`,
`handleReportGeneration`, `handleResizeImage`, and `handleExportCSV` use
`time.Sleep` as a stand-in for real work. To make these real, replace each
`handleXxx` body with the actual integration (SMTP/email provider, report
renderer, image processing library, CSV writer to object storage), and consider
moving from the current `switch job.Type` in `handleJob` to a
`map[JobType]func(*Job) error` registry so adding a new job type doesn't require
editing the dispatch switch.

**Observability is minimal.** ClickHouse is listed as the planned log/event
store, but it is not wired up yet, and there is currently **no structured
logging and no metrics**. Logging is plain `log.Printf` (unleveled, unstructured
text). A `pkg/logger` leveled wrapper now exists as the seam to fix this: today
it formats to stdlib `log`, but because every call site can route through it,
swapping the backend to structured JSON (`log/slog`, zerolog, or zap) is a
one-file change. Next steps for production readiness: adopt `pkg/logger`
everywhere, emit request/trace IDs, add Prometheus metrics (queue depth, job
latency, retry/DLQ counts), and ship job lifecycle events to ClickHouse for
historical analytics.

**Rate limiting is global, not per-user.** `internal/ratelimit` is a single
token bucket shared by every request (`middleware.RateLimit` applies one
`*TokenBucket` to the whole app). That protects the service as a whole but lets
one noisy client consume everyone's budget. To make it per-user: key buckets by
identity — `map[userID]*TokenBucket` (with eviction of idle entries), or extract
the user/IP from the request and look up its bucket in the middleware. For a
horizontally-scaled deployment (multiple server instances), an in-memory map
isn't shared across instances, so move the counter to **Redis** (e.g.
`INCR` + `EXPIRE` per key, or a Redis-backed token/leaky bucket) so the limit is
enforced cluster-wide.

## Testing

```bash
# Unit tests (no external services needed)
go test ./...

# Integration tests (require Postgres + RabbitMQ from docker-compose)
go test -tags=integration ./...
```

- `internal/ratelimit/token_bucket_test.go` — table-driven unit tests for the
  token bucket (initial capacity, refill, capacity cap).
- `internal/jobs/service_integration_test.go` — integration test scaffolding for
  job creation, gated behind the `integration` build tag.

## Environment Variables

See `.env.example` for all required variables.

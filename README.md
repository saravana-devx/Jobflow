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
  cron/         → Refresh token cleanup job
  database/     → PostgreSQL connection
  health/       → Health check handler
  jobs/         → Job model, service, handler, repository
  middleware/   → Auth + rate limit middleware
  rabbitmq/     → Connection, publisher, consumer, queue declarations
  ratelimit/    → Token bucket implementation
  redis/        → Redis client, pub/sub
  routes/       → Route registration
  sse/          → SSE client manager, handler, subscriber
  worker/       → Job handlers, worker repository
```

## Environment Variables

See `.env.example` for all required variables.

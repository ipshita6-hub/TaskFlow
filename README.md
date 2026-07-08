# TaskFlow

A production-ready backend task scheduling system built with Go, PostgreSQL, and concurrent worker pools. Execute one-time and recurring tasks automatically with REST APIs, JWT authentication, retry mechanisms, and detailed execution logging.

## Features

- **User Authentication**: Secure JWT-based auth with bcrypt password hashing
- **Task Management**: Create, update, delete, and list scheduled tasks
- **Cron Scheduling**: Support for 5-field POSIX cron expressions and one-time tasks
- **Concurrent Execution**: Go goroutine-based worker pool with atomic job claiming
- **Retry Logic**: Configurable exponential backoff for failed tasks
- **Execution Logging**: Full audit trail of task runs with status, duration, and errors
- **Dead Letter Queue**: Track exhausted jobs for manual intervention
- **Worker Registry**: Monitor active workers and their heartbeats
- **REST API**: Clean, documented API with OpenAPI 3.0 spec
- **Graceful Shutdown**: Signal handling with proper resource cleanup

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Docker (optional, for containerized PostgreSQL)

### Setup

1. **Clone and navigate to project**:
   ```bash
   git clone https://github.com/yourusername/taskflow.git
   cd taskflow
   ```

2. **Start PostgreSQL** (Docker):
   ```bash
   docker-compose up -d
   ```

3. **Configure environment** (optional):
   ```bash
   # Defaults are suitable for local development
   export DATABASE_URL="postgres://taskflow:taskflow@localhost:5432/taskflow"
   export JWT_SECRET="your-secret-key"
   export WORKER_CONCURRENCY=5
   ```

4. **Build and run**:
   ```bash
   go build -o bin/server ./cmd/server
   ./bin/server
   ```

   The server will:
   - Run migrations automatically
   - Start the scheduler and worker pool
   - Listen on `http://localhost:8080`

## API Documentation

Once running, visit **`http://localhost:8080/api/docs`** for the OpenAPI specification.

### Core Endpoints

**Authentication**:
- `POST /api/v1/auth/register` — Create account
- `POST /api/v1/auth/login` — Get JWT token

**Tasks**:
- `POST /api/v1/tasks` — Create task
- `GET /api/v1/tasks` — List your tasks
- `GET /api/v1/tasks/{id}` — Get task details
- `PATCH /api/v1/tasks/{id}` — Update task
- `DELETE /api/v1/tasks/{id}` — Delete task

**Execution & Monitoring**:
- `GET /api/v1/tasks/{id}/logs` — Task execution history
- `GET /api/v1/jobs/dlq` — Failed/exhausted jobs
- `GET /api/v1/workers` — Active worker processes
- `GET /api/v1/schedule/validate` — Validate cron expressions

## Configuration

Environment variables (with defaults):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `JWT_SECRET` | *(required)* | Secret key for signing JWTs |
| `JWT_EXPIRY_HOURS` | `24` | JWT token expiration |
| `WORKER_CONCURRENCY` | `5` | Number of concurrent task workers |
| `SCHEDULER_TICK_MS` | `5000` | Scheduler polling interval (ms) |
| `HEARTBEAT_INTERVAL_S` | `30` | Worker heartbeat interval (s) |
| `STALE_JOB_THRESHOLD_S` | `90` | Mark job stale after (s) |
| `LOG_RETENTION_DAYS` | `30` | Retention for execution logs |
| `SERVER_PORT` | `8080` | HTTP server port |

## Development

### Project Structure

```
taskflow/
├── cmd/server/          # Entry point
├── internal/
│   ├── api/             # HTTP routing and middleware
│   ├── auth/            # Authentication service
│   ├── config/          # Configuration loader
│   ├── db/              # Database and migrations
│   ├── handler/         # Task handler registry
│   ├── job/             # Job repository and service
│   ├── models/          # Domain models and DTOs
│   ├── scheduler/       # Cron scheduler
│   ├── task/            # Task repository and service
│   └── worker/          # Worker pool and heartbeat
├── pkg/validate/        # Schedule validation
├── test/property/       # Property-based tests (optional)
├── docker-compose.yml   # Local PostgreSQL setup
├── go.mod              # Go module definition
└── Makefile            # Build targets
```

### Build Targets

```bash
make build          # Compile the server
make run            # Build and run
make test           # Run tests
make migrate-up     # Run database migrations
make migrate-down   # Rollback migrations
make lint           # Run linter
```

### Running Tests

```bash
go test ./...
```

## Architecture

### Concurrency Model

- **Single-process server** with goroutine-based workers
- **Atomic job claiming** using PostgreSQL `SELECT FOR UPDATE SKIP LOCKED`
- **Scheduler loop** periodically enqueues due tasks
- **Worker pool** concurrently executes claimed jobs
- **Heartbeat mechanism** detects stale workers and reclaims jobs

### Retry Strategy

- **Exponential backoff**: `delay = backoff_seconds × attempt`
- **Configurable per task**: Set `max_attempts` and `backoff_seconds` on creation
- **Failed jobs tracked** in dead letter queue for manual review

### Data Model

**Tasks** store the schedule definition (cron or one-time datetime).
**Jobs** are execution instances created by the scheduler.
**ExecutionLogs** record each job's outcome (success, failure, duration).
**Workers** register themselves and send heartbeats to stay alive.

## Deployment

### Docker (Future)

A `Dockerfile` can be added for containerized deployment. The server is stateless and scales horizontally with load balancing.

### Database

Migrations are applied automatically on startup. Ensure PostgreSQL is accessible at `DATABASE_URL`.

### Environment

Use a secret manager (e.g., AWS Secrets Manager, HashiCorp Vault) to store `JWT_SECRET` in production.

## Contributing

Contributions are welcome. Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -am 'Add feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a pull request

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or suggestions, please open an issue on [GitHub Issues](https://github.com/yourusername/taskflow/issues).

## Roadmap

- [ ] Distributed deployment (multiple scheduler instances with leader election)
- [ ] UI dashboard for task monitoring
- [ ] Webhook callbacks for task completion
- [ ] gRPC API for high-performance clients
- [ ] Prometheus metrics endpoint
- [ ] Rate limiting and quota management

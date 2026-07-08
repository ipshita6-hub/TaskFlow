# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-08

### Added
- Initial release of TaskFlow
- User authentication with JWT and bcrypt password hashing
- Task management (CRUD operations) for one-time and recurring tasks
- Cron-based scheduling with 5-field POSIX expressions
- Concurrent worker pool with atomic job claiming
- Retry mechanism with exponential backoff
- Execution logging and audit trail
- Dead letter queue for failed/exhausted jobs
- Worker registry with heartbeat tracking
- OpenAPI 3.0 specification
- REST API with 11 endpoints
- PostgreSQL-backed persistence
- Graceful shutdown with signal handling
- Docker Compose setup for local development
- GitHub Actions CI/CD workflow
- Comprehensive documentation and examples

### Architecture
- Single-process server with goroutine-based concurrency
- Atomic job claiming using `SELECT FOR UPDATE SKIP LOCKED`
- Scheduler loop for task enqueueing
- Worker pool for concurrent execution
- Health monitoring via heartbeats

### Performance
- Connection pooling: 25 max open, 5 idle connections
- Worker pool scalability via `WORKER_CONCURRENCY` config
- Efficient cron expression parsing and validation
- Indexed database queries for fast lookups

## Future Roadmap

### [2.0.0] - Planned
- Distributed scheduler with leader election
- Web UI dashboard
- Webhook callbacks
- gRPC API
- Prometheus metrics
- Rate limiting and quotas
- Task dependencies and workflows
- Bulk operations API
- Task templates and libraries

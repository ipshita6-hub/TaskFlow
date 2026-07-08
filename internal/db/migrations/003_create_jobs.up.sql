CREATE TABLE jobs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id           UUID NOT NULL REFERENCES tasks(id),
    user_id           UUID NOT NULL REFERENCES users(id),
    status            TEXT NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'running', 'completed', 'failed', 'exhausted')),
    attempt           INT NOT NULL DEFAULT 1,
    enqueued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    run_after         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    worker_id         UUID,
    last_heartbeat_at TIMESTAMPTZ,
    error_message     TEXT
);
CREATE INDEX idx_jobs_claim ON jobs(status, run_after, enqueued_at) WHERE status = 'queued';
CREATE INDEX idx_jobs_stale ON jobs(last_heartbeat_at) WHERE status = 'running';
CREATE INDEX idx_jobs_task  ON jobs(task_id);

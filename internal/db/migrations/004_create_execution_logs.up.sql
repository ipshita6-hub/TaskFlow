CREATE TABLE execution_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID NOT NULL REFERENCES jobs(id),
    task_id       UUID NOT NULL REFERENCES tasks(id),
    user_id       UUID NOT NULL REFERENCES users(id),
    attempt       INT NOT NULL,
    status        TEXT NOT NULL CHECK (status IN ('completed', 'failed')),
    started_at    TIMESTAMPTZ NOT NULL,
    ended_at      TIMESTAMPTZ NOT NULL,
    duration_ms   BIGINT NOT NULL,
    output        TEXT,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_logs_task_time ON execution_logs(task_id, started_at DESC);
CREATE INDEX idx_logs_cleanup   ON execution_logs(created_at);

CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    task_type       TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    schedule_type   TEXT NOT NULL CHECK (schedule_type IN ('one_time', 'recurring')),
    scheduled_at    TIMESTAMPTZ,
    cron_expr       TEXT,
    next_run_at     TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'active', 'running', 'completed', 'failed', 'deleted')),
    max_attempts    INT NOT NULL DEFAULT 3 CHECK (max_attempts BETWEEN 1 AND 10),
    backoff_seconds INT NOT NULL DEFAULT 60 CHECK (backoff_seconds BETWEEN 1 AND 3600),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tasks_user_status ON tasks(user_id, status);
CREATE INDEX idx_tasks_next_run ON tasks(next_run_at) WHERE status = 'active';

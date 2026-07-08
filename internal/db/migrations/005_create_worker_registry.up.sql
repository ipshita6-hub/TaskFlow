CREATE TABLE worker_registry (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname          TEXT NOT NULL,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'stale'))
);

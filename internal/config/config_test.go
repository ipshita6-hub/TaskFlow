package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "supersecretkey")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/testdb")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/testdb")
	t.Setenv("JWT_SECRET", "supersecretkey")
	// Clear optional vars so defaults kick in.
	for _, key := range []string{
		"JWT_EXPIRY_HOURS", "WORKER_CONCURRENCY", "SCHEDULER_TICK_MS",
		"HEARTBEAT_INTERVAL_S", "STALE_JOB_THRESHOLD_S", "LOG_RETENTION_DAYS", "SERVER_PORT",
	} {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "postgres://localhost/testdb", cfg.DatabaseURL)
	assert.Equal(t, "supersecretkey", cfg.JWTSecret)
	assert.Equal(t, 24, cfg.JWTExpiryHours)
	assert.Equal(t, 5, cfg.WorkerConcurrency)
	assert.Equal(t, 1000, cfg.SchedulerTickMs)
	assert.Equal(t, 30, cfg.HeartbeatIntervalS)
	assert.Equal(t, 90, cfg.StaleJobThresholdS)
	assert.Equal(t, 90, cfg.LogRetentionDays)
	assert.Equal(t, "8080", cfg.ServerPort)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@host/db")
	t.Setenv("JWT_SECRET", "mysecret")
	t.Setenv("JWT_EXPIRY_HOURS", "48")
	t.Setenv("WORKER_CONCURRENCY", "10")
	t.Setenv("SCHEDULER_TICK_MS", "500")
	t.Setenv("HEARTBEAT_INTERVAL_S", "15")
	t.Setenv("STALE_JOB_THRESHOLD_S", "60")
	t.Setenv("LOG_RETENTION_DAYS", "30")
	t.Setenv("SERVER_PORT", "9090")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "postgres://user:pass@host/db", cfg.DatabaseURL)
	assert.Equal(t, "mysecret", cfg.JWTSecret)
	assert.Equal(t, 48, cfg.JWTExpiryHours)
	assert.Equal(t, 10, cfg.WorkerConcurrency)
	assert.Equal(t, 500, cfg.SchedulerTickMs)
	assert.Equal(t, 15, cfg.HeartbeatIntervalS)
	assert.Equal(t, 60, cfg.StaleJobThresholdS)
	assert.Equal(t, 30, cfg.LogRetentionDays)
	assert.Equal(t, "9090", cfg.ServerPort)
}

func TestLoad_InvalidIntegerFallsBackToDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/testdb")
	t.Setenv("JWT_SECRET", "supersecretkey")
	t.Setenv("JWT_EXPIRY_HOURS", "not-a-number")
	t.Setenv("WORKER_CONCURRENCY", "abc")

	cfg, err := Load()
	require.NoError(t, err)

	// Invalid integers should fall back to defaults.
	assert.Equal(t, 24, cfg.JWTExpiryHours)
	assert.Equal(t, 5, cfg.WorkerConcurrency)
}

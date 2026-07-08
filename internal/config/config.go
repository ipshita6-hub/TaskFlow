package config

import (
	"errors"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string (required).
	DatabaseURL string
	// JWTSecret is the HS256 signing key (required).
	JWTSecret string
	// JWTExpiryHours is the JWT token lifetime in hours (default 24).
	JWTExpiryHours int
	// WorkerConcurrency is the number of concurrent worker goroutines (default 5).
	WorkerConcurrency int
	// SchedulerTickMs is the scheduler poll interval in milliseconds (default 1000).
	SchedulerTickMs int
	// HeartbeatIntervalS is the worker heartbeat interval in seconds (default 30).
	HeartbeatIntervalS int
	// StaleJobThresholdS is the seconds before a running job is reclaimed (default 90).
	StaleJobThresholdS int
	// LogRetentionDays is the minimum days to retain execution logs (default 90).
	LogRetentionDays int
	// ServerPort is the HTTP server port (default "8080").
	ServerPort string
}

// Load reads environment variables, applies defaults for optional fields, and
// returns an error if any required variables are missing.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required but not set")
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required but not set")
	}

	cfg.JWTExpiryHours = envInt("JWT_EXPIRY_HOURS", 24)
	cfg.WorkerConcurrency = envInt("WORKER_CONCURRENCY", 5)
	cfg.SchedulerTickMs = envInt("SCHEDULER_TICK_MS", 1000)
	cfg.HeartbeatIntervalS = envInt("HEARTBEAT_INTERVAL_S", 30)
	cfg.StaleJobThresholdS = envInt("STALE_JOB_THRESHOLD_S", 90)
	cfg.LogRetentionDays = envInt("LOG_RETENTION_DAYS", 90)

	cfg.ServerPort = os.Getenv("SERVER_PORT")
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}

	return cfg, nil
}

// envInt reads an environment variable and parses it as an integer.
// If the variable is absent or not a valid integer, the provided default is returned.
func envInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return v
}

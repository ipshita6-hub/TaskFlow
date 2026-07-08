package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskflow/internal/api"
	"taskflow/internal/auth"
	"taskflow/internal/config"
	"taskflow/internal/db"
	"taskflow/internal/handler"
	jobpkg "taskflow/internal/job"
	"taskflow/internal/scheduler"
	"taskflow/internal/task"
	"taskflow/internal/worker"
	"taskflow/pkg/validate"
)

func main() {
	// ─── Setup Logging ────────────────────────────────────────────────────────

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ─── Load Configuration ───────────────────────────────────────────────────

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ─── Connect Database ────────────────────────────────────────────────────

	dbConn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// ─── Run Migrations ─────────────────────────────────────────────────────

	if err := db.RunMigrations(dbConn); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// ─── Instantiate Repositories ──────────────────────────────────────────

	userRepo := auth.NewPostgresUserRepository(dbConn)
	taskRepo := task.NewPostgresTaskRepository(dbConn)
	jobRepo := jobpkg.NewPostgresJobRepository(dbConn)
	logRepo := jobpkg.NewPostgresLogRepository(dbConn)
	workerRepo := worker.NewPostgresWorkerRepo(dbConn)

	// ─── Instantiate Services ──────────────────────────────────────────────

	authService := auth.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)

	cronParser := scheduler.NewCronParser()
	taskService := task.NewTaskService(taskRepo, cronParser)
	jobService := jobpkg.NewJobService(jobRepo, logRepo)

	// ─── Instantiate Handler Registry ──────────────────────────────────────

	handlerRegistry := handler.NewRegistry()
	// "noop" handler is pre-registered by NewRegistry()

	// ─── Instantiate Scheduler ─────────────────────────────────────────────

	sched := scheduler.NewScheduler(
		taskRepo,
		jobRepo,
		cronParser,
		time.Duration(cfg.SchedulerTickMs)*time.Millisecond,
		time.Duration(cfg.StaleJobThresholdS)*time.Second,
	)

	// ─── Instantiate Worker Pool ──────────────────────────────────────────

	workerPool := worker.NewPool(
		jobRepo,
		workerRepo,
		handlerRegistry,
		cfg.WorkerConcurrency,
		time.Duration(cfg.HeartbeatIntervalS)*time.Second,
	)

	// ─── Instantiate HTTP Handlers ────────────────────────────────────────

	authHandler := auth.NewHandler(authService, cfg.JWTExpiryHours)
	taskHandler := task.NewHandler(taskService)
	jobHandler := jobpkg.NewHandler(jobService)
	workerHandler := worker.NewHandler(workerRepo)
	validateHandler := validate.NewValidator(cronParser)

	// ─── Build HTTP Router ────────────────────────────────────────────────

	handlers := &api.Handlers{
		RegisterHandler:    authHandler.Register,
		LoginHandler:       authHandler.Login,
		CreateTaskHandler:  taskHandler.Create,
		ListTasksHandler:   taskHandler.List,
		GetTaskHandler:     taskHandler.Get,
		UpdateTaskHandler:  taskHandler.Update,
		DeleteTaskHandler:  taskHandler.Delete,
		GetLogsHandler:     jobHandler.GetLogs,
		GetDLQHandler:      jobHandler.GetDLQ,
		ListWorkersHandler: workerHandler.ListWorkers,
		ValidateHandler:    validateHandler.Handler,
	}

	router := api.NewRouter(auth.JWTMiddleware(authService), handlers)
	logger.Info("http router initialized")

	// ─── Context with Cancellation ────────────────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── Start Background Engine ───────────────────────────────────────────

	sched.Start(ctx)
	logger.Info("scheduler started", "tick_ms", cfg.SchedulerTickMs)

	workerPool.Start(ctx)
	logger.Info("worker pool started", "concurrency", cfg.WorkerConcurrency)

	// ─── Start HTTP Server ────────────────────────────────────────────────

	httpServer := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "port", cfg.ServerPort)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// ─── Graceful Shutdown ────────────────────────────────────────────────

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("signal received", "signal", sig)

		// Cancel context to stop scheduler and workers
		cancel()

		// Stop worker pool gracefully
		logger.Info("stopping worker pool")
		workerPool.Stop()

		// Shutdown HTTP server with timeout
		logger.Info("shutting down http server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", "error", err)
		}

		logger.Info("server stopped gracefully")
		os.Exit(0)

	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}
}

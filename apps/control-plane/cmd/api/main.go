package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/config"
	dockerexecutor "github.com/ghaem51/ephemeral/apps/control-plane/internal/executor/docker"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/server"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/storage/sqlite"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/createenvironment"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/environmentapi"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/environmentlifecycle"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	store, err := sqlite.Open(context.Background(), cfg.DatabasePath)
	if err != nil {
		logger.Error("database startup failed", "database_path", cfg.DatabasePath, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()
	recovered, err := store.RecoverStaleWorkflows(context.Background(), time.Now().UTC())
	if err != nil {
		logger.Error("recover stale workflows", "error", err)
		os.Exit(1)
	}
	for _, workflow := range recovered {
		logger.Warn("recovered stale workflow", "environment_id", workflow.EnvironmentID, "workflow_id", workflow.WorkflowID)
	}
	runtimeExecutor, err := dockerexecutor.NewFromEnv(dockerexecutor.Options{
		AllowedImages: cfg.DockerImages, HealthPath: cfg.HealthPath,
		HealthAttempts: cfg.HealthAttempts, HealthInterval: cfg.HealthInterval,
		HealthTimeout: cfg.HealthTimeout, StopTimeout: cfg.DockerStopTimeout,
	})
	if err != nil {
		logger.Error("configure Docker executor", "error", err)
		os.Exit(1)
	}
	dockerCtx, cancelDocker := context.WithTimeout(context.Background(), cfg.DockerConnectTimeout)
	if err := runtimeExecutor.Ping(dockerCtx); err != nil {
		cancelDocker()
		logger.Error("Docker connectivity check failed", "error", err)
		os.Exit(1)
	}
	cancelDocker()
	defer func() {
		if err := runtimeExecutor.Close(); err != nil {
			logger.Error("close Docker client", "error", err)
		}
	}()

	createEnvironment := createenvironment.New(store.Environments(), store.Workflows(), runtimeExecutor, logger)
	lifecycle := environmentlifecycle.New(store.Environments(), store.Workflows(), runtimeExecutor, logger)
	environmentService := environmentapi.New(createEnvironment, lifecycle, store.Environments(), store.Workflows())
	environmentHandler := server.NewEnvironmentHandler(environmentService)

	httpServer := &http.Server{
		Addr:              cfg.Address(),
		Handler:           server.NewRouter(environmentHandler, logger),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server", "address", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logger.Info("shutting down HTTP server")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("HTTP server stopped")
}

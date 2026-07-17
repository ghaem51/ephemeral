package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/config"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/server"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/storage/sqlite"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/createenvironment"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/environmentapi"
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
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	createEnvironment := createenvironment.New(store.Environments(), store.Workflows(), unavailableExecutor{})
	environmentService := environmentapi.New(createEnvironment, store.Environments(), store.Workflows())
	environmentHandler := server.NewEnvironmentHandler(environmentService)

	httpServer := &http.Server{
		Addr:              cfg.Address(),
		Handler:           server.NewRouter(environmentHandler),
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

type unavailableExecutor struct{}

func (unavailableExecutor) Create(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
	return domain.RuntimeInfo{}, errors.New("container executor is not configured")
}

func (unavailableExecutor) Start(context.Context, domain.RuntimeInfo) error {
	return errors.New("container executor is not configured")
}

func (unavailableExecutor) CheckHealth(context.Context, domain.RuntimeInfo) error {
	return errors.New("container executor is not configured")
}

func (unavailableExecutor) Destroy(context.Context, domain.RuntimeInfo) error {
	return errors.New("container executor is not configured")
}

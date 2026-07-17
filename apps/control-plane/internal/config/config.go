package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	defaultPort              = "8080"
	defaultDatabasePath      = "envpilot.db"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type Config struct {
	Port              string
	DatabasePath      string
	LogLevel          slog.Level
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	logLevel, err := parseLogLevel(envOrDefault("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := durationFromEnv("HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationFromEnv("HTTP_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:              envOrDefault("PORT", defaultPort),
		DatabasePath:      envOrDefault("DATABASE_PATH", defaultDatabasePath),
		LogLevel:          logLevel,
		ReadHeaderTimeout: readHeaderTimeout,
		ShutdownTimeout:   shutdownTimeout,
	}, nil
}

func (c Config) Address() string {
	return ":" + c.Port
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return duration, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("parse LOG_LEVEL: %w", err)
	}
	return level, nil
}

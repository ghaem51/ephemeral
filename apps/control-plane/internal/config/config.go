package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort                 = "8080"
	defaultDatabasePath         = "envpilot.db"
	defaultReadHeaderTimeout    = 5 * time.Second
	defaultShutdownTimeout      = 10 * time.Second
	defaultDockerImages         = "*"
	defaultHealthPath           = "/health"
	defaultDockerHealthHost     = "localhost"
	defaultHealthAttempts       = 15
	defaultHealthInterval       = time.Second
	defaultHealthTimeout        = 2 * time.Second
	defaultDockerStopTimeout    = 10 * time.Second
	defaultDockerConnectTimeout = 5 * time.Second
)

type Config struct {
	Port                 string
	DatabasePath         string
	LogLevel             slog.Level
	ReadHeaderTimeout    time.Duration
	ShutdownTimeout      time.Duration
	DockerImages         []string
	HealthPath           string
	DockerHealthHost     string
	HealthAttempts       int
	HealthInterval       time.Duration
	HealthTimeout        time.Duration
	DockerStopTimeout    time.Duration
	DockerConnectTimeout time.Duration
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
	healthAttempts, err := positiveIntFromEnv("DOCKER_HEALTH_ATTEMPTS", defaultHealthAttempts)
	if err != nil {
		return Config{}, err
	}
	healthInterval, err := durationFromEnv("DOCKER_HEALTH_INTERVAL", defaultHealthInterval)
	if err != nil {
		return Config{}, err
	}
	healthTimeout, err := durationFromEnv("DOCKER_HEALTH_TIMEOUT", defaultHealthTimeout)
	if err != nil {
		return Config{}, err
	}
	dockerStopTimeout, err := durationFromEnv("DOCKER_STOP_TIMEOUT", defaultDockerStopTimeout)
	if err != nil {
		return Config{}, err
	}
	dockerConnectTimeout, err := durationFromEnv("DOCKER_CONNECT_TIMEOUT", defaultDockerConnectTimeout)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:                 envOrDefault("PORT", defaultPort),
		DatabasePath:         envOrDefault("DATABASE_PATH", defaultDatabasePath),
		LogLevel:             logLevel,
		ReadHeaderTimeout:    readHeaderTimeout,
		ShutdownTimeout:      shutdownTimeout,
		DockerImages:         commaSeparatedEnv("DOCKER_ALLOWED_IMAGES", defaultDockerImages),
		HealthPath:           envOrDefault("DOCKER_HEALTH_PATH", defaultHealthPath),
		DockerHealthHost:     envOrDefault("DOCKER_HEALTH_HOST", defaultDockerHealthHost),
		HealthAttempts:       healthAttempts,
		HealthInterval:       healthInterval,
		HealthTimeout:        healthTimeout,
		DockerStopTimeout:    dockerStopTimeout,
		DockerConnectTimeout: dockerConnectTimeout,
	}, nil
}

func commaSeparatedEnv(key, fallback string) []string {
	values := strings.Split(envOrDefault(key, fallback), ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func positiveIntFromEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	healthyMode   = "healthy"
	unhealthyMode = "unhealthy"
)

type config struct {
	Port            string
	EnvironmentName string
	AppVersion      string
	HealthMode      string
	Hostname        string
	now             func() time.Time
}

type infoResponse struct {
	EnvironmentName    string `json:"environmentName"`
	ApplicationVersion string `json:"applicationVersion"`
	Timestamp          string `json:"timestamp"`
	Hostname           string `json:"hostname"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func loadConfig() (config, error) {
	mode := envOrDefault("HEALTH_MODE", healthyMode)
	if mode != healthyMode && mode != unhealthyMode {
		return config{}, fmt.Errorf("HEALTH_MODE must be %q or %q", healthyMode, unhealthyMode)
	}
	return config{
		Port:            envOrDefault("PORT", "8080"),
		EnvironmentName: envOrDefault("ENVIRONMENT_NAME", "local-demo"),
		AppVersion:      envOrDefault("APP_VERSION", "dev"),
		HealthMode:      mode,
		Hostname:        hostname(),
		now:             time.Now,
	}, nil
}

func newHandler(config config) http.Handler {
	mux := http.NewServeMux()
	info := func(response http.ResponseWriter, _ *http.Request) {
		writeJSON(response, http.StatusOK, infoResponse{
			EnvironmentName:    config.EnvironmentName,
			ApplicationVersion: config.AppVersion,
			Timestamp:          config.now().UTC().Format(time.RFC3339),
			Hostname:           config.Hostname,
		})
	}
	mux.HandleFunc("GET /{$}", info)
	mux.HandleFunc("GET /info", info)
	mux.HandleFunc("GET /health", func(response http.ResponseWriter, _ *http.Request) {
		if config.HealthMode == unhealthyMode {
			writeJSON(response, http.StatusServiceUnavailable, healthResponse{Status: unhealthyMode})
			return
		}
		writeJSON(response, http.StatusOK, healthResponse{Status: healthyMode})
	})
	return mux
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

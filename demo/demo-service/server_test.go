package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInfoEndpoints(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 30, 0, 0, time.UTC)
	handler := newHandler(config{
		EnvironmentName: "feature-payment", AppVersion: "1.2.3", HealthMode: healthyMode,
		Hostname: "demo-host", now: func() time.Time { return now },
	})

	for _, path := range []string{"/", "/info"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", response.Code)
			}
			var decoded infoResponse
			if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if decoded.EnvironmentName != "feature-payment" || decoded.ApplicationVersion != "1.2.3" || decoded.Timestamp != now.Format(time.RFC3339) || decoded.Hostname != "demo-host" {
				t.Fatalf("unexpected response: %#v", decoded)
			}
		})
	}
}

func TestHealthModes(t *testing.T) {
	tests := []struct {
		mode   string
		status int
	}{
		{mode: healthyMode, status: http.StatusOK},
		{mode: unhealthyMode, status: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			handler := newHandler(config{HealthMode: test.mode, now: time.Now})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
			if response.Code != test.status {
				t.Fatalf("expected %d, got %d", test.status, response.Code)
			}
		})
	}
}

func TestLoadConfigFromEnvironment(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ENVIRONMENT_NAME", "feature-payment")
	t.Setenv("APP_VERSION", "1.2.3")
	t.Setenv("HEALTH_MODE", unhealthyMode)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.Port != "9090" || config.EnvironmentName != "feature-payment" || config.AppVersion != "1.2.3" || config.HealthMode != unhealthyMode {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestLoadConfigRejectsUnknownHealthMode(t *testing.T) {
	t.Setenv("HEALTH_MODE", "sometimes")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected invalid health mode error")
	}
}

package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func TestCreateOptionsAreRestrictedAndLabeled(t *testing.T) {
	spec := validSpec()
	options := createOptions(spec)

	if options.Name != "envpilot-env-1" || options.Config.Image != spec.Image {
		t.Fatalf("unexpected create options: %#v", options)
	}
	if len(options.Config.Env) != 2 || options.Config.Env[0] != "ENVIRONMENT_NAME=preview" || options.Config.Env[1] != "APP_VERSION=1.2.3" {
		t.Fatalf("unexpected environment configuration: %#v", options.Config.Env)
	}
	wantLabels := map[string]string{
		LabelManaged: "true", LabelEnvironmentID: spec.ID, LabelEnvironmentName: spec.Name,
	}
	for key, want := range wantLabels {
		if got := options.Config.Labels[key]; got != want {
			t.Fatalf("label %q: want %q, got %q", key, want, got)
		}
	}
	if options.HostConfig.Privileged {
		t.Fatal("container must not be privileged")
	}
	if len(options.HostConfig.Binds) != 0 || len(options.HostConfig.Mounts) != 0 || len(options.Config.Volumes) != 0 {
		t.Fatal("container must not mount host paths or volumes")
	}
	if len(options.Config.ExposedPorts) != 1 || len(options.HostConfig.PortBindings) != 1 {
		t.Fatalf("unexpected port configuration: %#v", options.HostConfig.PortBindings)
	}
	for _, bindings := range options.HostConfig.PortBindings {
		if len(bindings) != 1 || bindings[0].HostIP.String() != "127.0.0.1" || bindings[0].HostPort != "" {
			t.Fatalf("expected dynamic loopback binding, got %#v", bindings)
		}
	}
}

func TestValidateSpecEnforcesImagePolicy(t *testing.T) {
	allowed := map[string]struct{}{"envpilot/demo-service:healthy": {}}
	if err := validateSpec(validSpec(), allowed, false); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	custom := validSpec()
	custom.Image = "ghcr.io/example/service:v2"
	if err := validateSpec(custom, allowed, true); err != nil {
		t.Fatalf("custom image rejected when any image is allowed: %v", err)
	}

	tests := []domain.EnvironmentSpec{
		{ID: "", Name: "preview", Image: "envpilot/demo-service:healthy", ContainerPort: 8080},
		{ID: "env-1", Name: "", Image: "envpilot/demo-service:healthy", ContainerPort: 8080},
		{ID: "env-1", Name: "preview", Image: "docker.io/library/alpine:latest", ContainerPort: 8080},
		{ID: "env-1", Name: "preview", Image: "envpilot/demo-service:healthy", ContainerPort: 0},
	}
	for _, spec := range tests {
		if err := validateSpec(spec, allowed, false); err == nil {
			t.Fatalf("expected spec to be rejected: %#v", spec)
		}
	}

	emptyImage := validSpec()
	emptyImage.Image = " "
	if err := validateSpec(emptyImage, allowed, true); err == nil {
		t.Fatal("expected empty image to be rejected even when any image is allowed")
	}
}

func TestRuntimeFromInspectionMapsDynamicHostPort(t *testing.T) {
	containerPort := 8080
	port := network.MustParsePort(strconv.Itoa(containerPort) + "/tcp")
	inspection := client.ContainerInspectResult{Container: container.InspectResponse{
		NetworkSettings: &container.NetworkSettings{
			Ports: network.PortMap{port: []network.PortBinding{{HostPort: "49152"}}},
		},
	}}

	runtime, err := runtimeFromInspection("container-1", containerPort, inspection)
	if err != nil {
		t.Fatalf("map runtime: %v", err)
	}
	if runtime.ContainerID != "container-1" || runtime.ContainerPort != 8080 || runtime.HostPort != 49152 || runtime.URL != "http://localhost:49152" {
		t.Fatalf("unexpected runtime: %#v", runtime)
	}
}

func TestRuntimeFromInspectionRejectsMissingPort(t *testing.T) {
	inspection := client.ContainerInspectResult{Container: container.InspectResponse{
		NetworkSettings: &container.NetworkSettings{Ports: network.PortMap{}},
	}}
	if _, err := runtimeFromInspection("container-1", 8080, inspection); err == nil {
		t.Fatal("expected missing port error")
	}
}

func TestHealthCheckURLUsesInternalHostWithoutChangingPublicRuntimeURL(t *testing.T) {
	publicURL := "http://localhost:49152"
	got, err := healthCheckURL(publicURL, "host.docker.internal", "/health")
	if err != nil {
		t.Fatalf("build health URL: %v", err)
	}
	if got != "http://host.docker.internal:49152/health" {
		t.Fatalf("unexpected health URL %q", got)
	}
	if publicURL != "http://localhost:49152" {
		t.Fatalf("public URL was changed: %q", publicURL)
	}
}

func TestCheckHealthRetriesUntilSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			http.Error(response, "not ready", http.StatusServiceUnavailable)
			return
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	executor := &Executor{
		healthClient: server.Client(), healthPath: "/health",
		healthAttempts: 3, healthInterval: time.Millisecond,
	}

	if err := executor.CheckHealth(context.Background(), domain.RuntimeInfo{URL: server.URL}); err != nil {
		t.Fatalf("check health: %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func validSpec() domain.EnvironmentSpec {
	return domain.EnvironmentSpec{
		ID: "env-1", Name: "preview", Image: "envpilot/demo-service:healthy", ContainerPort: 8080,
		ApplicationVersion: "1.2.3",
	}
}

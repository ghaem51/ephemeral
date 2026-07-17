package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor"
	"github.com/moby/moby/client"
)

const (
	LabelManaged         = "envpilot.managed"
	LabelEnvironmentID   = "envpilot.environment.id"
	LabelEnvironmentName = "envpilot.environment.name"
)

var _ executor.EnvironmentExecutor = (*Executor)(nil)

type Options struct {
	AllowedImages  []string
	HealthPath     string
	HealthAttempts int
	HealthInterval time.Duration
	HealthTimeout  time.Duration
	StopTimeout    time.Duration
}

type Executor struct {
	client         *client.Client
	allowedImages  map[string]struct{}
	healthClient   *http.Client
	healthPath     string
	healthAttempts int
	healthInterval time.Duration
	stopTimeout    time.Duration
}

func NewFromEnv(options Options) (*Executor, error) {
	if err := validateOptions(options); err != nil {
		return nil, err
	}

	// The current SDK negotiates the best compatible Engine API version by
	// default. client.FromEnv allows DOCKER_API_VERSION to pin it when needed.
	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}

	allowedImages := make(map[string]struct{}, len(options.AllowedImages))
	for _, image := range options.AllowedImages {
		allowedImages[image] = struct{}{}
	}
	return &Executor{
		client: dockerClient, allowedImages: allowedImages,
		healthClient: &http.Client{Timeout: options.HealthTimeout},
		healthPath:   options.HealthPath, healthAttempts: options.HealthAttempts,
		healthInterval: options.HealthInterval, stopTimeout: options.StopTimeout,
	}, nil
}

func (e *Executor) Close() error {
	return e.client.Close()
}

func (e *Executor) Create(ctx context.Context, spec domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
	if err := validateSpec(spec, e.allowedImages); err != nil {
		return domain.RuntimeInfo{}, err
	}

	result, err := e.client.ContainerCreate(ctx, createOptions(spec))
	if err != nil {
		return domain.RuntimeInfo{}, fmt.Errorf("create Docker container for environment %q: %w", spec.ID, err)
	}
	return domain.RuntimeInfo{ContainerID: result.ID, ContainerPort: spec.ContainerPort}, nil
}

func (e *Executor) Start(ctx context.Context, runtime domain.RuntimeInfo) (domain.RuntimeInfo, error) {
	if runtime.ContainerID == "" {
		return runtime, errors.New("start Docker container: container ID is required")
	}
	if runtime.ContainerPort < 1 || runtime.ContainerPort > 65535 {
		return runtime, fmt.Errorf("start Docker container %q: invalid container port %d", runtime.ContainerID, runtime.ContainerPort)
	}

	if _, err := e.client.ContainerStart(ctx, runtime.ContainerID, client.ContainerStartOptions{}); err != nil {
		return runtime, fmt.Errorf("start Docker container %q: %w", runtime.ContainerID, err)
	}
	inspection, err := e.client.ContainerInspect(ctx, runtime.ContainerID, client.ContainerInspectOptions{})
	if err != nil {
		return runtime, fmt.Errorf("inspect started Docker container %q: %w", runtime.ContainerID, err)
	}
	started, err := runtimeFromInspection(runtime.ContainerID, runtime.ContainerPort, inspection)
	if err != nil {
		return runtime, err
	}
	return started, nil
}

func (e *Executor) CheckHealth(ctx context.Context, runtime domain.RuntimeInfo) error {
	if runtime.URL == "" {
		return errors.New("check container health: runtime URL is required")
	}
	healthURL := strings.TrimRight(runtime.URL, "/") + e.healthPath
	var lastErr error

	for attempt := 1; attempt <= e.healthAttempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return fmt.Errorf("build health request: %w", err)
		}
		response, err := e.healthClient.Do(request)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("unexpected status %s", response.Status)
		} else {
			lastErr = err
		}

		if attempt < e.healthAttempts {
			timer := time.NewTimer(e.healthInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("check container health at %s: %w", healthURL, ctx.Err())
			case <-timer.C:
			}
		}
	}
	return fmt.Errorf("container health check failed after %d attempts at %s: %w", e.healthAttempts, healthURL, lastErr)
}

func (e *Executor) Destroy(ctx context.Context, runtime domain.RuntimeInfo) error {
	if runtime.ContainerID == "" {
		return nil
	}

	seconds := int(math.Ceil(e.stopTimeout.Seconds()))
	_, err := e.client.ContainerStop(ctx, runtime.ContainerID, client.ContainerStopOptions{Timeout: &seconds})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("stop Docker container %q: %w", runtime.ContainerID, err)
	}
	if errdefs.IsNotFound(err) {
		return nil
	}

	_, err = e.client.ContainerRemove(ctx, runtime.ContainerID, client.ContainerRemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("remove Docker container %q: %w", runtime.ContainerID, err)
	}
	return nil
}

func validateOptions(options Options) error {
	if len(options.AllowedImages) == 0 {
		return errors.New("configure Docker executor: at least one allowed image is required")
	}
	for _, image := range options.AllowedImages {
		if strings.TrimSpace(image) == "" {
			return errors.New("configure Docker executor: allowed images cannot be empty")
		}
	}
	if !strings.HasPrefix(options.HealthPath, "/") {
		return errors.New("configure Docker executor: health path must start with /")
	}
	if options.HealthAttempts < 1 || options.HealthInterval <= 0 || options.HealthTimeout <= 0 || options.StopTimeout <= 0 {
		return errors.New("configure Docker executor: health attempts and timeouts must be greater than zero")
	}
	return nil
}

func validateSpec(spec domain.EnvironmentSpec, allowedImages map[string]struct{}) error {
	if strings.TrimSpace(spec.ID) == "" {
		return errors.New("validate Docker environment spec: environment ID is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("validate Docker environment spec: environment name is required")
	}
	if _, allowed := allowedImages[spec.Image]; !allowed {
		return fmt.Errorf("validate Docker environment spec: image %q is not allowed", spec.Image)
	}
	if spec.ContainerPort < 1 || spec.ContainerPort > 65535 {
		return fmt.Errorf("validate Docker environment spec: container port %d is outside 1-65535", spec.ContainerPort)
	}
	return nil
}

func containerName(spec domain.EnvironmentSpec) string {
	return "envpilot-" + spec.ID
}

func localURL(hostPort int) string {
	return "http://localhost:" + strconv.Itoa(hostPort)
}

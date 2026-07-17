package executor_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor/executortest"
)

func TestWorkflowCanUseEnvironmentExecutor(t *testing.T) {
	ctx := context.Background()
	spec := domain.EnvironmentSpec{Name: "preview", Image: "demo:latest", ContainerPort: 8080}
	runtime := domain.RuntimeInfo{ContainerID: "container-1", HostPort: 49152, URL: "http://localhost:49152"}
	var calls []string

	fake := &executortest.Fake{
		CreateFunc: func(_ context.Context, got domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
			calls = append(calls, "create")
			if got != spec {
				t.Fatalf("unexpected spec: %#v", got)
			}
			return runtime, nil
		},
		StartFunc: func(_ context.Context, got domain.RuntimeInfo) (domain.RuntimeInfo, error) {
			calls = append(calls, "start")
			assertRuntime(t, got, runtime)
			return got, nil
		},
		CheckHealthFunc: func(_ context.Context, got domain.RuntimeInfo) error {
			calls = append(calls, "check health")
			assertRuntime(t, got, runtime)
			return nil
		},
		DestroyFunc: func(_ context.Context, got domain.RuntimeInfo) error {
			calls = append(calls, "destroy")
			assertRuntime(t, got, runtime)
			return nil
		},
	}

	if err := runLifecycle(ctx, fake, spec); err != nil {
		t.Fatalf("run lifecycle: %v", err)
	}

	want := []string{"create", "start", "check health", "destroy"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected calls: want %v, got %v", want, calls)
	}
}

func TestWorkflowStopsWhenExecutorOperationFails(t *testing.T) {
	startError := errors.New("start failed")
	healthCalled := false
	fake := &executortest.Fake{
		CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
			return domain.RuntimeInfo{ContainerID: "container-1"}, nil
		},
		StartFunc: func(_ context.Context, runtime domain.RuntimeInfo) (domain.RuntimeInfo, error) {
			return runtime, startError
		},
		CheckHealthFunc: func(context.Context, domain.RuntimeInfo) error {
			healthCalled = true
			return nil
		},
	}

	err := provision(context.Background(), fake, domain.EnvironmentSpec{Image: "demo:latest"})

	if !errors.Is(err, startError) {
		t.Fatalf("expected start error, got %v", err)
	}
	if healthCalled {
		t.Fatal("expected workflow to stop before health check")
	}
}

func runLifecycle(ctx context.Context, runtimeExecutor executor.EnvironmentExecutor, spec domain.EnvironmentSpec) error {
	runtime, err := runtimeExecutor.Create(ctx, spec)
	if err != nil {
		return err
	}
	runtime, err = runtimeExecutor.Start(ctx, runtime)
	if err != nil {
		return err
	}
	if err := runtimeExecutor.CheckHealth(ctx, runtime); err != nil {
		return err
	}
	return runtimeExecutor.Destroy(ctx, runtime)
}

func provision(ctx context.Context, runtimeExecutor executor.EnvironmentExecutor, spec domain.EnvironmentSpec) error {
	runtime, err := runtimeExecutor.Create(ctx, spec)
	if err != nil {
		return err
	}
	runtime, err = runtimeExecutor.Start(ctx, runtime)
	if err != nil {
		return err
	}
	return runtimeExecutor.CheckHealth(ctx, runtime)
}

func assertRuntime(t *testing.T, got, want domain.RuntimeInfo) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected runtime: want %#v, got %#v", want, got)
	}
}

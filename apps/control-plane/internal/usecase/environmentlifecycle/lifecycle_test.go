package environmentlifecycle

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor/executortest"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/storage/sqlite"
)

func TestSuccessfulDestruction(t *testing.T) {
	var destroyed domain.RuntimeInfo
	fake := &executortest.Fake{DestroyFunc: func(_ context.Context, runtime domain.RuntimeInfo) error {
		destroyed = runtime
		return nil
	}}
	uc, store := newLifecycleTest(t, fake)
	environment := createEnvironment(t, store, domain.EnvironmentStatusReady)

	accepted, err := uc.Destroy(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("destroy environment: %v", err)
	}
	if accepted.Status != domain.EnvironmentStatusDestroying {
		t.Fatalf("expected DESTROYING, got %s", accepted.Status)
	}
	workflow := waitForLifecycle(t, uc, store, environment.ID)
	persisted := getEnvironment(t, store, environment.ID)

	if persisted.Status != domain.EnvironmentStatusDestroyed || persisted.ContainerID != "" || persisted.URL != "" {
		t.Fatalf("environment was not destroyed: %#v", persisted)
	}
	if workflow.Operation != domain.OperationDestroy || workflow.Status != domain.WorkflowStatusSucceeded {
		t.Fatalf("destroy workflow did not succeed: %#v", workflow)
	}
	if destroyed.ContainerID != environment.ContainerID {
		t.Fatalf("unexpected destroyed runtime: %#v", destroyed)
	}
}

func TestRepeatedDestructionIsRejected(t *testing.T) {
	destroyCalls := 0
	fake := &executortest.Fake{DestroyFunc: func(context.Context, domain.RuntimeInfo) error {
		destroyCalls++
		return nil
	}}
	uc, store := newLifecycleTest(t, fake)
	environment := createEnvironment(t, store, domain.EnvironmentStatusReady)
	if _, err := uc.Destroy(context.Background(), environment.ID); err != nil {
		t.Fatalf("first destroy: %v", err)
	}
	waitForLifecycle(t, uc, store, environment.ID)

	_, err := uc.Destroy(context.Background(), environment.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
	if destroyCalls != 1 {
		t.Fatalf("expected one executor destroy call, got %d", destroyCalls)
	}
}

func TestDestroyRejectsInvalidState(t *testing.T) {
	uc, store := newLifecycleTest(t, &executortest.Fake{})
	environment := createEnvironment(t, store, domain.EnvironmentStatusPending)

	_, err := uc.Destroy(context.Background(), environment.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestDestroyFailureIsPersisted(t *testing.T) {
	failure := errors.New("daemon refused removal")
	fake := &executortest.Fake{DestroyFunc: func(context.Context, domain.RuntimeInfo) error { return failure }}
	uc, store := newLifecycleTest(t, fake)
	environment := createEnvironment(t, store, domain.EnvironmentStatusReady)
	if _, err := uc.Destroy(context.Background(), environment.ID); err != nil {
		t.Fatalf("destroy environment: %v", err)
	}
	workflow := waitForLifecycle(t, uc, store, environment.ID)
	persisted := getEnvironment(t, store, environment.ID)

	if persisted.Status != domain.EnvironmentStatusFailed || persisted.ErrorMessage != failure.Error() {
		t.Fatalf("destroy failure was not persisted: %#v", persisted)
	}
	if workflow.Status != domain.WorkflowStatusFailed || workflow.Steps[0].Status != domain.StepStatusFailed || workflow.Steps[0].ErrorMessage != failure.Error() {
		t.Fatalf("workflow failure was not persisted: %#v", workflow)
	}
	if workflow.Steps[1].Status != domain.StepStatusSkipped {
		t.Fatalf("step after destroy failure was not skipped: %#v", workflow.Steps[1])
	}
}

func TestRetryAfterHealthCheckFailure(t *testing.T) {
	var calls []string
	oldContainerID := "old-container"
	newRuntime := domain.RuntimeInfo{ContainerID: "new-container", ContainerPort: 8080}
	readyRuntime := domain.RuntimeInfo{
		ContainerID: "new-container", ContainerPort: 8080,
		HostPort: 49153, URL: "http://localhost:49153", HealthCheckPath: "/ready",
	}
	fake := &executortest.Fake{
		DestroyFunc: func(_ context.Context, runtime domain.RuntimeInfo) error {
			calls = append(calls, "destroy:"+runtime.ContainerID)
			return nil
		},
		CreateFunc: func(_ context.Context, spec domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
			calls = append(calls, "create")
			if spec.ApplicationVersion != "1.2.3" || spec.HealthCheckPath != "/ready" || len(spec.EnvironmentVariables) != 1 || spec.EnvironmentVariables[0] != "LOG_LEVEL=debug" {
				t.Fatalf("retry lost saved configuration: %#v", spec)
			}
			return newRuntime, nil
		},
		StartFunc: func(context.Context, domain.RuntimeInfo) (domain.RuntimeInfo, error) {
			calls = append(calls, "start")
			return readyRuntime, nil
		},
		CheckHealthFunc: func(_ context.Context, runtime domain.RuntimeInfo) error {
			calls = append(calls, "health")
			if runtime.HealthCheckPath != "/ready" {
				t.Fatalf("health check lost custom path: %#v", runtime)
			}
			return nil
		},
	}
	uc, store := newLifecycleTest(t, fake)
	environment := createEnvironment(t, store, domain.EnvironmentStatusFailed)
	environment.ContainerID = oldContainerID
	environment.ErrorMessage = "health check failed"
	if err := store.Environments().Update(context.Background(), environment); err != nil {
		t.Fatalf("update failed environment: %v", err)
	}
	previous := createPreviousWorkflow(t, store, environment.ID)

	accepted, err := uc.Retry(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("retry environment: %v", err)
	}
	if accepted.Status != domain.EnvironmentStatusProvisioning {
		t.Fatalf("expected PROVISIONING, got %s", accepted.Status)
	}
	retryWorkflow := waitForLifecycle(t, uc, store, environment.ID)
	persisted := getEnvironment(t, store, environment.ID)

	if got, want := calls, []string{"destroy:old-container", "create", "start", "health"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanup policy order: want %v, got %v", want, got)
	}
	if persisted.Status != domain.EnvironmentStatusReady || persisted.ContainerID != readyRuntime.ContainerID || persisted.URL != readyRuntime.URL {
		t.Fatalf("retry did not make environment ready: %#v", persisted)
	}
	if retryWorkflow.Operation != domain.OperationRetry || retryWorkflow.Status != domain.WorkflowStatusSucceeded {
		t.Fatalf("retry workflow did not succeed: %#v", retryWorkflow)
	}
	if _, err := store.Workflows().GetWithSteps(context.Background(), previous.ID); err != nil {
		t.Fatalf("previous workflow history was not preserved: %v", err)
	}
}

func TestRetryRejectsReadyEnvironment(t *testing.T) {
	uc, store := newLifecycleTest(t, &executortest.Fake{})
	environment := createEnvironment(t, store, domain.EnvironmentStatusReady)

	_, err := uc.Retry(context.Background(), environment.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func newLifecycleTest(t *testing.T, fake *executortest.Fake) (*UseCase, *sqlite.Store) {
	t.Helper()
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "envpilot.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	uc := New(store.Environments(), store.Workflows(), fake)
	uc.now = func() time.Time { return time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC) }
	return uc, store
}

func createEnvironment(t *testing.T, store *sqlite.Store, status domain.EnvironmentStatus) *domain.Environment {
	t.Helper()
	now := time.Date(2026, time.July, 16, 11, 0, 0, 0, time.UTC)
	environment := &domain.Environment{
		ID: "env-1", Name: "preview", Image: "envpilot/demo-service:healthy", ContainerPort: 8080,
		HealthCheckPath: "/ready", ApplicationVersion: "1.2.3",
		EnvironmentVariables: []string{"LOG_LEVEL=debug"},
		ContainerID:          "container-1", HostPort: 49152, URL: "http://localhost:49152",
		Status: status, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Environments().Create(context.Background(), environment); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	return environment
}

func createPreviousWorkflow(t *testing.T, store *sqlite.Store, environmentID string) *domain.Workflow {
	t.Helper()
	workflow := &domain.Workflow{
		ID: "previous-workflow", EnvironmentID: environmentID,
		Operation: domain.OperationCreate, Status: domain.WorkflowStatusFailed,
		Steps: []domain.WorkflowStep{{
			ID: "previous-step", WorkflowID: "previous-workflow", Name: StepCheckHealth,
			Order: 1, Status: domain.StepStatusFailed, ErrorMessage: "health check failed",
		}},
	}
	if err := store.Workflows().CreateWithSteps(context.Background(), workflow); err != nil {
		t.Fatalf("create previous workflow: %v", err)
	}
	return workflow
}

func waitForLifecycle(t *testing.T, uc *UseCase, store *sqlite.Store, environmentID string) *domain.Workflow {
	t.Helper()
	workflow, err := store.Workflows().GetLatestForEnvironment(context.Background(), environmentID)
	if err != nil {
		t.Fatalf("get latest workflow: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := uc.Wait(ctx, workflow.ID); err != nil {
		t.Fatalf("wait for workflow: %v", err)
	}
	workflow, err = store.Workflows().GetWithSteps(context.Background(), workflow.ID)
	if err != nil {
		t.Fatalf("reload workflow: %v", err)
	}
	return workflow
}

func getEnvironment(t *testing.T, store *sqlite.Store, id string) *domain.Environment {
	t.Helper()
	environment, err := store.Environments().GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("get environment: %v", err)
	}
	return environment
}

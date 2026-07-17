package createenvironment

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor/executortest"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/storage/sqlite"
)

func TestCreateRunsSuccessfulWorkflow(t *testing.T) {
	runtime := domain.RuntimeInfo{
		ContainerID: "container-1", HostPort: 49152, URL: "http://localhost:49152",
	}
	var calls []string
	fake := &executortest.Fake{
		CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
			calls = append(calls, "create")
			return runtime, nil
		},
		StartFunc: func(_ context.Context, runtime domain.RuntimeInfo) (domain.RuntimeInfo, error) {
			calls = append(calls, "start")
			return runtime, nil
		},
		CheckHealthFunc: func(context.Context, domain.RuntimeInfo) error {
			calls = append(calls, "health")
			return nil
		},
	}
	uc, store := newTestUseCase(t, fake)

	environment, err := uc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if environment.Status != domain.EnvironmentStatusPending {
		t.Fatalf("expected immediate PENDING environment, got %s", environment.Status)
	}
	workflow := waitForLatestWorkflow(t, uc, store, environment.ID)

	persisted, err := store.Environments().GetByID(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("get environment: %v", err)
	}
	if persisted.Status != domain.EnvironmentStatusReady {
		t.Fatalf("expected READY environment, got %s", persisted.Status)
	}
	if persisted.ContainerID != runtime.ContainerID || persisted.HostPort != runtime.HostPort || persisted.URL != runtime.URL {
		t.Fatalf("runtime was not persisted: %#v", persisted)
	}
	if workflow.Status != domain.WorkflowStatusSucceeded || workflow.StartedAt == nil || workflow.CompletedAt == nil {
		t.Fatalf("workflow did not succeed: %#v", workflow)
	}
	if len(workflow.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(workflow.Steps))
	}
	for index, step := range workflow.Steps {
		if step.Order != index+1 || step.Status != domain.StepStatusSucceeded {
			t.Fatalf("unexpected step %d: %#v", index, step)
		}
		if step.StartedAt == nil || step.CompletedAt == nil || step.Message != "step completed" {
			t.Fatalf("step lifecycle was not persisted: %#v", step)
		}
	}
	if got, want := strings.Join(calls, ","), "create,start,health"; got != want {
		t.Fatalf("unexpected executor calls: want %q, got %q", want, got)
	}
}

func TestCreateRejectsInvalidRequest(t *testing.T) {
	called := false
	fake := &executortest.Fake{CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
		called = true
		return domain.RuntimeInfo{}, nil
	}}
	uc, store := newTestUseCase(t, fake)

	_, err := uc.Create(context.Background(), Request{Name: " ", Image: "demo:latest", ContainerPort: 8080})

	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if called {
		t.Fatal("executor should not be called for an invalid request")
	}
	environments, listErr := store.Environments().List(context.Background())
	if listErr != nil {
		t.Fatalf("list environments: %v", listErr)
	}
	if len(environments) != 0 {
		t.Fatalf("invalid request persisted environments: %#v", environments)
	}
}

func TestCreateValidationRules(t *testing.T) {
	tests := []Request{
		{Name: "UPPERCASE", Image: "demo:latest", ContainerPort: 8080},
		{Name: "invalid_name", Image: "demo:latest", ContainerPort: 8080},
		{Name: "-leading-hyphen", Image: "demo:latest", ContainerPort: 8080},
		{Name: strings.Repeat("a", 64), Image: "demo:latest", ContainerPort: 8080},
		{Name: "preview", Image: "", ContainerPort: 8080},
		{Name: "preview", Image: "demo:latest", ContainerPort: 0},
		{Name: "preview", Image: "demo:latest", ContainerPort: 65536},
	}

	for _, request := range tests {
		t.Run(fmt.Sprintf("%s-%d", request.Name, request.ContainerPort), func(t *testing.T) {
			uc, _ := newTestUseCase(t, &executortest.Fake{})
			if _, err := uc.Create(context.Background(), request); !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestSimulateFailureSelectsUnhealthyDemoImage(t *testing.T) {
	spec, err := validate(Request{
		Name: "preview", Image: "envpilot/demo-service:healthy", ContainerPort: 8080,
		SimulateFailure: true,
	})
	if err != nil {
		t.Fatalf("validate request: %v", err)
	}
	if spec.Image != unhealthyDemoImage {
		t.Fatalf("expected image %q, got %q", unhealthyDemoImage, spec.Image)
	}
}

func TestCreateRejectsDuplicateActiveName(t *testing.T) {
	block := make(chan struct{})
	fake := &executortest.Fake{CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
		<-block
		return domain.RuntimeInfo{}, nil
	}}
	uc, store := newTestUseCase(t, fake)
	first, err := uc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("create first environment: %v", err)
	}
	defer func() {
		close(block)
		waitForLatestWorkflow(t, uc, store, first.ID)
	}()

	_, err = uc.Create(context.Background(), validRequest())
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestCreateWorkflowFailures(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(*executortest.Fake, error)
		failedStep  string
		wantRuntime bool
	}{
		{
			name: "container creation failure", failedStep: StepCreateContainer, wantRuntime: true,
			configure: func(fake *executortest.Fake, failure error) {
				fake.CreateFunc = func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
					return testRuntime(), failure
				}
			},
		},
		{
			name: "container start failure", failedStep: StepStartContainer, wantRuntime: true,
			configure: func(fake *executortest.Fake, failure error) {
				fake.CreateFunc = func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
					return testRuntime(), nil
				}
				fake.StartFunc = func(_ context.Context, runtime domain.RuntimeInfo) (domain.RuntimeInfo, error) {
					return runtime, failure
				}
			},
		},
		{
			name: "health check failure", failedStep: StepCheckHealth, wantRuntime: true,
			configure: func(fake *executortest.Fake, failure error) {
				fake.CreateFunc = func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
					return testRuntime(), nil
				}
				fake.CheckHealthFunc = func(context.Context, domain.RuntimeInfo) error { return failure }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := errors.New("infrastructure unavailable")
			fake := &executortest.Fake{}
			tt.configure(fake, failure)
			uc, store := newTestUseCase(t, fake)

			environment, err := uc.Create(context.Background(), validRequest())
			if err != nil {
				t.Fatalf("create environment: %v", err)
			}
			workflow := waitForLatestWorkflow(t, uc, store, environment.ID)
			persisted, err := store.Environments().GetByID(context.Background(), environment.ID)
			if err != nil {
				t.Fatalf("get environment: %v", err)
			}

			if persisted.Status != domain.EnvironmentStatusFailed || persisted.ErrorMessage != failure.Error() {
				t.Fatalf("environment failure was not persisted: %#v", persisted)
			}
			if tt.wantRuntime && persisted.ContainerID != testRuntime().ContainerID {
				t.Fatalf("available runtime was not preserved: %#v", persisted)
			}
			if workflow.Status != domain.WorkflowStatusFailed || workflow.CompletedAt == nil {
				t.Fatalf("workflow failure was not persisted: %#v", workflow)
			}
			failedStep := findStep(t, workflow, tt.failedStep)
			if failedStep.Status != domain.StepStatusFailed || failedStep.ErrorMessage != failure.Error() || failedStep.Message != "step failed" {
				t.Fatalf("step failure was not persisted: %#v", failedStep)
			}
		})
	}
}

func TestCreateRecoversPanicAndPersistsError(t *testing.T) {
	fake := &executortest.Fake{CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
		panic("executor exploded")
	}}
	uc, store := newTestUseCase(t, fake)

	environment, err := uc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	workflow := waitForLatestWorkflow(t, uc, store, environment.ID)
	persisted, err := store.Environments().GetByID(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("get environment: %v", err)
	}

	if persisted.Status != domain.EnvironmentStatusFailed || !strings.Contains(persisted.ErrorMessage, "workflow panic: executor exploded") {
		t.Fatalf("panic information was not persisted: %#v", persisted)
	}
	step := findStep(t, workflow, StepCreateContainer)
	if step.Status != domain.StepStatusFailed || !strings.Contains(step.ErrorMessage, "executor exploded") {
		t.Fatalf("panic was not persisted on current step: %#v", step)
	}
}

func TestBackgroundWorkflowDoesNotUseRequestCancellation(t *testing.T) {
	release := make(chan struct{})
	fake := &executortest.Fake{CreateFunc: func(ctx context.Context, _ domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
		<-release
		if err := ctx.Err(); err != nil {
			return domain.RuntimeInfo{}, err
		}
		return testRuntime(), nil
	}}
	uc, store := newTestUseCase(t, fake)
	requestCtx, cancel := context.WithCancel(context.Background())

	environment, err := uc.Create(requestCtx, validRequest())
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	cancel()
	close(release)
	workflow := waitForLatestWorkflow(t, uc, store, environment.ID)

	if workflow.Status != domain.WorkflowStatusSucceeded {
		t.Fatalf("request cancellation stopped background workflow: %#v", workflow)
	}
}

func TestWorkflowCannotStartTwiceInProcess(t *testing.T) {
	release := make(chan struct{})
	fake := &executortest.Fake{CreateFunc: func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
		<-release
		return testRuntime(), nil
	}}
	uc, store := newTestUseCase(t, fake)
	environment, err := uc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	workflow, err := store.Workflows().GetLatestForEnvironment(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}

	duplicateEnvironment := *environment
	duplicateWorkflow := *workflow
	if uc.start(&duplicateEnvironment, &duplicateWorkflow, domain.EnvironmentSpec{
		Name: environment.Name, Image: environment.Image, ContainerPort: environment.ContainerPort,
	}) {
		t.Fatal("expected duplicate workflow start to be rejected")
	}

	close(release)
	waitForLatestWorkflow(t, uc, store, environment.ID)
}

func newTestUseCase(t *testing.T, fake *executortest.Fake) (*UseCase, *sqlite.Store) {
	t.Helper()
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "envpilot.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close SQLite store: %v", err)
		}
	})
	uc := New(store.Environments(), store.Workflows(), fake)
	uc.now = func() time.Time { return time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC) }
	nextID := 0
	uc.newID = func() (string, error) {
		nextID++
		return fmt.Sprintf("id-%d", nextID), nil
	}
	return uc, store
}

func waitForLatestWorkflow(t *testing.T, uc *UseCase, store *sqlite.Store, environmentID string) *domain.Workflow {
	t.Helper()
	workflow, err := store.Workflows().GetLatestForEnvironment(context.Background(), environmentID)
	if err != nil {
		t.Fatalf("get latest workflow: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := uc.Wait(waitCtx, workflow.ID); err != nil {
		t.Fatalf("wait for workflow: %v", err)
	}
	workflow, err = store.Workflows().GetWithSteps(context.Background(), workflow.ID)
	if err != nil {
		t.Fatalf("reload workflow: %v", err)
	}
	return workflow
}

func validRequest() Request {
	return Request{Name: "preview", Image: "demo:latest", ContainerPort: 8080}
}

func testRuntime() domain.RuntimeInfo {
	return domain.RuntimeInfo{ContainerID: "container-1", HostPort: 49152, URL: "http://localhost:49152"}
}

func findStep(t *testing.T, workflow *domain.Workflow, name string) domain.WorkflowStep {
	t.Helper()
	for _, step := range workflow.Steps {
		if step.Name == name {
			return step
		}
	}
	t.Fatalf("step %q not found", name)
	return domain.WorkflowStep{}
}

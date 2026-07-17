package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

func TestEnvironmentRepository(t *testing.T) {
	store := openTestStore(t)
	repository := store.Environments()
	ctx := context.Background()
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	environment := &domain.Environment{
		ID: "env-1", Name: "preview", Image: "demo:latest", ContainerPort: 8080,
		EnvironmentVariables: []string{"LOG_LEVEL=debug", "EMPTY="},
		Status:               domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now,
	}

	if err := repository.Create(ctx, environment); err != nil {
		t.Fatalf("create environment: %v", err)
	}

	byID, err := repository.GetByID(ctx, environment.ID)
	if err != nil {
		t.Fatalf("get environment by ID: %v", err)
	}
	if !reflect.DeepEqual(*byID, *environment) {
		t.Fatalf("environment mismatch:\nwant: %#v\n got: %#v", environment, byID)
	}

	byName, err := repository.FindByName(ctx, environment.Name)
	if err != nil {
		t.Fatalf("find environment by name: %v", err)
	}
	if byName.ID != environment.ID {
		t.Fatalf("expected environment %q, got %q", environment.ID, byName.ID)
	}

	environment.Status = domain.EnvironmentStatusReady
	environment.HostPort = 49152
	environment.ContainerID = "container-1"
	environment.URL = "http://localhost:49152"
	environment.UpdatedAt = now.Add(time.Minute)
	if err := repository.Update(ctx, environment); err != nil {
		t.Fatalf("update environment: %v", err)
	}

	environments, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if len(environments) != 1 || environments[0].Status != domain.EnvironmentStatusReady {
		t.Fatalf("unexpected environments: %#v", environments)
	}
}

func TestEnvironmentRepositoryMapsDomainErrors(t *testing.T) {
	store := openTestStore(t)
	repository := store.Environments()
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := repository.GetByID(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	first := &domain.Environment{ID: "env-1", Name: "preview", Status: domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now}
	second := &domain.Environment{ID: "env-2", Name: "preview", Status: domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("create first environment: %v", err)
	}
	if err := repository.Create(ctx, second); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestWorkflowRepositoryCreatesAndLoadsOrderedSteps(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	createEnvironment(t, store, "env-1", "preview")

	workflow := &domain.Workflow{
		ID: "workflow-1", EnvironmentID: "env-1", Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusPending,
		Steps: []domain.WorkflowStep{
			{ID: "step-2", Name: "start", Order: 2, Status: domain.StepStatusPending},
			{ID: "step-1", Name: "validate", Order: 1, Status: domain.StepStatusPending},
		},
	}
	workflows := store.Workflows()
	if err := workflows.CreateWithSteps(ctx, workflow); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	loaded, err := workflows.GetWithSteps(ctx, workflow.ID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}
	if len(loaded.Steps) != 2 || loaded.Steps[0].Order != 1 || loaded.Steps[1].Order != 2 {
		t.Fatalf("steps are not ordered: %#v", loaded.Steps)
	}
	if loaded.Steps[0].WorkflowID != workflow.ID {
		t.Fatalf("expected workflow ID %q, got %q", workflow.ID, loaded.Steps[0].WorkflowID)
	}

	startedAt := time.Now().UTC().Truncate(time.Nanosecond)
	workflow.Status = domain.WorkflowStatusRunning
	workflow.StartedAt = &startedAt
	if err := workflows.Update(ctx, workflow); err != nil {
		t.Fatalf("update workflow: %v", err)
	}

	step := &workflow.Steps[0]
	step.Status = domain.StepStatusRunning
	step.StartedAt = &startedAt
	if err := workflows.UpdateStep(ctx, step); err != nil {
		t.Fatalf("update step: %v", err)
	}

	updated, err := workflows.GetWithSteps(ctx, workflow.ID)
	if err != nil {
		t.Fatalf("get updated workflow: %v", err)
	}
	if updated.Status != domain.WorkflowStatusRunning || updated.StartedAt == nil || !updated.StartedAt.Equal(startedAt) {
		t.Fatalf("workflow update was not persisted: %#v", updated)
	}
}

func TestWorkflowRepositoryReturnsLatestWorkflow(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	createEnvironment(t, store, "env-1", "preview")
	workflows := store.Workflows()

	for _, id := range []string{"workflow-1", "workflow-2"} {
		workflow := &domain.Workflow{
			ID: id, EnvironmentID: "env-1", Operation: domain.OperationCreate,
			Status: domain.WorkflowStatusPending,
		}
		if err := workflows.CreateWithSteps(ctx, workflow); err != nil {
			t.Fatalf("create workflow %q: %v", id, err)
		}
	}

	latest, err := workflows.GetLatestForEnvironment(ctx, "env-1")
	if err != nil {
		t.Fatalf("get latest workflow: %v", err)
	}
	if latest.ID != "workflow-2" {
		t.Fatalf("expected latest workflow %q, got %q", "workflow-2", latest.ID)
	}
}

func TestCreateWorkflowWithStepsRollsBackOnInvalidStep(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	createEnvironment(t, store, "env-1", "preview")
	workflow := &domain.Workflow{
		ID: "workflow-1", EnvironmentID: "env-1", Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusPending,
		Steps: []domain.WorkflowStep{{
			ID: "step-1", WorkflowID: "another-workflow", Name: "validate", Order: 1,
			Status: domain.StepStatusPending,
		}},
	}

	err := store.Workflows().CreateWithSteps(ctx, workflow)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if _, err := store.Workflows().GetWithSteps(ctx, workflow.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected rolled-back workflow to be missing, got %v", err)
	}
}

func TestForeignKeysAreEnabled(t *testing.T) {
	store := openTestStore(t)
	workflow := &domain.Workflow{
		ID: "workflow-1", EnvironmentID: "missing", Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusPending,
	}

	if err := store.Workflows().CreateWithSteps(context.Background(), workflow); err == nil {
		t.Fatal("expected foreign-key violation")
	}
}

func TestOpenAddsNewColumnsToExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE environments (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, image TEXT NOT NULL,
		container_port INTEGER NOT NULL, host_port INTEGER NOT NULL,
		container_id TEXT NOT NULL, url TEXT NOT NULL, status TEXT NOT NULL,
		error_message TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	environment := &domain.Environment{
		ID: "env-versioned", Name: "versioned", Image: "demo:latest", ContainerPort: 8080,
		HealthCheckPath: "/ready", ApplicationVersion: "2.0.0",
		EnvironmentVariables: []string{"LOG_LEVEL=debug"},
		Status:               domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Environments().Create(context.Background(), environment); err != nil {
		t.Fatalf("create migrated environment: %v", err)
	}
	loaded, err := store.Environments().GetByID(context.Background(), environment.ID)
	if err != nil {
		t.Fatalf("load migrated environment: %v", err)
	}
	if loaded.ApplicationVersion != environment.ApplicationVersion {
		t.Fatalf("application version was not migrated: %#v", loaded)
	}
	if loaded.HealthCheckPath != environment.HealthCheckPath {
		t.Fatalf("health check path was not migrated: %#v", loaded)
	}
	if !reflect.DeepEqual(loaded.EnvironmentVariables, environment.EnvironmentVariables) {
		t.Fatalf("environment variables were not migrated: %#v", loaded)
	}
}

func TestRecoverStaleWorkflowsMarksWorkflowStepAndEnvironmentFailed(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	environment := &domain.Environment{
		ID: "env-stale", Name: "stale-preview", Image: "demo:latest", ContainerPort: 8080,
		ContainerID: "container-still-present", Status: domain.EnvironmentStatusProvisioning,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Minute),
	}
	if err := store.Environments().Create(ctx, environment); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	startedAt := now.Add(-time.Minute)
	workflow := &domain.Workflow{
		ID: "workflow-stale", EnvironmentID: environment.ID, Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusRunning, StartedAt: &startedAt,
		Steps: []domain.WorkflowStep{
			{ID: "step-running", Name: "CHECK_HEALTH", Order: 1, Status: domain.StepStatusRunning, StartedAt: &startedAt},
			{ID: "step-pending", Name: "MARK_READY", Order: 2, Status: domain.StepStatusPending},
		},
	}
	if err := store.Workflows().CreateWithSteps(ctx, workflow); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	recovered, err := store.RecoverStaleWorkflows(ctx, now)
	if err != nil {
		t.Fatalf("recover stale workflows: %v", err)
	}
	if len(recovered) != 1 || recovered[0].WorkflowID != workflow.ID || recovered[0].EnvironmentID != environment.ID {
		t.Fatalf("unexpected recovery result: %#v", recovered)
	}

	gotWorkflow, err := store.Workflows().GetWithSteps(ctx, workflow.ID)
	if err != nil {
		t.Fatalf("get recovered workflow: %v", err)
	}
	if gotWorkflow.Status != domain.WorkflowStatusFailed || gotWorkflow.CompletedAt == nil || !gotWorkflow.CompletedAt.Equal(now) {
		t.Fatalf("workflow was not failed: %#v", gotWorkflow)
	}
	if gotWorkflow.Steps[0].Status != domain.StepStatusFailed || gotWorkflow.Steps[0].ErrorMessage != staleWorkflowMessage || gotWorkflow.Steps[0].CompletedAt == nil {
		t.Fatalf("running step was not failed with recovery context: %#v", gotWorkflow.Steps[0])
	}
	if gotWorkflow.Steps[1].Status != domain.StepStatusSkipped || gotWorkflow.Steps[1].Message != "skipped after workflow interruption" {
		t.Fatalf("pending step should be closed as skipped: %#v", gotWorkflow.Steps[1])
	}

	gotEnvironment, err := store.Environments().GetByID(ctx, environment.ID)
	if err != nil {
		t.Fatalf("get recovered environment: %v", err)
	}
	if gotEnvironment.Status != domain.EnvironmentStatusFailed || gotEnvironment.ErrorMessage != staleWorkflowMessage {
		t.Fatalf("environment was not left recoverable: %#v", gotEnvironment)
	}
	if gotEnvironment.ContainerID != environment.ContainerID {
		t.Fatalf("known runtime must be retained for retry or destroy: %#v", gotEnvironment)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "envpilot.db"))
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close test store: %v", err)
		}
	})
	return store
}

func createEnvironment(t *testing.T, store *Store, id, name string) {
	t.Helper()
	now := time.Now().UTC()
	environment := &domain.Environment{
		ID: id, Name: name, Image: "demo:latest", ContainerPort: 8080,
		Status: domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Environments().Create(context.Background(), environment); err != nil {
		t.Fatalf("create environment fixture: %v", err)
	}
}

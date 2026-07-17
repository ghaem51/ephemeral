package environmentlifecycle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/repository"
)

const (
	StepDestroyContainer = "DESTROY_CONTAINER"
	StepMarkDestroyed    = "MARK_DESTROYED"
	StepCleanupContainer = "CLEANUP_CONTAINER"
	StepCreateContainer  = "CREATE_CONTAINER"
	StepStartContainer   = "START_CONTAINER"
	StepCheckHealth      = "CHECK_HEALTH"
	StepMarkReady        = "MARK_READY"
)

type UseCase struct {
	environments repository.EnvironmentRepository
	workflows    repository.WorkflowRepository
	executor     executor.EnvironmentExecutor
	now          func() time.Time
	newID        func() (string, error)
	logger       *slog.Logger

	mu          sync.Mutex
	started     map[string]chan struct{}
	admissionMu sync.Mutex
}

func New(
	environments repository.EnvironmentRepository,
	workflows repository.WorkflowRepository,
	runtimeExecutor executor.EnvironmentExecutor,
	loggers ...*slog.Logger,
) *UseCase {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &UseCase{
		environments: environments, workflows: workflows, executor: runtimeExecutor,
		now: func() time.Time { return time.Now().UTC() }, newID: randomID,
		logger:  logger,
		started: make(map[string]chan struct{}),
	}
}

func (uc *UseCase) Destroy(ctx context.Context, environmentID string) (*domain.Environment, error) {
	uc.admissionMu.Lock()
	defer uc.admissionMu.Unlock()

	environment, err := uc.environments.GetByID(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	if environment.Status != domain.EnvironmentStatusReady && environment.Status != domain.EnvironmentStatusFailed {
		return nil, invalidState("destroy", environment.Status)
	}

	workflow, err := uc.newWorkflow(environment.ID, domain.OperationDestroy, []string{
		StepDestroyContainer, StepMarkDestroyed,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.workflows.CreateWithSteps(ctx, workflow); err != nil {
		return nil, fmt.Errorf("persist destroy workflow: %w", err)
	}
	if err := environment.TransitionTo(domain.EnvironmentStatusDestroying, uc.now()); err != nil {
		return nil, err
	}
	environment.ErrorMessage = ""
	if err := uc.environments.Update(ctx, environment); err != nil {
		return nil, fmt.Errorf("persist destroying environment: %w", err)
	}

	backgroundEnvironment := *environment
	backgroundWorkflow := cloneWorkflow(workflow)
	uc.start(&backgroundEnvironment, backgroundWorkflow, uc.executeDestroy)
	return environment, nil
}

// Retry always cleans up a known runtime before creating a fresh container.
// This deliberately trades speed for a simple guarantee that retries cannot
// accidentally leave two containers serving the same environment.
func (uc *UseCase) Retry(ctx context.Context, environmentID string) (*domain.Environment, error) {
	uc.admissionMu.Lock()
	defer uc.admissionMu.Unlock()

	environment, err := uc.environments.GetByID(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	if environment.Status != domain.EnvironmentStatusFailed {
		return nil, invalidState("retry", environment.Status)
	}

	workflow, err := uc.newWorkflow(environment.ID, domain.OperationRetry, []string{
		StepCleanupContainer, StepCreateContainer, StepStartContainer, StepCheckHealth, StepMarkReady,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.workflows.CreateWithSteps(ctx, workflow); err != nil {
		return nil, fmt.Errorf("persist retry workflow: %w", err)
	}
	if err := environment.TransitionTo(domain.EnvironmentStatusProvisioning, uc.now()); err != nil {
		return nil, err
	}
	environment.ErrorMessage = ""
	if err := uc.environments.Update(ctx, environment); err != nil {
		return nil, fmt.Errorf("persist retrying environment: %w", err)
	}

	backgroundEnvironment := *environment
	backgroundWorkflow := cloneWorkflow(workflow)
	uc.start(&backgroundEnvironment, backgroundWorkflow, uc.executeRetry)
	return environment, nil
}

func (uc *UseCase) Wait(ctx context.Context, workflowID string) error {
	uc.mu.Lock()
	done, ok := uc.started[workflowID]
	uc.mu.Unlock()
	if !ok {
		return fmt.Errorf("workflow %q: %w", workflowID, domain.ErrNotFound)
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (uc *UseCase) newWorkflow(environmentID string, operation domain.Operation, names []string) (*domain.Workflow, error) {
	workflowID, err := uc.newID()
	if err != nil {
		return nil, fmt.Errorf("generate workflow ID: %w", err)
	}
	steps := make([]domain.WorkflowStep, 0, len(names))
	for index, name := range names {
		stepID, err := uc.newID()
		if err != nil {
			return nil, fmt.Errorf("generate workflow step ID: %w", err)
		}
		steps = append(steps, domain.WorkflowStep{
			ID: stepID, WorkflowID: workflowID, Name: name,
			Order: index + 1, Status: domain.StepStatusPending,
		})
	}
	return &domain.Workflow{
		ID: workflowID, EnvironmentID: environmentID, Operation: operation,
		Status: domain.WorkflowStatusPending, Steps: steps,
	}, nil
}

func invalidState(operation string, status domain.EnvironmentStatus) error {
	return fmt.Errorf("cannot %s environment in %s state: %w", operation, status, domain.ErrInvalidTransition)
}

func cloneWorkflow(workflow *domain.Workflow) *domain.Workflow {
	clone := *workflow
	clone.Steps = append([]domain.WorkflowStep(nil), workflow.Steps...)
	return &clone
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

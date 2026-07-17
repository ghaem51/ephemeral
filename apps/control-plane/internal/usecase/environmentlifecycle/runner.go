package environmentlifecycle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

type execution func(context.Context, *domain.Environment, *domain.Workflow)
type stepOperation func(context.Context) error

func (uc *UseCase) start(environment *domain.Environment, workflow *domain.Workflow, execute execution) bool {
	uc.mu.Lock()
	if _, exists := uc.started[workflow.ID]; exists {
		uc.mu.Unlock()
		return false
	}
	done := make(chan struct{})
	uc.started[workflow.ID] = done
	uc.mu.Unlock()

	go func() {
		defer close(done)
		ctx := context.Background()
		defer func() {
			if recovered := recover(); recovered != nil {
				uc.fail(ctx, environment, workflow, runningStep(workflow), fmt.Errorf("workflow panic: %v", recovered))
			}
		}()
		execute(ctx, environment, workflow)
	}()
	return true
}

func (uc *UseCase) executeDestroy(ctx context.Context, environment *domain.Environment, workflow *domain.Workflow) {
	if !uc.beginWorkflow(ctx, environment, workflow) {
		return
	}
	runtime := runtimeFromEnvironment(environment)
	operations := []stepOperation{
		func(ctx context.Context) error { return uc.executor.Destroy(ctx, runtime) },
		func(ctx context.Context) error {
			destroyed := *environment
			if err := destroyed.TransitionTo(domain.EnvironmentStatusDestroyed, uc.now()); err != nil {
				return err
			}
			clearRuntime(&destroyed)
			destroyed.ErrorMessage = ""
			if err := uc.environments.Update(ctx, &destroyed); err != nil {
				return err
			}
			*environment = destroyed
			return nil
		},
	}
	uc.run(ctx, environment, workflow, operations)
}

func (uc *UseCase) executeRetry(ctx context.Context, environment *domain.Environment, workflow *domain.Workflow) {
	if !uc.beginWorkflow(ctx, environment, workflow) {
		return
	}
	runtime := runtimeFromEnvironment(environment)
	spec := domain.EnvironmentSpec{
		ID: environment.ID, Name: environment.Name, Image: environment.Image,
		ContainerPort: environment.ContainerPort, ApplicationVersion: environment.ApplicationVersion,
	}
	operations := []stepOperation{
		func(ctx context.Context) error {
			if runtime.ContainerID != "" {
				if err := uc.executor.Destroy(ctx, runtime); err != nil {
					return err
				}
			}
			clearRuntime(environment)
			return uc.environments.Update(ctx, environment)
		},
		func(ctx context.Context) error {
			created, err := uc.executor.Create(ctx, spec)
			runtime = created
			applyRuntime(environment, runtime)
			if updateErr := uc.environments.Update(ctx, environment); updateErr != nil {
				return fmt.Errorf("persist created retry runtime: %w", updateErr)
			}
			return err
		},
		func(ctx context.Context) error {
			started, err := uc.executor.Start(ctx, runtime)
			runtime = started
			applyRuntime(environment, runtime)
			if updateErr := uc.environments.Update(ctx, environment); updateErr != nil {
				return fmt.Errorf("persist started retry runtime: %w", updateErr)
			}
			return err
		},
		func(ctx context.Context) error { return uc.executor.CheckHealth(ctx, runtime) },
		func(ctx context.Context) error {
			ready := *environment
			if err := ready.TransitionTo(domain.EnvironmentStatusReady, uc.now()); err != nil {
				return err
			}
			applyRuntime(&ready, runtime)
			ready.ErrorMessage = ""
			if err := uc.environments.Update(ctx, &ready); err != nil {
				return err
			}
			*environment = ready
			return nil
		},
	}
	uc.run(ctx, environment, workflow, operations)
}

func (uc *UseCase) beginWorkflow(ctx context.Context, environment *domain.Environment, workflow *domain.Workflow) bool {
	if err := workflow.TransitionTo(domain.WorkflowStatusRunning, uc.now()); err != nil {
		uc.fail(ctx, environment, workflow, nil, err)
		return false
	}
	if err := uc.workflows.Update(ctx, workflow); err != nil {
		uc.fail(ctx, environment, workflow, nil, fmt.Errorf("persist running workflow: %w", err))
		return false
	}
	return true
}

func (uc *UseCase) run(
	ctx context.Context,
	environment *domain.Environment,
	workflow *domain.Workflow,
	operations []stepOperation,
) {
	logger := uc.logger.With("environment_id", environment.ID, "workflow_id", workflow.ID, "operation", workflow.Operation)
	logger.Info("lifecycle workflow started")
	for index := range workflow.Steps {
		step := &workflow.Steps[index]
		if err := uc.runStep(ctx, step, operations[index], logger); err != nil {
			uc.fail(ctx, environment, workflow, step, err)
			return
		}
	}
	succeeded := *workflow
	if err := succeeded.TransitionTo(domain.WorkflowStatusSucceeded, uc.now()); err != nil {
		uc.fail(ctx, environment, workflow, nil, err)
		return
	}
	if err := uc.workflows.Update(ctx, &succeeded); err != nil {
		uc.fail(ctx, environment, workflow, nil, fmt.Errorf("persist succeeded workflow: %w", err))
		return
	}
	*workflow = succeeded
	logger.Info("lifecycle workflow succeeded")
}

func (uc *UseCase) runStep(ctx context.Context, step *domain.WorkflowStep, operation stepOperation, logger *slog.Logger) error {
	if err := step.TransitionTo(domain.StepStatusRunning, uc.now()); err != nil {
		return err
	}
	step.Message = "step started"
	step.ErrorMessage = ""
	if err := uc.workflows.UpdateStep(ctx, step); err != nil {
		return fmt.Errorf("persist running step %s: %w", step.Name, err)
	}
	logger.Info("workflow step started", "step", step.Name)
	if err := operation(ctx); err != nil {
		return err
	}
	succeeded := *step
	if err := succeeded.TransitionTo(domain.StepStatusSucceeded, uc.now()); err != nil {
		return err
	}
	succeeded.Message = "step completed"
	if err := uc.workflows.UpdateStep(ctx, &succeeded); err != nil {
		return fmt.Errorf("persist succeeded step %s: %w", step.Name, err)
	}
	*step = succeeded
	logger.Info("workflow step succeeded", "step", step.Name)
	return nil
}

func (uc *UseCase) fail(ctx context.Context, environment *domain.Environment, workflow *domain.Workflow, step *domain.WorkflowStep, cause error) {
	message := cause.Error()
	uc.logger.Error("lifecycle workflow failed", "environment_id", environment.ID, "workflow_id", workflow.ID, "operation", workflow.Operation, "step", stepName(step), "error", cause)
	if step != nil && step.Status == domain.StepStatusRunning {
		if err := step.TransitionTo(domain.StepStatusFailed, uc.now()); err == nil {
			step.Message = "step failed"
			step.ErrorMessage = message
			_ = uc.workflows.UpdateStep(ctx, step)
		}
	}
	uc.skipPendingSteps(ctx, workflow)
	if workflow.Status == domain.WorkflowStatusRunning {
		if err := workflow.TransitionTo(domain.WorkflowStatusFailed, uc.now()); err == nil {
			_ = uc.workflows.Update(ctx, workflow)
		}
	}
	if environment.Status == domain.EnvironmentStatusDestroying || environment.Status == domain.EnvironmentStatusProvisioning {
		if err := environment.TransitionTo(domain.EnvironmentStatusFailed, uc.now()); err == nil {
			environment.ErrorMessage = message
			_ = uc.environments.Update(ctx, environment)
		}
	}
}

func (uc *UseCase) skipPendingSteps(ctx context.Context, workflow *domain.Workflow) {
	for index := range workflow.Steps {
		step := &workflow.Steps[index]
		if step.Status != domain.StepStatusPending {
			continue
		}
		if err := step.TransitionTo(domain.StepStatusSkipped, uc.now()); err == nil {
			step.Message = "skipped after workflow failure"
			_ = uc.workflows.UpdateStep(ctx, step)
		}
	}
}

func stepName(step *domain.WorkflowStep) string {
	if step == nil {
		return ""
	}
	return step.Name
}

func runningStep(workflow *domain.Workflow) *domain.WorkflowStep {
	for index := range workflow.Steps {
		if workflow.Steps[index].Status == domain.StepStatusRunning {
			return &workflow.Steps[index]
		}
	}
	return nil
}

func runtimeFromEnvironment(environment *domain.Environment) domain.RuntimeInfo {
	return domain.RuntimeInfo{
		ContainerID: environment.ContainerID, ContainerPort: environment.ContainerPort,
		HostPort: environment.HostPort, URL: environment.URL,
	}
}

func applyRuntime(environment *domain.Environment, runtime domain.RuntimeInfo) {
	environment.ContainerID = runtime.ContainerID
	environment.HostPort = runtime.HostPort
	environment.URL = runtime.URL
}

func clearRuntime(environment *domain.Environment) {
	environment.ContainerID = ""
	environment.HostPort = 0
	environment.URL = ""
}

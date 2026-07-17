package createenvironment

import (
	"context"
	"fmt"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

type stepOperation func(context.Context) error

func (uc *UseCase) start(environment *domain.Environment, workflow *domain.Workflow, spec domain.EnvironmentSpec) bool {
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
		// Never inherit cancellation from the request that initiated creation.
		ctx := context.Background()
		var currentStep *domain.WorkflowStep
		defer func() {
			if recovered := recover(); recovered != nil {
				uc.persistFailure(ctx, environment, workflow, currentStep, fmt.Errorf("workflow panic: %v", recovered))
			}
		}()

		uc.execute(ctx, environment, workflow, spec, &currentStep)
	}()
	return true
}

func (uc *UseCase) execute(
	ctx context.Context,
	environment *domain.Environment,
	workflow *domain.Workflow,
	spec domain.EnvironmentSpec,
	currentStep **domain.WorkflowStep,
) {
	if err := workflow.TransitionTo(domain.WorkflowStatusRunning, uc.now()); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, err)
		return
	}
	if err := uc.workflows.Update(ctx, workflow); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, fmt.Errorf("persist running workflow: %w", err))
		return
	}
	if err := environment.TransitionTo(domain.EnvironmentStatusProvisioning, uc.now()); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, err)
		return
	}
	if err := uc.environments.Update(ctx, environment); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, fmt.Errorf("persist provisioning environment: %w", err))
		return
	}

	var runtime domain.RuntimeInfo
	operations := []stepOperation{
		func(context.Context) error { return nil },
		func(ctx context.Context) error {
			created, err := uc.executor.Create(ctx, spec)
			runtime = created
			applyRuntime(environment, runtime)
			if updateErr := uc.environments.Update(ctx, environment); updateErr != nil {
				return fmt.Errorf("persist created runtime: %w", updateErr)
			}
			return err
		},
		func(ctx context.Context) error {
			started, err := uc.executor.Start(ctx, runtime)
			runtime = started
			applyRuntime(environment, runtime)
			if updateErr := uc.environments.Update(ctx, environment); updateErr != nil {
				return fmt.Errorf("persist started runtime: %w", updateErr)
			}
			return err
		},
		func(ctx context.Context) error { return uc.executor.CheckHealth(ctx, runtime) },
		func(ctx context.Context) error {
			ready := *environment
			applyRuntime(&ready, runtime)
			if err := ready.TransitionTo(domain.EnvironmentStatusReady, uc.now()); err != nil {
				return err
			}
			if err := uc.environments.Update(ctx, &ready); err != nil {
				return err
			}
			*environment = ready
			return nil
		},
	}

	for index := range workflow.Steps {
		step := &workflow.Steps[index]
		*currentStep = step
		if err := uc.runStep(ctx, step, operations[index]); err != nil {
			uc.persistFailure(ctx, environment, workflow, step, err)
			return
		}
	}
	*currentStep = nil

	succeeded := *workflow
	if err := succeeded.TransitionTo(domain.WorkflowStatusSucceeded, uc.now()); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, err)
		return
	}
	if err := uc.workflows.Update(ctx, &succeeded); err != nil {
		uc.persistFailure(ctx, environment, workflow, nil, fmt.Errorf("persist succeeded workflow: %w", err))
		return
	}
	*workflow = succeeded
}

func (uc *UseCase) runStep(ctx context.Context, step *domain.WorkflowStep, operation stepOperation) error {
	if err := step.TransitionTo(domain.StepStatusRunning, uc.now()); err != nil {
		return err
	}
	step.Message = "step started"
	step.ErrorMessage = ""
	if err := uc.workflows.UpdateStep(ctx, step); err != nil {
		return fmt.Errorf("persist running step %s: %w", step.Name, err)
	}

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
	return nil
}

func (uc *UseCase) persistFailure(
	ctx context.Context,
	environment *domain.Environment,
	workflow *domain.Workflow,
	step *domain.WorkflowStep,
	cause error,
) {
	message := cause.Error()
	if step != nil && step.Status == domain.StepStatusRunning {
		if err := step.TransitionTo(domain.StepStatusFailed, uc.now()); err == nil {
			step.Message = "step failed"
			step.ErrorMessage = message
			_ = uc.workflows.UpdateStep(ctx, step)
		}
	}
	if workflow.Status == domain.WorkflowStatusRunning {
		if err := workflow.TransitionTo(domain.WorkflowStatusFailed, uc.now()); err == nil {
			_ = uc.workflows.Update(ctx, workflow)
		}
	}
	if environment.Status == domain.EnvironmentStatusPending || environment.Status == domain.EnvironmentStatusProvisioning {
		if err := environment.TransitionTo(domain.EnvironmentStatusFailed, uc.now()); err == nil {
			environment.ErrorMessage = message
			_ = uc.environments.Update(ctx, environment)
		}
	}
}

func applyRuntime(environment *domain.Environment, runtime domain.RuntimeInfo) {
	environment.ContainerID = runtime.ContainerID
	environment.HostPort = runtime.HostPort
	environment.URL = runtime.URL
}

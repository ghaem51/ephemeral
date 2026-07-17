package domain

import (
	"errors"
	"testing"
	"time"
)

func TestEnvironmentTransitionTo(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		from EnvironmentStatus
		to   EnvironmentStatus
	}{
		{name: "begin provisioning", from: EnvironmentStatusPending, to: EnvironmentStatusProvisioning},
		{name: "provisioning succeeds", from: EnvironmentStatusProvisioning, to: EnvironmentStatusReady},
		{name: "provisioning fails", from: EnvironmentStatusProvisioning, to: EnvironmentStatusFailed},
		{name: "retry failure", from: EnvironmentStatusFailed, to: EnvironmentStatusProvisioning},
		{name: "destroy ready environment", from: EnvironmentStatusReady, to: EnvironmentStatusDestroying},
		{name: "finish destruction", from: EnvironmentStatusDestroying, to: EnvironmentStatusDestroyed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := Environment{Status: tt.from}

			if err := environment.TransitionTo(tt.to, now); err != nil {
				t.Fatalf("expected transition to succeed: %v", err)
			}
			if environment.Status != tt.to {
				t.Fatalf("expected status %s, got %s", tt.to, environment.Status)
			}
			if !environment.UpdatedAt.Equal(now) {
				t.Fatalf("expected updated time %s, got %s", now, environment.UpdatedAt)
			}
		})
	}
}

func TestEnvironmentTransitionToRejectsInvalidTransition(t *testing.T) {
	environment := Environment{Status: EnvironmentStatusPending}

	err := environment.TransitionTo(EnvironmentStatusReady, time.Now())

	assertInvalidTransition(t, err)
	if environment.Status != EnvironmentStatusPending {
		t.Fatalf("expected status to remain %s, got %s", EnvironmentStatusPending, environment.Status)
	}
}

func TestWorkflowTransitionToSetsLifecycleTimes(t *testing.T) {
	startedAt := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(time.Minute)
	workflow := Workflow{Status: WorkflowStatusPending}

	if err := workflow.TransitionTo(WorkflowStatusRunning, startedAt); err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	if err := workflow.TransitionTo(WorkflowStatusSucceeded, completedAt); err != nil {
		t.Fatalf("complete workflow: %v", err)
	}

	if workflow.StartedAt == nil || !workflow.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started time %s, got %v", startedAt, workflow.StartedAt)
	}
	if workflow.CompletedAt == nil || !workflow.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected completed time %s, got %v", completedAt, workflow.CompletedAt)
	}
}

func TestWorkflowTransitionToRejectsTerminalTransition(t *testing.T) {
	workflow := Workflow{Status: WorkflowStatusSucceeded}

	err := workflow.TransitionTo(WorkflowStatusRunning, time.Now())

	assertInvalidTransition(t, err)
}

func TestWorkflowStepTransitionTo(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	step := WorkflowStep{Status: StepStatusPending}

	if err := step.TransitionTo(StepStatusRunning, now); err != nil {
		t.Fatalf("start step: %v", err)
	}
	if step.StartedAt == nil || !step.StartedAt.Equal(now) {
		t.Fatalf("expected started time %s, got %v", now, step.StartedAt)
	}
}

func TestWorkflowStepTransitionToRejectsSkippingRunningStep(t *testing.T) {
	step := WorkflowStep{Status: StepStatusRunning}

	err := step.TransitionTo(StepStatusSkipped, time.Now())

	assertInvalidTransition(t, err)
}

func assertInvalidTransition(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	var transitionError *TransitionError
	if !errors.As(err, &transitionError) {
		t.Fatalf("expected TransitionError, got %T", err)
	}
}

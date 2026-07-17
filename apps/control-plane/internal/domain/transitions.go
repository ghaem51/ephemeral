package domain

import "time"

func (e *Environment) TransitionTo(next EnvironmentStatus, at time.Time) error {
	if !validEnvironmentTransition(e.Status, next) {
		return &TransitionError{Entity: "environment", From: string(e.Status), To: string(next)}
	}

	e.Status = next
	e.UpdatedAt = at
	return nil
}

func validEnvironmentTransition(from, to EnvironmentStatus) bool {
	switch from {
	case EnvironmentStatusPending:
		return to == EnvironmentStatusProvisioning || to == EnvironmentStatusFailed
	case EnvironmentStatusProvisioning:
		return to == EnvironmentStatusReady || to == EnvironmentStatusFailed
	case EnvironmentStatusReady:
		return to == EnvironmentStatusDestroying
	case EnvironmentStatusFailed:
		return to == EnvironmentStatusProvisioning || to == EnvironmentStatusDestroying
	case EnvironmentStatusDestroying:
		return to == EnvironmentStatusDestroyed || to == EnvironmentStatusFailed
	default:
		return false
	}
}

func (w *Workflow) TransitionTo(next WorkflowStatus, at time.Time) error {
	if !validWorkflowTransition(w.Status, next) {
		return &TransitionError{Entity: "workflow", From: string(w.Status), To: string(next)}
	}

	w.Status = next
	if next == WorkflowStatusRunning {
		w.StartedAt = timePointer(at)
	}
	if next == WorkflowStatusSucceeded || next == WorkflowStatusFailed {
		w.CompletedAt = timePointer(at)
	}
	return nil
}

func validWorkflowTransition(from, to WorkflowStatus) bool {
	switch from {
	case WorkflowStatusPending:
		return to == WorkflowStatusRunning
	case WorkflowStatusRunning:
		return to == WorkflowStatusSucceeded || to == WorkflowStatusFailed
	default:
		return false
	}
}

func (s *WorkflowStep) TransitionTo(next StepStatus, at time.Time) error {
	if !validStepTransition(s.Status, next) {
		return &TransitionError{Entity: "workflow step", From: string(s.Status), To: string(next)}
	}

	s.Status = next
	if next == StepStatusRunning {
		s.StartedAt = timePointer(at)
	}
	if next == StepStatusSucceeded || next == StepStatusFailed || next == StepStatusSkipped {
		s.CompletedAt = timePointer(at)
	}
	return nil
}

func validStepTransition(from, to StepStatus) bool {
	switch from {
	case StepStatusPending:
		return to == StepStatusRunning || to == StepStatusSkipped
	case StepStatusRunning:
		return to == StepStatusSucceeded || to == StepStatusFailed
	default:
		return false
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

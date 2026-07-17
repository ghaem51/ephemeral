package domain

import "time"

type Environment struct {
	ID            string
	Name          string
	Image         string
	ContainerPort int
	HostPort      int
	ContainerID   string
	URL           string
	Status        EnvironmentStatus
	ErrorMessage  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type EnvironmentSpec struct {
	ID            string
	Name          string
	Image         string
	ContainerPort int
}

type RuntimeInfo struct {
	ContainerID   string
	ContainerPort int
	HostPort      int
	URL           string
}

type Workflow struct {
	ID            string
	EnvironmentID string
	Operation     Operation
	Status        WorkflowStatus
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Steps         []WorkflowStep
}

type WorkflowStep struct {
	ID           string
	WorkflowID   string
	Name         string
	Order        int
	Status       StepStatus
	Message      string
	ErrorMessage string
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

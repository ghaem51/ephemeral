package repository

import (
	"context"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

type EnvironmentRepository interface {
	Create(context.Context, *domain.Environment) error
	Update(context.Context, *domain.Environment) error
	GetByID(context.Context, string) (*domain.Environment, error)
	List(context.Context) ([]domain.Environment, error)
	FindByName(context.Context, string) (*domain.Environment, error)
}

type WorkflowRepository interface {
	CreateWithSteps(context.Context, *domain.Workflow) error
	Update(context.Context, *domain.Workflow) error
	UpdateStep(context.Context, *domain.WorkflowStep) error
	GetLatestForEnvironment(context.Context, string) (*domain.Workflow, error)
	GetWithSteps(context.Context, string) (*domain.Workflow, error)
}

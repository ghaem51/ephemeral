package environmentapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/repository"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/createenvironment"
)

type Creator interface {
	Create(context.Context, createenvironment.Request) (*domain.Environment, error)
}

type Lifecycle interface {
	Destroy(context.Context, string) (*domain.Environment, error)
	Retry(context.Context, string) (*domain.Environment, error)
}

type Result struct {
	Environment domain.Environment
	Workflow    *domain.Workflow
}

type Service struct {
	creator      Creator
	lifecycle    Lifecycle
	environments repository.EnvironmentRepository
	workflows    repository.WorkflowRepository
}

func New(
	creator Creator,
	lifecycle Lifecycle,
	environments repository.EnvironmentRepository,
	workflows repository.WorkflowRepository,
) *Service {
	return &Service{creator: creator, lifecycle: lifecycle, environments: environments, workflows: workflows}
}

func (s *Service) Destroy(ctx context.Context, id string) (*Result, error) {
	environment, err := s.lifecycle.Destroy(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.result(ctx, environment)
}

func (s *Service) Retry(ctx context.Context, id string) (*Result, error) {
	environment, err := s.lifecycle.Retry(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.result(ctx, environment)
}

func (s *Service) Create(ctx context.Context, request createenvironment.Request) (*Result, error) {
	environment, err := s.creator.Create(ctx, request)
	if err != nil {
		return nil, err
	}
	return s.result(ctx, environment)
}

func (s *Service) List(ctx context.Context) ([]Result, error) {
	environments, err := s.environments.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	results := make([]Result, 0, len(environments))
	for index := range environments {
		result, err := s.result(ctx, &environments[index])
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	return results, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Result, error) {
	environment, err := s.environments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.result(ctx, environment)
}

func (s *Service) result(ctx context.Context, environment *domain.Environment) (*Result, error) {
	workflow, err := s.workflows.GetLatestForEnvironment(ctx, environment.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get latest workflow for environment %q: %w", environment.ID, err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		workflow = nil
	}
	return &Result{Environment: *environment, Workflow: workflow}, nil
}

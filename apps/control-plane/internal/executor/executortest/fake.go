package executortest

import (
	"context"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor"
)

var _ executor.EnvironmentExecutor = (*Fake)(nil)

// Fake provides operation hooks for workflow unit tests. Unset hooks succeed
// with zero values so each test only needs to configure behavior it cares about.
type Fake struct {
	CreateFunc      func(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error)
	StartFunc       func(context.Context, domain.RuntimeInfo) (domain.RuntimeInfo, error)
	CheckHealthFunc func(context.Context, domain.RuntimeInfo) error
	DestroyFunc     func(context.Context, domain.RuntimeInfo) error
}

func (f *Fake) Create(ctx context.Context, spec domain.EnvironmentSpec) (domain.RuntimeInfo, error) {
	if f.CreateFunc == nil {
		return domain.RuntimeInfo{}, nil
	}
	return f.CreateFunc(ctx, spec)
}

func (f *Fake) Start(ctx context.Context, runtime domain.RuntimeInfo) (domain.RuntimeInfo, error) {
	if f.StartFunc == nil {
		return runtime, nil
	}
	return f.StartFunc(ctx, runtime)
}

func (f *Fake) CheckHealth(ctx context.Context, runtime domain.RuntimeInfo) error {
	if f.CheckHealthFunc == nil {
		return nil
	}
	return f.CheckHealthFunc(ctx, runtime)
}

func (f *Fake) Destroy(ctx context.Context, runtime domain.RuntimeInfo) error {
	if f.DestroyFunc == nil {
		return nil
	}
	return f.DestroyFunc(ctx, runtime)
}

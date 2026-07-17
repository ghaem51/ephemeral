package executor

import (
	"context"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

// EnvironmentExecutor defines the runtime operations needed by EnvPilot's
// provisioning workflows. Implementations keep infrastructure-specific types
// behind this boundary.
type EnvironmentExecutor interface {
	Create(context.Context, domain.EnvironmentSpec) (domain.RuntimeInfo, error)
	Start(context.Context, domain.RuntimeInfo) (domain.RuntimeInfo, error)
	CheckHealth(context.Context, domain.RuntimeInfo) error
	Destroy(context.Context, domain.RuntimeInfo) error
}

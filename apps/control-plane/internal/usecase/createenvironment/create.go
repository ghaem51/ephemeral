package createenvironment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/executor"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/repository"
)

var environmentNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
var applicationVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var environmentVariableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const unhealthyDemoImage = "envpilot/demo-service:unhealthy"
const defaultHealthCheckPath = "/health"

const (
	StepValidateRequest = "VALIDATE_REQUEST"
	StepCreateContainer = "CREATE_CONTAINER"
	StepStartContainer  = "START_CONTAINER"
	StepCheckHealth     = "CHECK_HEALTH"
	StepMarkReady       = "MARK_READY"
)

type Request struct {
	Name                 string
	Image                string
	ContainerPort        int
	HealthCheckPath      string
	SimulateFailure      bool
	ApplicationVersion   string
	EnvironmentVariables []string
}

type UseCase struct {
	environments repository.EnvironmentRepository
	workflows    repository.WorkflowRepository
	executor     executor.EnvironmentExecutor
	now          func() time.Time
	newID        func() (string, error)
	logger       *slog.Logger

	mu      sync.Mutex
	started map[string]chan struct{}
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
		environments: environments,
		workflows:    workflows,
		executor:     runtimeExecutor,
		now:          func() time.Time { return time.Now().UTC() },
		newID:        randomID,
		logger:       logger,
		started:      make(map[string]chan struct{}),
	}
}

func (uc *UseCase) Create(ctx context.Context, request Request) (*domain.Environment, error) {
	spec, err := validate(request)
	if err != nil {
		return nil, err
	}

	existing, err := uc.environments.FindByName(ctx, spec.Name)
	if err == nil && existing.Status != domain.EnvironmentStatusDestroyed {
		return nil, fmt.Errorf("environment name %q: %w", spec.Name, domain.ErrAlreadyExists)
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("find environment by name: %w", err)
	}

	environmentID, err := uc.newID()
	if err != nil {
		return nil, fmt.Errorf("generate environment ID: %w", err)
	}
	workflowID, err := uc.newID()
	if err != nil {
		return nil, fmt.Errorf("generate workflow ID: %w", err)
	}
	spec.ID = environmentID

	now := uc.now()
	environment := &domain.Environment{
		ID: environmentID, Name: spec.Name, Image: spec.Image, ContainerPort: spec.ContainerPort,
		HealthCheckPath:      spec.HealthCheckPath,
		ApplicationVersion:   spec.ApplicationVersion,
		EnvironmentVariables: append([]string(nil), spec.EnvironmentVariables...),
		Status:               domain.EnvironmentStatusPending, CreatedAt: now, UpdatedAt: now,
	}
	workflow, err := uc.newWorkflow(workflowID, environmentID)
	if err != nil {
		return nil, err
	}

	if err := uc.environments.Create(ctx, environment); err != nil {
		return nil, fmt.Errorf("persist pending environment: %w", err)
	}
	if err := uc.workflows.CreateWithSteps(ctx, workflow); err != nil {
		return nil, fmt.Errorf("persist create workflow: %w", err)
	}

	// Background execution owns separate values so callers can safely inspect the
	// immediately returned PENDING environment without racing workflow updates.
	backgroundEnvironment := *environment
	backgroundWorkflow := *workflow
	backgroundWorkflow.Steps = append([]domain.WorkflowStep(nil), workflow.Steps...)
	uc.start(&backgroundEnvironment, &backgroundWorkflow, spec)

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

func (uc *UseCase) newWorkflow(id, environmentID string) (*domain.Workflow, error) {
	names := []string{
		StepValidateRequest,
		StepCreateContainer,
		StepStartContainer,
		StepCheckHealth,
		StepMarkReady,
	}
	steps := make([]domain.WorkflowStep, 0, len(names))
	for index, name := range names {
		stepID, err := uc.newID()
		if err != nil {
			return nil, fmt.Errorf("generate workflow step ID: %w", err)
		}
		steps = append(steps, domain.WorkflowStep{
			ID: stepID, WorkflowID: id, Name: name,
			Order: index + 1, Status: domain.StepStatusPending,
		})
	}
	return &domain.Workflow{
		ID: id, EnvironmentID: environmentID, Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusPending, Steps: steps,
	}, nil
}

func validate(request Request) (domain.EnvironmentSpec, error) {
	spec := domain.EnvironmentSpec{
		Name: strings.TrimSpace(request.Name), Image: strings.TrimSpace(request.Image),
		ContainerPort: request.ContainerPort, HealthCheckPath: strings.TrimSpace(request.HealthCheckPath),
		ApplicationVersion:   strings.TrimSpace(request.ApplicationVersion),
		EnvironmentVariables: append([]string(nil), request.EnvironmentVariables...),
	}
	if spec.Name == "" {
		return domain.EnvironmentSpec{}, fmt.Errorf("name is required: %w", domain.ErrValidation)
	}
	if len(spec.Name) > 63 {
		return domain.EnvironmentSpec{}, fmt.Errorf("name must be 63 characters or fewer: %w", domain.ErrValidation)
	}
	if !environmentNamePattern.MatchString(spec.Name) {
		return domain.EnvironmentSpec{}, fmt.Errorf("name must contain only lowercase letters, numbers, and hyphens, and start and end with a letter or number: %w", domain.ErrValidation)
	}
	if spec.Image == "" {
		return domain.EnvironmentSpec{}, fmt.Errorf("image is required: %w", domain.ErrValidation)
	}
	if request.SimulateFailure {
		spec.Image = unhealthyDemoImage
	}
	if spec.HealthCheckPath == "" {
		spec.HealthCheckPath = defaultHealthCheckPath
	}
	if len(spec.HealthCheckPath) > 255 {
		return domain.EnvironmentSpec{}, fmt.Errorf("health check path must be 255 characters or fewer: %w", domain.ErrValidation)
	}
	if !strings.HasPrefix(spec.HealthCheckPath, "/") || strings.ContainsAny(spec.HealthCheckPath, "?#") {
		return domain.EnvironmentSpec{}, fmt.Errorf("health check path must start with / and cannot contain a query or fragment: %w", domain.ErrValidation)
	}
	if spec.ApplicationVersion != "" && !applicationVersionPattern.MatchString(spec.ApplicationVersion) {
		return domain.EnvironmentSpec{}, fmt.Errorf("application version must be 1-64 letters, numbers, dots, underscores, or hyphens: %w", domain.ErrValidation)
	}
	if err := validateEnvironmentVariables(spec.EnvironmentVariables); err != nil {
		return domain.EnvironmentSpec{}, err
	}
	if spec.ContainerPort < 1 || spec.ContainerPort > 65535 {
		return domain.EnvironmentSpec{}, fmt.Errorf("container port must be between 1 and 65535: %w", domain.ErrValidation)
	}
	return spec, nil
}

func validateEnvironmentVariables(variables []string) error {
	if len(variables) > 100 {
		return fmt.Errorf("environment variables cannot contain more than 100 entries: %w", domain.ErrValidation)
	}
	seen := make(map[string]struct{}, len(variables))
	for _, variable := range variables {
		if len(variable) > 4096 || strings.ContainsRune(variable, '\x00') {
			return fmt.Errorf("environment variable entries must be 4096 characters or fewer and cannot contain null bytes: %w", domain.ErrValidation)
		}
		name, _, found := strings.Cut(variable, "=")
		if !found || !environmentVariableNamePattern.MatchString(name) {
			return fmt.Errorf("environment variable %q must use KEY=VALUE format with a valid name: %w", variable, domain.ErrValidation)
		}
		if name == "ENVIRONMENT_NAME" || name == "APP_VERSION" {
			return fmt.Errorf("environment variable %q is managed by EnvPilot: %w", name, domain.ErrValidation)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("environment variable %q is duplicated: %w", name, domain.ErrValidation)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

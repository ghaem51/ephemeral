package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/createenvironment"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/environmentapi"
)

type stubEnvironmentService struct {
	create func(context.Context, createenvironment.Request) (*environmentapi.Result, error)
	list   func(context.Context) ([]environmentapi.Result, error)
	get    func(context.Context, string) (*environmentapi.Result, error)
}

func (s *stubEnvironmentService) Create(ctx context.Context, request createenvironment.Request) (*environmentapi.Result, error) {
	return s.create(ctx, request)
}

func (s *stubEnvironmentService) List(ctx context.Context) ([]environmentapi.Result, error) {
	return s.list(ctx)
}

func (s *stubEnvironmentService) Get(ctx context.Context, id string) (*environmentapi.Result, error) {
	return s.get(ctx, id)
}

func TestCreateEnvironment(t *testing.T) {
	result := testEnvironmentResult()
	service := defaultStubService()
	service.create = func(_ context.Context, request createenvironment.Request) (*environmentapi.Result, error) {
		if request.Name != "feature-payment" || request.Image != "envpilot/demo-service:healthy" || request.ContainerPort != 8080 {
			t.Fatalf("unexpected create request: %#v", request)
		}
		if !request.SimulateFailure {
			t.Fatal("expected simulateFailure to be passed to the use case")
		}
		return &result, nil
	}
	router := NewRouter(NewEnvironmentHandler(service))
	body := `{"name":"feature-payment","image":"envpilot/demo-service:healthy","containerPort":8080,"simulateFailure":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/environments", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}
	var decoded environmentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.ID != result.Environment.ID || decoded.LatestWorkflow == nil {
		t.Fatalf("unexpected response: %#v", decoded)
	}
	if len(decoded.LatestWorkflow.Steps) != 2 || decoded.LatestWorkflow.Steps[0].Order != 1 {
		t.Fatalf("workflow steps missing or unordered: %#v", decoded.LatestWorkflow)
	}
}

func TestListEnvironments(t *testing.T) {
	result := testEnvironmentResult()
	service := defaultStubService()
	service.list = func(context.Context) ([]environmentapi.Result, error) {
		return []environmentapi.Result{result}, nil
	}
	response := httptest.NewRecorder()

	NewRouter(NewEnvironmentHandler(service)).ServeHTTP(
		response, httptest.NewRequest(http.MethodGet, "/api/v1/environments", nil),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var decoded []environmentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(decoded) != 1 || decoded[0].LatestWorkflow == nil {
		t.Fatalf("unexpected response: %#v", decoded)
	}
}

func TestGetEnvironment(t *testing.T) {
	result := testEnvironmentResult()
	service := defaultStubService()
	service.get = func(_ context.Context, id string) (*environmentapi.Result, error) {
		if id != result.Environment.ID {
			t.Fatalf("unexpected environment ID: %q", id)
		}
		return &result, nil
	}
	response := httptest.NewRecorder()

	NewRouter(NewEnvironmentHandler(service)).ServeHTTP(
		response, httptest.NewRequest(http.MethodGet, "/api/v1/environments/env-1", nil),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
}

func TestEnvironmentErrorsUseConsistentResponse(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: domain.ErrValidation, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "duplicate", err: domain.ErrAlreadyExists, wantStatus: http.StatusConflict, wantCode: "ENVIRONMENT_ALREADY_EXISTS"},
		{name: "not found", err: domain.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "ENVIRONMENT_NOT_FOUND"},
		{name: "internal", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := defaultStubService()
			service.get = func(context.Context, string) (*environmentapi.Result, error) { return nil, tt.err }
			request := httptest.NewRequest(http.MethodGet, "/api/v1/environments/env-1", nil)
			request.Header.Set("X-Request-ID", "request-123")
			response := httptest.NewRecorder()

			NewRouter(NewEnvironmentHandler(service)).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, response.Code)
			}
			var decoded errorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if decoded.Code != tt.wantCode || decoded.Message == "" || decoded.RequestID != "request-123" {
				t.Fatalf("unexpected error response: %#v", decoded)
			}
		})
	}
}

func TestInvalidJSONReturnsDetailsAndRequestID(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/environments", strings.NewReader(`{"name":`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	NewRouter(NewEnvironmentHandler(defaultStubService())).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	var decoded errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if decoded.Code != "INVALID_REQUEST" || decoded.Details == nil || decoded.RequestID == "" {
		t.Fatalf("unexpected error response: %#v", decoded)
	}
	if response.Header().Get("X-Request-ID") != decoded.RequestID {
		t.Fatal("response header and error request ID do not match")
	}
}

func TestDestroyAndRetryAreNotImplemented(t *testing.T) {
	router := NewRouter(NewEnvironmentHandler(defaultStubService()))
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodDelete, path: "/api/v1/environments/env-1"},
		{method: http.MethodPost, path: "/api/v1/environments/env-1/retry"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
		if response.Code != http.StatusNotImplemented {
			t.Fatalf("%s %s: expected 501, got %d", test.method, test.path, response.Code)
		}
	}
}

func defaultStubService() *stubEnvironmentService {
	return &stubEnvironmentService{
		create: func(context.Context, createenvironment.Request) (*environmentapi.Result, error) {
			result := testEnvironmentResult()
			return &result, nil
		},
		list: func(context.Context) ([]environmentapi.Result, error) { return []environmentapi.Result{}, nil },
		get: func(context.Context, string) (*environmentapi.Result, error) {
			result := testEnvironmentResult()
			return &result, nil
		},
	}
}

func testEnvironmentResult() environmentapi.Result {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)
	workflow := &domain.Workflow{
		ID: "workflow-1", EnvironmentID: "env-1", Operation: domain.OperationCreate,
		Status: domain.WorkflowStatusRunning, StartedAt: &now,
		Steps: []domain.WorkflowStep{
			{ID: "step-1", WorkflowID: "workflow-1", Name: "VALIDATE_REQUEST", Order: 1, Status: domain.StepStatusSucceeded},
			{ID: "step-2", WorkflowID: "workflow-1", Name: "CREATE_CONTAINER", Order: 2, Status: domain.StepStatusRunning},
		},
	}
	return environmentapi.Result{
		Environment: domain.Environment{
			ID: "env-1", Name: "feature-payment", Image: "envpilot/demo-service:healthy",
			ContainerPort: 8080, Status: domain.EnvironmentStatusProvisioning, CreatedAt: now, UpdatedAt: now,
		},
		Workflow: workflow,
	}
}

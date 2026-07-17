package server

import (
	"context"
	"net/http"
	"time"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/createenvironment"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/usecase/environmentapi"
	"github.com/gin-gonic/gin"
)

type EnvironmentService interface {
	Create(context.Context, createenvironment.Request) (*environmentapi.Result, error)
	List(context.Context) ([]environmentapi.Result, error)
	Get(context.Context, string) (*environmentapi.Result, error)
	Destroy(context.Context, string) (*environmentapi.Result, error)
	Retry(context.Context, string) (*environmentapi.Result, error)
}

type EnvironmentHandler struct {
	service EnvironmentService
}

func NewEnvironmentHandler(service EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{service: service}
}

type createEnvironmentRequest struct {
	Name               string `json:"name"`
	Image              string `json:"image"`
	ContainerPort      int    `json:"containerPort"`
	SimulateFailure    bool   `json:"simulateFailure"`
	ApplicationVersion string `json:"applicationVersion"`
}

func (h *EnvironmentHandler) Create(c *gin.Context) {
	var request createEnvironmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON", err.Error())
		return
	}

	result, err := h.service.Create(c.Request.Context(), createenvironment.Request{
		Name: request.Name, Image: request.Image, ContainerPort: request.ContainerPort,
		SimulateFailure:    request.SimulateFailure,
		ApplicationVersion: request.ApplicationVersion,
	})
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toEnvironmentResponse(result))
}

func (h *EnvironmentHandler) List(c *gin.Context) {
	results, err := h.service.List(c.Request.Context())
	if err != nil {
		writeDomainError(c, err)
		return
	}

	response := make([]environmentResponse, 0, len(results))
	for index := range results {
		response = append(response, toEnvironmentResponse(&results[index]))
	}
	c.JSON(http.StatusOK, response)
}

func (h *EnvironmentHandler) Get(c *gin.Context) {
	result, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toEnvironmentResponse(result))
}

func (h *EnvironmentHandler) Destroy(c *gin.Context) {
	result, err := h.service.Destroy(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toEnvironmentResponse(result))
}

func (h *EnvironmentHandler) Retry(c *gin.Context) {
	result, err := h.service.Retry(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toEnvironmentResponse(result))
}

type environmentResponse struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	ContainerPort  int               `json:"containerPort"`
	HostPort       int               `json:"hostPort"`
	ContainerID    string            `json:"containerId"`
	URL            string            `json:"url"`
	Status         string            `json:"status"`
	ErrorMessage   string            `json:"errorMessage,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	LatestWorkflow *workflowResponse `json:"latestWorkflow"`
}

type workflowResponse struct {
	ID            string         `json:"id"`
	EnvironmentID string         `json:"environmentId"`
	Operation     string         `json:"operation"`
	Status        string         `json:"status"`
	StartedAt     *time.Time     `json:"startedAt"`
	CompletedAt   *time.Time     `json:"completedAt"`
	Steps         []stepResponse `json:"steps"`
}

type stepResponse struct {
	ID           string     `json:"id"`
	WorkflowID   string     `json:"workflowId"`
	Name         string     `json:"name"`
	Order        int        `json:"order"`
	Status       string     `json:"status"`
	Message      string     `json:"message"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

func toEnvironmentResponse(result *environmentapi.Result) environmentResponse {
	environment := result.Environment
	response := environmentResponse{
		ID: environment.ID, Name: environment.Name, Image: environment.Image,
		ContainerPort: environment.ContainerPort, HostPort: environment.HostPort,
		ContainerID: environment.ContainerID, URL: environment.URL, Status: string(environment.Status),
		ErrorMessage: environment.ErrorMessage, CreatedAt: environment.CreatedAt, UpdatedAt: environment.UpdatedAt,
	}
	if result.Workflow != nil {
		response.LatestWorkflow = toWorkflowResponse(result.Workflow)
	}
	return response
}

func toWorkflowResponse(workflow *domain.Workflow) *workflowResponse {
	steps := make([]stepResponse, 0, len(workflow.Steps))
	for _, step := range workflow.Steps {
		steps = append(steps, stepResponse{
			ID: step.ID, WorkflowID: step.WorkflowID, Name: step.Name, Order: step.Order,
			Status: string(step.Status), Message: step.Message, ErrorMessage: step.ErrorMessage,
			StartedAt: step.StartedAt, CompletedAt: step.CompletedAt,
		})
	}
	return &workflowResponse{
		ID: workflow.ID, EnvironmentID: workflow.EnvironmentID, Operation: string(workflow.Operation),
		Status: string(workflow.Status), StartedAt: workflow.StartedAt, CompletedAt: workflow.CompletedAt,
		Steps: steps,
	}
}

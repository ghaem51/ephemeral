package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"requestId"`
}

func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "ENVIRONMENT_ALREADY_EXISTS", err.Error(), nil)
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "ENVIRONMENT_NOT_FOUND", "environment not found", nil)
	default:
		slog.Error("request failed", "request_id", requestID(c), "error", err)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", nil)
	}
}

func writeError(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, errorResponse{
		Code: code, Message: message, Details: details, RequestID: requestID(c),
	})
}

package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter(environments *EnvironmentHandler, loggers ...*slog.Logger) http.Handler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(requestIDMiddleware(), requestLoggingMiddleware(logger), gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("HTTP handler panic", "request_id", requestID(c), "method", c.Request.Method, "path", c.Request.URL.Path, "panic", recovered)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", nil)
	}))
	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "route not found", nil)
	})
	router.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if environments != nil {
		api := router.Group("/api/v1/environments")
		api.POST("", environments.Create)
		api.GET("", environments.List)
		api.GET("/:id", environments.Get)
		api.DELETE("/:id", environments.Destroy)
		api.POST("/:id/retry", environments.Retry)
	}

	return router
}

func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("HTTP request completed",
			"request_id", requestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(environments *EnvironmentHandler) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(requestIDMiddleware(), gin.CustomRecovery(func(c *gin.Context, _ any) {
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

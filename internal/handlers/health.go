package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db       *database.DB
	registry *policy.Registry
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, registry *policy.Registry) *HealthHandler {
	return &HealthHandler{
		db:       db,
		registry: registry,
	}
}

// Health returns service health status (lightweight, safe for production)
func (h *HealthHandler) Health(c *gin.Context) {
	version := "1.0.0"
	if h.registry != nil {
		version = h.registry.GetSnapshot().Version
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"service":        "codexsecure",
		"version":        version,
		"policy_version": policy.Version,
	})
}

// Ready checks if service is ready (DB connected)
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	
	if err := h.db.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "connected",
	})
}

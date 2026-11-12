package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoadWebRoutes is the Go equivalent of Laravel routes/web.php.
func LoadWebRoutes(r *gin.Engine, deps Dependencies) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "codexsecure",
			"version": deps.AppVersion,
			"status":  "operational",
			"docs":    "/api/v1/status",
		})
	})

	if deps.HealthHandler != nil {
		r.GET("/health", deps.HealthHandler.Health)
		r.GET("/ready", deps.HealthHandler.Ready)
	}

	if deps.MetricsHandler != nil {
		r.GET("/metrics", deps.MetricsHandler.GetMetrics)
	}
}

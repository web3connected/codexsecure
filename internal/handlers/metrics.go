package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	tracker *middleware.ServiceTracker
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(tracker *middleware.ServiceTracker) *MetricsHandler {
	return &MetricsHandler{tracker: tracker}
}

// GetMetrics returns service endpoint metrics
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours < 1 {
		hours = 24
	}

	ctx := c.Request.Context()
	metrics, err := h.tracker.GetMetrics(ctx, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

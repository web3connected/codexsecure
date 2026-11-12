package handlers

// AnalyzerHandler - HTTP handlers for route analysis
// Provides endpoints for route zone detection and batch analysis.
//
// @version 2.0.0

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
)

// AnalyzerHandler handles route analysis requests
type AnalyzerHandler struct {
	analyzer *analyzer.Analyzer
}

// NewAnalyzerHandler creates a new analyzer handler
func NewAnalyzerHandler(a *analyzer.Analyzer) *AnalyzerHandler {
	return &AnalyzerHandler{analyzer: a}
}

// AnalyzeRoutes handles batch route analysis
// POST /api/v1/analyzer/routes
func (h *AnalyzerHandler) AnalyzeRoutes(c *gin.Context) {
	var req analyzer.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Routes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "empty_routes",
			"message": "No routes provided for analysis",
		})
		return
	}

	result := h.analyzer.AnalyzeRoutes(req.Routes)

	c.JSON(http.StatusOK, result)
}

// AnalyzeSingleRoute handles single route analysis
// POST /api/v1/analyzer/route
func (h *AnalyzerHandler) AnalyzeSingleRoute(c *gin.Context) {
	var req analyzer.AnalyzeRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Build RouteInfo from request
	route := analyzer.RouteInfo{
		URI:        req.URI,
		Methods:    req.Methods,
		Middleware: req.Middleware,
		Action: analyzer.ActionInfo{
			Controller: req.Controller,
			Method:     req.Method,
			Type:       "controller",
		},
	}

	if req.Controller == "" {
		route.Action.Type = "closure"
	}

	analyzed := h.analyzer.AnalyzeRoute(route)

	c.JSON(http.StatusOK, gin.H{
		"uri":         analyzed.URI,
		"methods":     analyzed.Methods,
		"zone_id":     analyzed.ZoneID,
		"zone_name":   analyzed.ZoneName,
		"zone_key":    analyzed.ZoneKey,
		"score":       analyzed.Score,
		"match_details": gin.H{
			"uri_match":        analyzed.ZoneAssignment.Details.URIMatch,
			"controller_match": analyzed.ZoneAssignment.Details.ControllerMatch,
			"middleware_match": analyzed.ZoneAssignment.Details.MiddlewareMatch,
		},
	})
}

// DetectZone is a quick zone detection endpoint
// GET /api/v1/analyzer/detect?uri=/some/path&controller=UserController
func (h *AnalyzerHandler) DetectZone(c *gin.Context) {
	uri := c.Query("uri")
	controller := c.Query("controller")

	if uri == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_uri",
			"message": "URI parameter is required",
		})
		return
	}

	zoneKey := h.analyzer.DetectRouteZone(uri, controller)

	c.JSON(http.StatusOK, gin.H{
		"uri":        uri,
		"controller": controller,
		"zone_key":   zoneKey,
	})
}

// GetStrategies returns the current matching strategies
// GET /api/v1/analyzer/strategies
func (h *AnalyzerHandler) GetStrategies(c *gin.Context) {
	strategies := h.analyzer.GetStrategies()
	uriStrategies := h.analyzer.GetURIStrategies()

	c.JSON(http.StatusOK, gin.H{
		"uri_strategies":        uriStrategies,
		"controller_patterns":   strategies.Controller.Patterns,
		"middleware_strategies": strategies.Middleware.ZoneMiddleware,
	})
}

// CheckRouteInZone checks if a route belongs to a specific zone
// POST /api/v1/analyzer/check-zone
func (h *AnalyzerHandler) CheckRouteInZone(c *gin.Context) {
	var req struct {
		URI        string   `json:"uri" binding:"required"`
		Methods    []string `json:"methods"`
		Controller string   `json:"controller,omitempty"`
		Method     string   `json:"method,omitempty"`
		Middleware []string `json:"middleware,omitempty"`
		ZoneName   string   `json:"zone_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	route := analyzer.RouteInfo{
		URI:        req.URI,
		Methods:    req.Methods,
		Middleware: req.Middleware,
		Action: analyzer.ActionInfo{
			Controller: req.Controller,
			Method:     req.Method,
		},
	}

	isInZone := h.analyzer.IsRouteInZone(route, req.ZoneName)

	c.JSON(http.StatusOK, gin.H{
		"uri":        req.URI,
		"zone_name":  req.ZoneName,
		"is_in_zone": isInZone,
	})
}

// GetAnalysisStats returns analysis statistics summary
// POST /api/v1/analyzer/stats
func (h *AnalyzerHandler) GetAnalysisStats(c *gin.Context) {
	var req analyzer.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Routes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "empty_routes",
			"message": "No routes provided for analysis",
		})
		return
	}

	result := h.analyzer.AnalyzeRoutes(req.Routes)

	c.JSON(http.StatusOK, gin.H{
		"stats":       result.Stats,
		"zones_used":  len(result.Zones),
		"zone_counts": buildZoneCounts(result.Zones),
	})
}

// buildZoneCounts extracts zone counts from groups
func buildZoneCounts(zones map[string]analyzer.ZoneGroup) map[string]int {
	counts := make(map[string]int)
	for name, group := range zones {
		counts[name] = group.Count
	}
	return counts
}

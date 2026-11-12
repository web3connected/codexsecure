package handlers

// FingerprintHandler - HTTP handlers for request fingerprinting
// Provides endpoints for sync analysis and drift calculation.
//
// @version 2.0.0

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
)

// FingerprintHandler handles fingerprint analysis requests
type FingerprintHandler struct {
	analyzer *fingerprint.Analyzer
}

// NewFingerprintHandler creates a new fingerprint handler
func NewFingerprintHandler(a *fingerprint.Analyzer) *FingerprintHandler {
	return &FingerprintHandler{analyzer: a}
}

// Analyze performs full request analysis
// POST /api/v1/fingerprint/analyze
func (h *FingerprintHandler) Analyze(c *gin.Context) {
	var req fingerprint.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	result := h.analyzer.Analyze(req.Route, req.Input)

	c.JSON(http.StatusOK, result)
}

// GenerateRouteFingerprint generates a fingerprint for a route
// POST /api/v1/fingerprint/route
func (h *FingerprintHandler) GenerateRouteFingerprint(c *gin.Context) {
	var req fingerprint.FingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	fp := h.analyzer.GenerateRouteFingerprint(req.Route)

	c.JSON(http.StatusOK, gin.H{
		"fingerprint": fp,
		"route": gin.H{
			"uri":        req.Route.URI,
			"controller": req.Route.Controller,
			"method":     req.Route.Method,
		},
	})
}

// GenerateInputFingerprint generates a fingerprint for input data
// POST /api/v1/fingerprint/input
func (h *FingerprintHandler) GenerateInputFingerprint(c *gin.Context) {
	var req struct {
		Input  map[string]any `json:"input" binding:"required"`
		Filter bool           `json:"filter"` // Whether to apply intent filtering
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	var fp string
	var filterResult *fingerprint.FilterResult

	if req.Filter {
		result := fingerprint.FilterIntentFields(req.Input, h.analyzer.GetConfig())
		filterResult = &result
		fp = h.analyzer.GenerateInputFingerprint(result.Filtered)
	} else {
		fp = h.analyzer.GenerateInputFingerprint(req.Input)
	}

	response := gin.H{
		"fingerprint":    fp,
		"original_count": len(req.Input),
	}

	if filterResult != nil {
		response["filter_result"] = filterResult
	}

	c.JSON(http.StatusOK, response)
}

// CalculateDrift calculates drift between two fingerprints
// POST /api/v1/fingerprint/drift
func (h *FingerprintHandler) CalculateDrift(c *gin.Context) {
	var req fingerprint.DriftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	result := h.analyzer.CompareFingerprints(req.Fingerprint1, req.Fingerprint2)

	c.JSON(http.StatusOK, gin.H{
		"fingerprint1": req.Fingerprint1,
		"fingerprint2": req.Fingerprint2,
		"drift":        result.Drift,
		"sync_status":  result.SyncStatus,
		"threshold":    h.analyzer.GetConfig().TIUSyncThreshold,
	})
}

// FilterInput filters input to intent-driven fields
// POST /api/v1/fingerprint/filter
func (h *FingerprintHandler) FilterInput(c *gin.Context) {
	var req fingerprint.FilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	result := fingerprint.FilterIntentFields(req.Input, h.analyzer.GetConfig())

	c.JSON(http.StatusOK, result)
}

// GetConfig returns the current fingerprint configuration
// GET /api/v1/fingerprint/config
func (h *FingerprintHandler) GetConfig(c *gin.Context) {
	config := h.analyzer.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"tiu_sync_threshold": config.TIUSyncThreshold,
		"warning_threshold":  fingerprint.WarningThreshold,
		"intent_fields":      config.IntentFields,
		"noise_fields":       config.NoiseFields,
		"sync_statuses": gin.H{
			"perfect": "drift == 0.0",
			"good":    "drift <= tiu_sync_threshold",
			"warning": "drift <= 0.05",
			"error":   "drift > 0.05",
		},
	})
}

// QuickHash generates a quick fingerprint for any data
// POST /api/v1/fingerprint/hash
func (h *FingerprintHandler) QuickHash(c *gin.Context) {
	var req struct {
		Data any `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	fp := h.analyzer.QuickFingerprint(req.Data)

	c.JSON(http.StatusOK, gin.H{
		"fingerprint": fp,
	})
}

// BatchAnalyze analyzes multiple route/input pairs
// POST /api/v1/fingerprint/batch
func (h *FingerprintHandler) BatchAnalyze(c *gin.Context) {
	var req struct {
		Items []fingerprint.AnalyzeRequest `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "empty_items",
			"message": "No items provided for analysis",
		})
		return
	}

	results := make([]fingerprint.AnalysisResult, 0, len(req.Items))
	stats := struct {
		Total   int            `json:"total"`
		Perfect int            `json:"perfect"`
		Good    int            `json:"good"`
		Warning int            `json:"warning"`
		Error   int            `json:"error"`
		ByRoute map[string]int `json:"by_route"`
	}{
		Total:   len(req.Items),
		ByRoute: make(map[string]int),
	}

	for _, item := range req.Items {
		result := h.analyzer.Analyze(item.Route, item.Input)
		results = append(results, result)

		// Update stats
		switch result.SyncStatus {
		case fingerprint.SyncStatusPerfect:
			stats.Perfect++
		case fingerprint.SyncStatusGood:
			stats.Good++
		case fingerprint.SyncStatusWarning:
			stats.Warning++
		case fingerprint.SyncStatusError:
			stats.Error++
		}

		stats.ByRoute[result.Details.Route]++
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"stats":   stats,
	})
}

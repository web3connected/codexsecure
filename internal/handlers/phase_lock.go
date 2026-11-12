package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/internal/schema"
)

// PhaseLockHandler exposes explicit Phase Lock recording endpoints.
//
// This is intended for applications that want CodexSecure to evaluate and record
// an application's page/route intent + the user's submitted data pattern, then
// bind the resulting fingerprints and score to the resolved zone (e.g. Z2 identity).
//
// It does not store raw secrets.
type PhaseLockHandler struct {
	routeAnalyzer *analyzer.Analyzer
	fp            *fingerprint.Analyzer
	schemaReg     *schema.Registry
}

func NewPhaseLockHandler(routeAnalyzer *analyzer.Analyzer, fp *fingerprint.Analyzer, schemaReg *schema.Registry) *PhaseLockHandler {
	return &PhaseLockHandler{routeAnalyzer: routeAnalyzer, fp: fp, schemaReg: schemaReg}
}

type phaseLockRecordRequest struct {
	Method  string         `json:"method" binding:"required"`
	Path    string         `json:"path" binding:"required"`
	Payload map[string]any `json:"payload"`
}

// Record evaluates a Phase Lock snapshot and returns the normalized event.
// POST /api/v1/phase-lock/record
func (h *PhaseLockHandler) Record(c *gin.Context) {
	var req phaseLockRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	zoneKey := ""
	zoneID := analyzer.UnassignedZoneID
	zoneSource := ""
	zoneConfidence := 0.0
	if h.routeAnalyzer != nil {
		analysis := h.routeAnalyzer.AnalyzeRoute(analyzer.RouteInfo{URI: path, Methods: []string{method}})
		zoneKey = analysis.ZoneKey
		zoneID = analysis.ZoneID
		zoneSource = analysis.Source
		zoneConfidence = analysis.Confidence
	}

	reqID := middleware.GetRequestID(c)
	if reqID == "" {
		reqID = c.GetHeader(middleware.HeaderRequestID)
	}

	event := middleware.ComputePhaseLockEvent(
		h.fp,
		h.schemaReg,
		reqID,
		zoneKey,
		zoneID,
		method,
		path,
		"phase_lock_record",
		req.Payload,
		http.StatusOK,
	)

	c.JSON(http.StatusOK, gin.H{
		"event": event,
		"zone_assignment": gin.H{
			"zone_id":     zoneID,
			"zone_key":    zoneKey,
			"source":      zoneSource,
			"confidence":  zoneConfidence,
			"expected_z2": zoneID == 2 || zoneKey == "identity",
		},
	})
}

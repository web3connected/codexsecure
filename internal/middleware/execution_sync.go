package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextKeyCodexRequestID     = "codex_request_id"
	ContextKeyCodexTraceID       = "codex_trace_id"
	ContextKeyCodexCorrelationID = "codex_correlation_id"
	ContextKeyCodexExecutionReport = "codex_execution_report"

	HeaderTraceID         = "X-Codex-Trace-ID"
	HeaderCorrelationID   = "X-Codex-Correlation-ID"
	HeaderPipelineVersion = "X-Codex-Pipeline-Version"
	HeaderZoneTier        = "X-Codex-Zone-Tier"
	HeaderResult          = "X-Codex-Result"
	HeaderStage           = "X-Codex-Stage"
	HeaderRecordedAt      = "X-Codex-Recorded-At"

	PipelineVersion = "2.0.0"
)

// ExecutionSyncReport captures the finalized Stage 7 middleware cycle for a request.
type ExecutionSyncReport struct {
	RequestID       string `json:"request_id"`
	TraceID         string `json:"trace_id"`
	CorrelationID   string `json:"correlation_id"`
	PipelineVersion string `json:"pipeline_version"`
	RecordedAt      string `json:"recorded_at"`
	DurationMS      float64 `json:"duration_ms"`
	Accepted        bool   `json:"accepted"`
	FinalStatus     string `json:"final_status"`
	FinalStage      int    `json:"final_stage"`
	FinalStageName  string `json:"final_stage_name"`
	StatusCode      int    `json:"status_code"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	ZoneKey         string `json:"zone_key"`
	ZoneName        string `json:"zone_name"`
	ZoneTier        string `json:"zone_tier"`
	ZoneSource      string `json:"zone_source"`
	ZoneConfidence  float64 `json:"zone_confidence"`
	RecordedVia     string `json:"recorded_via"`
}

// ExecutionSyncMiddleware is the Go Stage 7 response harmonizer.
// It stamps request/trace metadata, final zone headers, and a concrete Stage 7 execution report.
func ExecutionSyncMiddleware(appVersion string) gin.HandlerFunc {
	return func(c *gin.Context) {
		recordedAt := time.Now().UTC()
		requestID := firstNonEmpty(
			c.GetHeader(HeaderRequestID),
			getStringValue(c, ContextKeyCodexRequestID),
			GetRequestID(c),
			uuid.NewString(),
		)
		traceID := firstNonEmpty(
			c.GetHeader(HeaderTraceID),
			getStringValue(c, ContextKeyCodexTraceID),
			requestID,
		)
		correlationID := firstNonEmpty(
			c.GetHeader(HeaderCorrelationID),
			getStringValue(c, ContextKeyCodexCorrelationID),
			traceID,
		)

		c.Set("request_id", requestID)
		c.Set(ContextKeyCodexRequestID, requestID)
		c.Set(ContextKeyCodexTraceID, traceID)
		c.Set(ContextKeyCodexCorrelationID, correlationID)

		startedAt := time.Now().UTC()
		c.Next()

		zoneKey := GetZoneKey(c)
		if zoneKey == "" {
			zoneKey = "unassigned"
		}

		zoneTier := GetZoneTier(c)
		if zoneTier == "" {
			zoneTier = "unknown"
		}

		statusCode := c.Writer.Status()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		durationMS := float64(time.Since(startedAt).Microseconds()) / 1000
		accepted, finalStatus := deriveExecutionOutcome(statusCode)
		report := ExecutionSyncReport{
			RequestID:       requestID,
			TraceID:         traceID,
			CorrelationID:   correlationID,
			PipelineVersion: PipelineVersion,
			RecordedAt:      recordedAt.Format(time.RFC3339),
			DurationMS:      durationMS,
			Accepted:        accepted,
			FinalStatus:     finalStatus,
			FinalStage:      7,
			FinalStageName:  "execution_sync_middleware",
			StatusCode:      statusCode,
			Method:          c.Request.Method,
			Path:            firstNonEmpty(c.FullPath(), c.Request.URL.Path),
			ZoneKey:         zoneKey,
			ZoneName:        GetZoneName(c),
			ZoneTier:        zoneTier,
			ZoneSource:      GetZoneSource(c),
			ZoneConfidence:  GetZoneConfidence(c),
			RecordedVia:     "middleware",
		}
		c.Set(ContextKeyCodexExecutionReport, report)

		applyExecutionResponseRules(c, zoneTier, statusCode)

		c.Header(HeaderRequestID, requestID)
		c.Header(HeaderTraceID, traceID)
		c.Header(HeaderCorrelationID, correlationID)
		c.Header(HeaderPipelineVersion, PipelineVersion)
		c.Header(HeaderZoneTier, zoneTier)
		c.Header("X-Codex-Zone", zoneKey)
		if report.ZoneSource != "" {
			c.Header("X-Codex-Zone-Source", report.ZoneSource)
		}
		c.Header("X-Codex-Zone-Confidence", fmt.Sprintf("%.2f", report.ZoneConfidence))
		c.Header(HeaderResult, finalStatus)
		c.Header(HeaderStage, report.FinalStageName)
		c.Header(HeaderRecordedAt, report.RecordedAt)
		c.Header(HeaderExecutionTime, fmt.Sprintf("%.2fms", durationMS))

		if appVersion != "" {
			c.Header(HeaderVersion, appVersion)
		}
	}
}

// GetExecutionSyncReport retrieves the finalized Stage 7 middleware report from context.
func GetExecutionSyncReport(c *gin.Context) (ExecutionSyncReport, bool) {
	if value, exists := c.Get(ContextKeyCodexExecutionReport); exists {
		if report, ok := value.(ExecutionSyncReport); ok {
			return report, true
		}
	}
	return ExecutionSyncReport{}, false
}

func deriveExecutionOutcome(statusCode int) (bool, string) {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return true, "accepted"
	case statusCode == http.StatusTooManyRequests:
		return false, "throttled"
	case statusCode >= 400 && statusCode < 500:
		return false, "rejected"
	default:
		return false, "failed"
	}
}

func applyExecutionResponseRules(c *gin.Context, zoneTier string, statusCode int) {
	if zoneTier == "critical" || statusCode >= 400 {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
}

func getStringValue(c *gin.Context, key string) string {
	if value, exists := c.Get(key); exists {
		if typed, ok := value.(string); ok {
			return typed
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
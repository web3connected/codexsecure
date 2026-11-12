package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCreateExecutionRecordsViaPersistenceSeam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedReq *PipelineExecutionRequest
	var capturedRecordedAt time.Time

	handler := &PipelineHandler{
		persistExecutionFn: func(_ context.Context, req *PipelineExecutionRequest, recordedAt time.Time) (*pipelinePersistenceResult, error) {
			capturedReq = req
			capturedRecordedAt = recordedAt

			resonanceLogID := int64(7002)
			tiuEventID := int64(7003)

			return &pipelinePersistenceResult{
				RecordID:       "stage7:req-123:7001",
				ValidationID:   7001,
				ResonanceLogID: &resonanceLogID,
				TIUEventID:     &tiuEventID,
			}, nil
		},
	}

	recorder := httptest.NewRecorder()
	ctx, router := buildPipelineTestContext(recorder, handler)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/executions", bytes.NewReader(mustJSON(t, validPipelineExecutionPayload())))
	ctx.Request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, ctx.Request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	if capturedReq == nil {
		t.Fatal("expected persistence seam to receive request")
	}

	if capturedReq.RequestID != "req-123" {
		t.Fatalf("expected request_id req-123, got %q", capturedReq.RequestID)
	}

	if capturedReq.Zone.ZoneID == nil || *capturedReq.Zone.ZoneID != 6 {
		t.Fatalf("expected zone_id 6, got %#v", capturedReq.Zone.ZoneID)
	}

	if capturedRecordedAt.Format(time.RFC3339) != "2026-04-10T16:30:00Z" {
		t.Fatalf("unexpected recorded_at %s", capturedRecordedAt.Format(time.RFC3339))
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response["status"] != "recorded" {
		t.Fatalf("expected recorded status, got %#v", response["status"])
	}

	if response["record_id"] != "stage7:req-123:7001" {
		t.Fatalf("expected record_id stage7:req-123:7001, got %#v", response["record_id"])
	}

	if response["persistence_owner"] != "codexsecure_api" {
		t.Fatalf("expected persistence_owner codexsecure_api, got %#v", response["persistence_owner"])
	}
}

func TestCreateExecutionReturnsPersistenceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &PipelineHandler{
		persistExecutionFn: func(_ context.Context, _ *PipelineExecutionRequest, _ time.Time) (*pipelinePersistenceResult, error) {
			return nil, errors.New("write failed")
		},
	}

	recorder := httptest.NewRecorder()
	ctx, router := buildPipelineTestContext(recorder, handler)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/executions", bytes.NewReader(mustJSON(t, validPipelineExecutionPayload())))
	ctx.Request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, ctx.Request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response["error"] != "persistence_failed" {
		t.Fatalf("expected persistence_failed error, got %#v", response["error"])
	}
}

func buildPipelineTestContext(recorder *httptest.ResponseRecorder, handler *PipelineHandler) (*gin.Context, *gin.Engine) {
	router := gin.New()
	router.POST("/api/v1/pipeline/executions", handler.CreateExecution)
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx, router
}

func validPipelineExecutionPayload() map[string]any {
	return map[string]any{
		"request_id":       "req-123",
		"trace_id":         "trace-123",
		"correlation_id":   "corr-123",
		"pipeline_version": "2.0.0",
		"recorded_at":      "2026-04-10T16:30:00Z",
		"source": map[string]any{
			"source_type":  "http",
			"transport":    "laravel",
			"service":      "codex_admin",
			"environment":  "testing",
		},
		"actor": map[string]any{
			"tenant_id":      10,
			"user_id":        501,
			"actor_type":     "user",
			"device_id":      "device-123",
			"session_id":     "session-123",
			"authenticated":  false,
		},
		"request": map[string]any{
			"method":           "POST",
			"path":             "/api/v1/demo",
			"route_name":       "demo.route",
			"ip":               "127.0.0.1",
			"user_agent":       "go-test",
			"fingerprint":      "fingerprint-123",
			"status_code":      403,
			"response_time_ms": 42.5,
		},
		"classification": map[string]any{
			"intent":         "payments",
			"pattern":        "payment_flow",
			"confidence":     91,
			"sensitivity":    "high",
			"risk":           "elevated",
			"traits":         []string{"financial_operation"},
			"zone_candidate": "Z6",
		},
		"coherence": map[string]any{
			"valid":            false,
			"tiu_valid":        false,
			"trust_valid":      false,
			"ethos_valid":      true,
			"method_valid":     true,
			"replay":           false,
			"rejection_reason": "policy_rejected",
		},
		"zone": map[string]any{
			"zone_id":         6,
			"zone_key":        "Z6",
			"zone_name":       "Payments",
			"zone_tier":       "critical",
			"zone_source":     "stage6_alignment",
			"zone_candidate":  "Z6",
		},
		"execution": map[string]any{
			"accepted":             false,
			"final_stage":          7,
			"final_stage_name":     "Execute",
			"final_status":         "rejected",
			"zone_token_generated": false,
			"zone_token_expires_at": "",
		},
		"delivery": map[string]any{
			"recorded_via":     "api",
			"attempt":          1,
			"origin_service":   "codex_admin",
			"origin_component": "CodexExecutionSyncMiddleware",
		},
		"artifacts": map[string]any{
			"fingerprint_drift":     0.42,
			"fingerprint_tolerance": 0.25,
			"tiu":                   0.75,
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return encoded
}
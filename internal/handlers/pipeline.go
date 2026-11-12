package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/internal/schema"
	"github.com/web3connected/codexsecure-api/pkg/mcp"
)

type PipelineHandler struct {
	db                  *database.DB
	apiToken            string
	mcpClient           *mcp.Client
	routeAnalyzer       *analyzer.Analyzer
	fingerprintAnalyzer *fingerprint.Analyzer
	policyRegistry      *policy.Registry
	schemaRegistry      *schema.Registry
	persistExecutionFn  func(context.Context, *PipelineExecutionRequest, time.Time) (*pipelinePersistenceResult, error)
}

func NewPipelineHandler(
	db *database.DB,
	apiToken string,
	mcpClient *mcp.Client,
	routeAnalyzer *analyzer.Analyzer,
	fingerprintAnalyzer *fingerprint.Analyzer,
	policyRegistry *policy.Registry,
	schemaRegistry *schema.Registry,
) *PipelineHandler {
	return &PipelineHandler{
		db:                  db,
		apiToken:            strings.TrimSpace(apiToken),
		mcpClient:           mcpClient,
		routeAnalyzer:       routeAnalyzer,
		fingerprintAnalyzer: fingerprintAnalyzer,
		policyRegistry:      policyRegistry,
		schemaRegistry:      schemaRegistry,
		persistExecutionFn:  nil,
	}
}

type PipelineExecutionRequest struct {
	RequestID       string                    `json:"request_id"`
	TraceID         string                    `json:"trace_id"`
	CorrelationID   string                    `json:"correlation_id"`
	PipelineVersion string                    `json:"pipeline_version"`
	RecordedAt      string                    `json:"recorded_at"`
	Source          PipelineSourceEnvelope    `json:"source"`
	Actor           PipelineActorEnvelope     `json:"actor"`
	Request         PipelineRequestEnvelope   `json:"request"`
	Classification  PipelineClassification    `json:"classification"`
	Coherence       PipelineCoherenceEnvelope `json:"coherence"`
	Zone            PipelineZoneEnvelope      `json:"zone"`
	Execution       PipelineExecutionEnvelope `json:"execution"`
	Delivery        PipelineDeliveryEnvelope  `json:"delivery"`
	Artifacts       map[string]any            `json:"artifacts"`
}

type PipelineSourceEnvelope struct {
	SourceType  string `json:"source_type"`
	Transport   string `json:"transport"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
}

type PipelineActorEnvelope struct {
	TenantID      any    `json:"tenant_id"`
	UserID        *int64 `json:"user_id"`
	ActorType     string `json:"actor_type"`
	DeviceID      string `json:"device_id"`
	SessionID     string `json:"session_id"`
	Authenticated bool   `json:"authenticated"`
}

type PipelineRequestEnvelope struct {
	Method         string  `json:"method"`
	Path           string  `json:"path"`
	RouteName      string  `json:"route_name"`
	IP             string  `json:"ip"`
	UserAgent      string  `json:"user_agent"`
	Fingerprint    string  `json:"fingerprint"`
	StatusCode     int     `json:"status_code"`
	ResponseTimeMS float64 `json:"response_time_ms"`
}

type PipelineClassification struct {
	Intent        string   `json:"intent"`
	Pattern       string   `json:"pattern"`
	Confidence    int      `json:"confidence"`
	Sensitivity   string   `json:"sensitivity"`
	Risk          string   `json:"risk"`
	Traits        []string `json:"traits"`
	ZoneCandidate string   `json:"zone_candidate"`
}

type PipelineCoherenceEnvelope struct {
	Valid           bool   `json:"valid"`
	TIUValid        *bool  `json:"tiu_valid"`
	TrustValid      *bool  `json:"trust_valid"`
	EthosValid      *bool  `json:"ethos_valid"`
	MethodValid     *bool  `json:"method_valid"`
	Replay          bool   `json:"replay"`
	RejectionReason string `json:"rejection_reason"`
}

type PipelineZoneEnvelope struct {
	ZoneID        *int   `json:"zone_id"`
	ZoneKey       string `json:"zone_key"`
	ZoneName      string `json:"zone_name"`
	ZoneTier      string `json:"zone_tier"`
	ZoneSource    string `json:"zone_source"`
	ZoneCandidate string `json:"zone_candidate"`
}

type PipelineExecutionEnvelope struct {
	Accepted           bool   `json:"accepted"`
	FinalStage         int    `json:"final_stage"`
	FinalStageName     string `json:"final_stage_name"`
	FinalStatus        string `json:"final_status"`
	ZoneTokenGenerated bool   `json:"zone_token_generated"`
	ZoneTokenExpiresAt string `json:"zone_token_expires_at"`
}

type PipelineDeliveryEnvelope struct {
	RecordedVia     string `json:"recorded_via"`
	Attempt         int    `json:"attempt"`
	OriginService   string `json:"origin_service"`
	OriginComponent string `json:"origin_component"`
}

func (h *PipelineHandler) CreateExecution(c *gin.Context) {
	if !h.authorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid Stage 7 API token",
		})
		return
	}

	var req PipelineExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := validatePipelineExecutionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	recordedAt, err := parseTimestamp(req.RecordedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_recorded_at",
			"message": err.Error(),
		})
		return
	}

	_, err = parseOptionalTimestamp(req.Execution.ZoneTokenExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_zone_token_expires_at",
			"message": err.Error(),
		})
		return
	}

	if err := h.validateAcceptedExecution(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "execution_policy_violation",
			"message": err.Error(),
		})
		return
	}

	requestID := strings.TrimSpace(req.RequestID)
	persistExecution := h.persistExecutionFn
	if persistExecution == nil {
		persistExecution = h.persistExecution
	}

	persistenceResult, err := persistExecution(c.Request.Context(), &req, recordedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "persistence_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":            "recorded",
		"request_id":        requestID,
		"record_id":         persistenceResult.RecordID,
		"validation_id":     persistenceResult.ValidationID,
		"resonance_log_id":  persistenceResult.ResonanceLogID,
		"tiu_event_id":      persistenceResult.TIUEventID,
		"trace_id":          strings.TrimSpace(req.TraceID),
		"accepted":          req.Execution.Accepted,
		"final_status":      strings.TrimSpace(req.Execution.FinalStatus),
		"validation_role":   "stage7_recorder",
		"persistence_owner": "codexsecure_api",
		"recorded_at":       recordedAt.UTC().Format(time.RFC3339),
		"zone_key":          strings.TrimSpace(req.Zone.ZoneKey),
	})
}

type pipelinePersistenceResult struct {
	RecordID       string
	ValidationID   int64
	ResonanceLogID *int64
	TIUEventID     *int64
}

func (h *PipelineHandler) persistExecution(ctx context.Context, req *PipelineExecutionRequest, recordedAt time.Time) (*pipelinePersistenceResult, error) {
	if h.db == nil || h.db.Pool == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	inputSnapshot, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input snapshot: %w", err)
	}

	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin stage7 transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	zoneID := 0
	if req.Zone.ZoneID != nil {
		zoneID = *req.Zone.ZoneID
	}

	status := strings.ToLower(strings.TrimSpace(req.Execution.FinalStatus))
	if status == "" {
		status = "pending"
	}

	hashValue := fingerprintHashValue(req.Request.Fingerprint)
	distance := extractArtifactFloat(req.Artifacts, []string{"fingerprint_drift", "drift", "distance"})
	tolerance := extractArtifactFloat(req.Artifacts, []string{"fingerprint_tolerance", "tolerance"})
	tiuValue := extractTIUValue(req)

	var validationID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO zone_validations (
			zone_id, agent_id, status, hash_value, distance, tolerance, tiu, input_snapshot, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
		RETURNING id
	`, zoneID, nil, status, hashValue, distance, tolerance, tiuValue, string(inputSnapshot), recordedAt, recordedAt).Scan(&validationID)
	if err != nil {
		return nil, fmt.Errorf("insert zone_validation: %w", err)
	}

	result := &pipelinePersistenceResult{
		RecordID:     fmt.Sprintf("stage7:%s:%d", strings.TrimSpace(req.RequestID), validationID),
		ValidationID: validationID,
	}

	if req.Execution.Accepted && strings.EqualFold(status, "accepted") {
		resonanceContext, err := json.Marshal(map[string]any{
			"request_id":    req.RequestID,
			"trace_id":      req.TraceID,
			"route_name":    req.Request.RouteName,
			"source":        req.Source,
			"classification": req.Classification,
			"coherence":     req.Coherence,
			"delivery":      req.Delivery,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal resonance context: %w", err)
		}

		var resonanceLogID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO resonance_logs (
				zone_id, agent_id, type, value, context, detected_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)
			RETURNING id
		`, zoneID, nil, "stage7_execution", float64(req.Classification.Confidence), string(resonanceContext), recordedAt, recordedAt, recordedAt).Scan(&resonanceLogID)
		if err != nil {
			return nil, fmt.Errorf("insert resonance_log: %w", err)
		}
		result.ResonanceLogID = &resonanceLogID
	}

	if req.Coherence.TIUValid != nil {
		tiuPayload, err := json.Marshal(map[string]any{
			"request_id": req.RequestID,
			"trace_id":   req.TraceID,
			"artifacts":  req.Artifacts,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal tiu payload: %w", err)
		}

		tiuResultJSON, err := json.Marshal(map[string]any{
			"valid":            *req.Coherence.TIUValid,
			"trust_valid":      req.Coherence.TrustValid,
			"rejection_reason": req.Coherence.RejectionReason,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal tiu result: %w", err)
		}

		var tiuEventID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO tiu_events (
				zone_id, agent_id, zone, event_type, tiu, payload, result, detected_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8, $9, $10)
			RETURNING id
		`, zoneID, nil, strings.TrimSpace(req.Zone.ZoneKey), "stage7_execution", tiuValue, string(tiuPayload), string(tiuResultJSON), recordedAt, recordedAt, recordedAt).Scan(&tiuEventID)
		if err != nil {
			return nil, fmt.Errorf("insert tiu_event: %w", err)
		}
		result.TIUEventID = &tiuEventID
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit stage7 transaction: %w", err)
	}

	return result, nil
}

func (h *PipelineHandler) authorized(c *gin.Context) bool {
	if h.apiToken == "" {
		return true
	}

	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return false
	}

	token := strings.TrimSpace(authorization[7:])
	return token == h.apiToken
}

func validatePipelineExecutionRequest(req *PipelineExecutionRequest) error {
	if strings.TrimSpace(req.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if strings.TrimSpace(req.TraceID) == "" {
		return fmt.Errorf("trace_id is required")
	}
	if strings.TrimSpace(req.PipelineVersion) == "" {
		return fmt.Errorf("pipeline_version is required")
	}
	if strings.TrimSpace(req.RecordedAt) == "" {
		return fmt.Errorf("recorded_at is required")
	}
	if strings.TrimSpace(req.Source.SourceType) == "" {
		return fmt.Errorf("source.source_type is required")
	}
	if strings.TrimSpace(req.Source.Service) == "" {
		return fmt.Errorf("source.service is required")
	}
	if strings.TrimSpace(req.Request.Method) == "" {
		return fmt.Errorf("request.method is required")
	}
	if strings.TrimSpace(req.Request.Path) == "" {
		return fmt.Errorf("request.path is required")
	}
	if strings.TrimSpace(req.Request.Fingerprint) == "" {
		return fmt.Errorf("request.fingerprint is required")
	}
	if strings.TrimSpace(req.Classification.Intent) == "" {
		return fmt.Errorf("classification.intent is required")
	}
	if strings.TrimSpace(req.Classification.Pattern) == "" {
		return fmt.Errorf("classification.pattern is required")
	}
	if strings.TrimSpace(req.Classification.Sensitivity) == "" {
		return fmt.Errorf("classification.sensitivity is required")
	}
	if strings.TrimSpace(req.Classification.Risk) == "" {
		return fmt.Errorf("classification.risk is required")
	}
	if req.Zone.ZoneID == nil {
		return fmt.Errorf("zone.zone_id is required")
	}
	if strings.TrimSpace(req.Zone.ZoneKey) == "" {
		return fmt.Errorf("zone.zone_key is required")
	}
	if strings.TrimSpace(req.Zone.ZoneTier) == "" {
		return fmt.Errorf("zone.zone_tier is required")
	}
	if req.Execution.FinalStage == 0 {
		return fmt.Errorf("execution.final_stage is required")
	}
	if strings.TrimSpace(req.Execution.FinalStatus) == "" {
		return fmt.Errorf("execution.final_status is required")
	}
	if math.IsNaN(req.Request.ResponseTimeMS) || math.IsInf(req.Request.ResponseTimeMS, 0) {
		return fmt.Errorf("request.response_time_ms must be a finite number")
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err == nil {
		return parsed.UTC(), nil
	}

	parsed, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err == nil {
		return parsed.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("timestamp must be RFC3339")
}

func parseOptionalTimestamp(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parsed, err := parseTimestamp(value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func (h *PipelineHandler) validateAcceptedExecution(req *PipelineExecutionRequest) error {
	finalStatus := strings.ToLower(strings.TrimSpace(req.Execution.FinalStatus))
	if finalStatus != "accepted" && !req.Execution.Accepted {
		return nil
	}

	zoneDefinition, err := h.resolveExecutionZone(req)
	if err != nil {
		return err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Request.Method))
	authValid := !zoneDefinition.Rules.AuthRequired || req.Actor.Authenticated
	tiuValid := !zoneDefinition.Rules.TIURequired || boolPointerTrue(req.Coherence.TIUValid)
	trustValid := trustLevelValid(zoneDefinition, Stage3ContextResult{TrustLevel: inferExecutionTrustLevel(req)})
	methodValid := h.policyRegistry == nil || h.policyRegistry.IsMethodAllowed(zoneDefinition.ID, method)
	executiveValid := !zoneDefinition.Rules.Executive || executionHasExecutiveAccess(req)
	mfaRequired := zoneDefinition.Rules.Executive && inferAction(method) != "read"
	mfaValid := !mfaRequired || executionHasMFAProof(req.Artifacts)

	if !req.Coherence.Valid {
		return fmt.Errorf("accepted execution requires coherence.valid=true")
	}
	if !authValid {
		return fmt.Errorf("accepted execution requires authenticated actor for zone %s", zoneDefinition.Key)
	}
	if !tiuValid {
		return fmt.Errorf("accepted execution requires TIU validation for zone %s", zoneDefinition.Key)
	}
	if !trustValid {
		return fmt.Errorf("accepted execution trust level is insufficient for zone %s", zoneDefinition.Key)
	}
	if !methodValid {
		return fmt.Errorf("accepted execution method %s is not allowed for zone %s", method, zoneDefinition.Key)
	}
	if !executiveValid {
		return fmt.Errorf("accepted execution requires executive access for zone %s", zoneDefinition.Key)
	}
	if !mfaValid {
		return fmt.Errorf("accepted execution requires MFA proof for zone %s", zoneDefinition.Key)
	}
	if finalStatus != "accepted" || !req.Execution.Accepted {
		return fmt.Errorf("accepted execution must set execution.accepted=true and final_status=accepted")
	}

	return nil
}

func (h *PipelineHandler) resolveExecutionZone(req *PipelineExecutionRequest) (policy.ZoneDefinition, error) {
	if req.Zone.ZoneID != nil {
		if zone, ok := h.zoneByID(*req.Zone.ZoneID); ok {
			if executionZoneKeyMatches(zone, req.Zone.ZoneKey) {
				return zone, nil
			}
			return policy.ZoneDefinition{}, fmt.Errorf("zone_key %q does not match zone_id %d", req.Zone.ZoneKey, *req.Zone.ZoneID)
		}
	}

	zoneKey := strings.TrimSpace(req.Zone.ZoneKey)
	if zoneKey == "" {
		return policy.ZoneDefinition{}, fmt.Errorf("zone resolution failed: zone_key is empty")
	}

	if zoneID, ok := policy.ZoneCodeToID(zoneKey); ok {
		if zone, ok := h.zoneByID(zoneID); ok {
			return zone, nil
		}
	}

	if h.policyRegistry != nil {
		if zone, ok := h.policyRegistry.GetZoneByKey(zoneKey); ok {
			return zone, nil
		}
	}

	return policy.ZoneDefinition{}, fmt.Errorf("zone resolution failed for %q", zoneKey)
}

func executionZoneKeyMatches(zone policy.ZoneDefinition, provided string) bool {
	trimmed := strings.TrimSpace(provided)
	if trimmed == "" {
		return false
	}
	if zone.HasKey(trimmed) {
		return true
	}
	return strings.EqualFold(trimmed, policy.ZoneIDToCode(zone.ID))
}

func inferExecutionTrustLevel(req *PipelineExecutionRequest) int {
	actorType := strings.ToLower(strings.TrimSpace(req.Actor.ActorType))
	if actorType == "superadmin" || actorType == "executive" {
		return 95
	}
	if actorType == "admin" || artifactBoolean(req.Artifacts, "executive_access") {
		return 85
	}
	if req.Actor.Authenticated {
		return 65
	}
	if strings.TrimSpace(req.Actor.DeviceID) != "" {
		return 35
	}
	return 10
}

func executionHasExecutiveAccess(req *PipelineExecutionRequest) bool {
	actorType := strings.ToLower(strings.TrimSpace(req.Actor.ActorType))
	if actorType == "admin" || actorType == "executive" || actorType == "superadmin" {
		return true
	}
	return artifactBoolean(req.Artifacts, "executive_access")
}

func executionHasMFAProof(artifacts map[string]any) bool {
	for _, key := range []string{"mfa_verified", "mfa_passed", "mfa_confirmed"} {
		if artifactBoolean(artifacts, key) {
			return true
		}
	}
	for _, key := range []string{"mfa_proof", "mfa_token", "otp", "totp", "verification_code"} {
		if value, ok := artifacts[key]; ok {
			if strings.TrimSpace(fmt.Sprint(value)) != "" {
				return true
			}
		}
	}
	return false
}

func artifactBoolean(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, ok := values[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized == "true" || normalized == "verified" || normalized == "passed"
	default:
		return false
	}
}

func boolPointerTrue(value *bool) bool {
	return value != nil && *value
}

func fingerprintHashValue(fingerprint string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.TrimSpace(fingerprint)))
	return int64(hasher.Sum64())
}

func extractArtifactFloat(values map[string]any, keys []string) float64 {
	for _, key := range keys {
		if values == nil {
			break
		}
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int:
			return float64(typed)
		case int32:
			return float64(typed)
		case int64:
			return float64(typed)
		case json.Number:
			if parsed, err := typed.Float64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
				return parsed
			}
		}
	}

	return 0
}

func extractTIUValue(req *PipelineExecutionRequest) float64 {
	if req.Artifacts != nil {
		if tiu := extractArtifactFloat(req.Artifacts, []string{"tiu", "tiu_score", "trust_tiu"}); tiu > 0 {
			return tiu
		}
	}
	if boolPointerTrue(req.Coherence.TIUValid) {
		return 1
	}
	return 0
}

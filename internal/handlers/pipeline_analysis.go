package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/pkg/mcp"
)

const pipelineAnalysisVersion = "2.0.0"

type PipelineAnalysisRequest struct {
	RequestID  string                 `json:"request_id"`
	TraceID    string                 `json:"trace_id"`
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	RouteName  string                 `json:"route_name"`
	Controller string                 `json:"controller"`
	Headers    map[string]string      `json:"headers"`
	Query      map[string][]string    `json:"query"`
	Payload    map[string]any         `json:"payload"`
	Middleware []string               `json:"middleware"`
	Source     PipelineAnalysisSource `json:"source"`
	Actor      PipelineAnalysisActor  `json:"actor"`
}

type PipelineAnalysisSource struct {
	SourceType  string `json:"source_type"`
	Transport   string `json:"transport"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
	IP          string `json:"ip"`
	UserAgent   string `json:"user_agent"`
}

type PipelineAnalysisActor struct {
	UserID        string `json:"user_id"`
	SessionID     string `json:"session_id"`
	DeviceID      string `json:"device_id"`
	Role          string `json:"role"`
	Authenticated bool   `json:"authenticated"`
}

type PipelineStageReport struct {
	Stage  int      `json:"stage"`
	Name   string   `json:"name"`
	Owner  string   `json:"owner"`
	Status string   `json:"status"`
	Notes  []string `json:"notes,omitempty"`
	Result any      `json:"result"`
}

type Stage1IngressPatternResult struct {
	RequestID           string            `json:"request_id"`
	TraceID             string            `json:"trace_id"`
	Method              string            `json:"method"`
	Path                string            `json:"path"`
	SourceType          string            `json:"source_type"`
	Transport           string            `json:"transport"`
	IngressPathZone     string            `json:"ingress_path_zone"`
	IngressRole         string            `json:"ingress_role"`
	ProposedZoneID      string            `json:"proposed_zone_id"`
	ProposedZoneName    string            `json:"proposed_zone_name"`
	Confidence          float64           `json:"confidence"`
	DetectionMethod     string            `json:"detection_method"`
	QuarantineCandidate bool              `json:"quarantine_candidate"`
	QuarantineReason    string            `json:"quarantine_reason,omitempty"`
	Z0Directive         string            `json:"z0_directive"`
	Z0Reason            string            `json:"z0_reason,omitempty"`
	Z0ConfidenceGate    float64           `json:"z0_confidence_gate"`
	MCPAvailable        bool              `json:"mcp_available"`
	Headers             map[string]string `json:"headers"`
}

const z0RedirectConfidenceGate = 0.90

type Stage2Z0EnvelopeResult struct {
	RequestID       string                 `json:"request_id"`
	TraceID         string                 `json:"trace_id"`
	StageZone       string                 `json:"stage_zone"`
	Method          string                 `json:"method"`
	Action          string                 `json:"action"`
	Path            string                 `json:"path"`
	NormalizedPath  string                 `json:"normalized_path"`
	Source          string                 `json:"source"`
	Actor           map[string]any         `json:"actor"`
	Timing          map[string]string      `json:"timing"`
	Payload         map[string]any         `json:"payload"`
	Fingerprint     string                 `json:"fingerprint"`
	HeadersCaptured int                    `json:"headers_captured"`
	QueryCaptured   int                    `json:"query_captured"`
	PayloadCaptured int                    `json:"payload_captured"`
}

type Stage3ContextResult struct {
	Authenticated bool              `json:"authenticated"`
	UserID        string            `json:"user_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	DeviceID      string            `json:"device_id,omitempty"`
	Role          string            `json:"role,omitempty"`
	RouteName     string            `json:"route_name,omitempty"`
	Controller    string            `json:"controller,omitempty"`
	Middleware    []string          `json:"middleware,omitempty"`
	SourceIP      string            `json:"source_ip,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	TrustLevel    int               `json:"trust_level"`
	FlowType      string            `json:"flow_type"`
	PhaseType     string            `json:"phase_type"`
	ZoneHints     map[string]string `json:"zone_hints"`
}

type Stage4IntentResult struct {
	Intent           string                     `json:"intent"`
	Pattern          string                     `json:"pattern"`
	ZoneCandidate    string                     `json:"zone_candidate"`
	Confidence       int                        `json:"confidence"`
	Sensitivity      string                     `json:"sensitivity"`
	Risk             string                     `json:"risk"`
	Traits           []string                   `json:"traits"`
	AnalyzerZoneID   int                        `json:"analyzer_zone_id"`
	AnalyzerZoneKey  string                     `json:"analyzer_zone_key"`
	AnalyzerZoneName string                     `json:"analyzer_zone_name"`
	MatchScore       int                        `json:"match_score"`
	MatchDetails     analyzer.MatchScoreDetails `json:"match_details"`
}

type Stage5CoherenceResult struct {
	Valid            bool     `json:"valid"`
	AuthValid        bool     `json:"auth_valid"`
	TIUValid         bool     `json:"tiu_valid"`
	TrustValid       bool     `json:"trust_valid"`
	MethodValid      bool     `json:"method_valid"`
	ExecutiveValid   bool     `json:"executive_valid"`
	MFAValid         bool     `json:"mfa_valid"`
	FingerprintEnforced bool   `json:"fingerprint_enforced"`
	Replay           bool     `json:"replay"`
	Drift            float64  `json:"drift"`
	SyncStatus       string   `json:"sync_status"`
	RouteFingerprint string   `json:"route_fingerprint"`
	InputFingerprint string   `json:"input_fingerprint"`
	CandidateZoneKey string   `json:"candidate_zone_key"`
	RequiredChecks   []string `json:"required_checks"`
	FailedChecks     []string `json:"failed_checks,omitempty"`
	RejectionReason  string   `json:"rejection_reason,omitempty"`
}

type Stage6ZoneAlignmentResult struct {
	ZoneID            int    `json:"zone_id"`
	ZoneCode          string `json:"zone_code"`
	ZoneKey           string `json:"zone_key"`
	ZoneName          string `json:"zone_name"`
	ZoneTier          string `json:"zone_tier"`
	ZoneSource        string `json:"zone_source"`
	MeetsRequirements bool   `json:"meets_requirements"`
	AlignmentReason   string `json:"alignment_reason"`
}

type Stage7ExecutionResult struct {
	Accepted        bool              `json:"accepted"`
	FinalStatus     string            `json:"final_status"`
	FinalStage      int               `json:"final_stage"`
	FinalStageName  string            `json:"final_stage_name"`
	AnalysisOnly    bool              `json:"analysis_only"`
	Recorded        bool              `json:"recorded"`
	RejectionReason string            `json:"rejection_reason,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

func (h *PipelineHandler) AnalyzeExecution(c *gin.Context) {
	if !h.authorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid pipeline API token",
		})
		return
	}

	var req PipelineAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := validatePipelineAnalysisRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	headers := normalizeHeaders(req.Headers)
	req.Headers = headers
	req.Method = normalizeMethod(req.Method)
	req.Path = normalizePath(req.Path)
	applyRequestDefaults(&req, headers)

	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = requestID
	}

	stage1Report, stage1Result := h.buildStage1Report(c.Request.Context(), &req, requestID, traceID)
	stage2Report, stage2Result := h.buildStage2Report(&req, requestID, traceID)
	stage3Report, stage3Result := h.buildStage3Report(&req, stage2Result)
	stage4Report, stage4Result := h.buildStage4Report(&req)
	stage5Report, stage5Result := h.buildStage5Report(&req, stage3Result, stage4Result)
	stage6Report, stage6Result := h.buildStage6Report(stage1Result, stage4Result, stage5Result)
	stage7Report, stage7Result := h.buildStage7Report(requestID, stage1Result, stage5Result, stage6Result)

	stages := []PipelineStageReport{
		stage1Report,
		stage2Report,
		stage3Report,
		stage4Report,
		stage5Report,
		stage6Report,
		stage7Report,
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":       requestID,
		"trace_id":         traceID,
		"pipeline_version": pipelineAnalysisVersion,
		"analysis_only":    true,
		"final_status":     stage7Result.FinalStatus,
		"accepted":         stage7Result.Accepted,
		"stages":           stages,
	})
}

func (h *PipelineHandler) buildStage1Report(ctx context.Context, req *PipelineAnalysisRequest, requestID string, traceID string) (PipelineStageReport, Stage1IngressPatternResult) {
	result := Stage1IngressPatternResult{
		RequestID:       requestID,
		TraceID:         traceID,
		Method:          req.Method,
		Path:            req.Path,
		SourceType:      req.Source.SourceType,
		Transport:       req.Source.Transport,
		IngressPathZone: "external",
		IngressRole:     "pattern-descriptor",
		ProposedZoneID:  "Z0",
		ProposedZoneName: "boundary",
		DetectionMethod: "fallback_z0_default",
		Confidence:      0,
		Z0Directive:     "run_full_middleware",
		Z0Reason:        "No trusted Stage 1 hypothesis yet; request remains in Z0 for orchestration.",
		Z0ConfidenceGate: z0RedirectConfidenceGate,
		MCPAvailable:    false,
		Headers:         req.Headers,
	}

	notes := []string{"Stage 1 describes the request pattern before internal middleware begins."}
	status := "completed"

	if h.mcpClient != nil && h.mcpClient.IsEnabled() {
		mcpResult, err := h.mcpClient.EvaluateIngress(ctx, mcp.IngressEvaluationRequest{
			Path:    req.Path,
			Method:  req.Method,
			Headers: req.Headers,
			Payload: req.Payload,
		})
		if err != nil {
			status = "degraded"
			notes = append(notes, "MCP unavailable, using fallback ingress description: "+err.Error())
		} else if mcpResult != nil {
			result.IngressPathZone = mcpResult.IngressPathZone
			result.IngressRole = mcpResult.IngressRole
			result.ProposedZoneID = mcpResult.ProposedZoneID
			result.ProposedZoneName = mcpResult.ProposedZoneName
			result.Confidence = mcpResult.Confidence
			result.DetectionMethod = mcpResult.DetectionMethod
			result.QuarantineCandidate = mcpResult.QuarantineCandidate
			result.QuarantineReason = mcpResult.QuarantineReason
			result.Z0Directive = strings.TrimSpace(mcpResult.Z0Directive)
			result.Z0Reason = strings.TrimSpace(mcpResult.Z0Reason)
			result.Z0ConfidenceGate = mcpResult.Z0ConfidenceGate
			result.MCPAvailable = true
		}
	} else {
		status = "degraded"
		notes = append(notes, "MCP disabled; Stage 1 reported from fallback defaults.")
	}

	if (!result.MCPAvailable || status == "degraded") && h.schemaRegistry != nil {
		match := h.schemaRegistry.MatchRequest(req.Method, req.Path)
		result.ProposedZoneID = match.ProposedZoneHint
		if strings.TrimSpace(match.RouteFamily) != "" {
			result.ProposedZoneName = match.RouteFamily
		} else if match.QuarantineCandidate {
			result.ProposedZoneName = "quarantine_envelope"
		}
		result.Confidence = match.Confidence
		result.DetectionMethod = match.DetectionMethod
		result.QuarantineCandidate = match.QuarantineCandidate
		result.QuarantineReason = match.QuarantineReason
		result.MCPAvailable = false

		if match.Matched {
			status = "completed"
			notes = append(notes, "Local schema preprocess produced a weighted route guess before Z0 orchestration.")
		} else {
			notes = append(notes, "Local schema preprocess could not align the request to a trusted route; request remains at the Z0 boundary as a quarantine candidate.")
		}
	}

	applyZ0Directive(&result)

	return PipelineStageReport{
		Stage:  1,
		Name:   "mcp_ingress_pattern",
		Owner:  "mcp",
		Status: status,
		Notes:  notes,
		Result: result,
	}, result
}

func applyZ0Directive(result *Stage1IngressPatternResult) {
	if result == nil {
		return
	}

	if result.Z0ConfidenceGate <= 0 {
		result.Z0ConfidenceGate = z0RedirectConfidenceGate
	}

	if result.QuarantineCandidate {
		result.Z0Directive = "run_full_middleware"
		if strings.TrimSpace(result.QuarantineReason) != "" {
			result.Z0Reason = result.QuarantineReason
		} else {
			result.Z0Reason = "Stage 1 marked the request as quarantine candidate, so Z0 must keep full control."
		}
		return
	}

	if result.Confidence >= result.Z0ConfidenceGate && strings.TrimSpace(result.ProposedZoneID) != "" && strings.TrimSpace(result.ProposedZoneID) != "Z0" {
		result.Z0Directive = "redirect_guessed_zone"
		if strings.TrimSpace(result.Z0Reason) == "" {
			result.Z0Reason = "Stage 1 produced a high-confidence zone guess; Z0 may fast-route the request while retaining orchestration authority."
		}
		return
	}

	result.Z0Directive = "run_full_middleware"
	if strings.TrimSpace(result.Z0Reason) == "" {
		result.Z0Reason = "Confidence is below the fast-route threshold, so Z0 should continue full middleware evaluation."
	}
}

func (h *PipelineHandler) buildStage2Report(req *PipelineAnalysisRequest, requestID string, traceID string) (PipelineStageReport, Stage2Z0EnvelopeResult) {
	cleanedPayload := normalizePayload(req.Payload)
	action := inferAction(req.Method)
	normalizedPath := normalizePath(req.Path)
	fingerprintValue := quickFingerprint(h.fingerprintAnalyzer, map[string]any{
		"method": req.Method,
		"path":   normalizedPath,
		"query":  req.Query,
		"body":   cleanedPayload,
	})

	result := Stage2Z0EnvelopeResult{
		RequestID:      requestID,
		TraceID:        traceID,
		StageZone:      "Z0",
		Method:         req.Method,
		Action:         action,
		Path:           req.Path,
		NormalizedPath: normalizedPath,
		Source:         req.Source.SourceType,
		Actor: map[string]any{
			"user_id":         req.Actor.UserID,
			"session_id":      req.Actor.SessionID,
			"device_id":       req.Actor.DeviceID,
			"authenticated":   req.Actor.Authenticated,
			"provisional_role": req.Actor.Role,
		},
		Timing: map[string]string{
			"received_at": time.Now().UTC().Format(time.RFC3339),
		},
		Payload:         cleanedPayload,
		Fingerprint:     fingerprintValue,
		HeadersCaptured: len(req.Headers),
		QueryCaptured:   len(req.Query),
		PayloadCaptured: len(cleanedPayload),
	}

	return PipelineStageReport{
		Stage:  2,
		Name:   "z0_middleware_envelope",
		Owner:  "middleware",
		Status: "completed",
		Notes:  []string{"Stage 2 is the first internal middleware stage and places the request on the Z0 path."},
		Result: result,
	}, result
}

func (h *PipelineHandler) buildStage3Report(req *PipelineAnalysisRequest, stage2 Stage2Z0EnvelopeResult) (PipelineStageReport, Stage3ContextResult) {
	flowType := inferFlowType(stage2.NormalizedPath, req.Controller)
	phaseType := inferPhaseType(req.Method, stage2.NormalizedPath)
	trustLevel := inferTrustLevel(req.Actor, req.Headers)
	zoneHints := map[string]string{
		"path_hint":       firstPathSegment(stage2.NormalizedPath),
		"controller_hint": strings.TrimSpace(req.Controller),
		"source_hint":     req.Source.SourceType,
	}

	result := Stage3ContextResult{
		Authenticated: req.Actor.Authenticated || headerPresent(req.Headers, "authorization"),
		UserID:        strings.TrimSpace(req.Actor.UserID),
		SessionID:     strings.TrimSpace(req.Actor.SessionID),
		DeviceID:      strings.TrimSpace(req.Actor.DeviceID),
		Role:          strings.TrimSpace(req.Actor.Role),
		RouteName:     strings.TrimSpace(req.RouteName),
		Controller:    strings.TrimSpace(req.Controller),
		Middleware:    append([]string(nil), req.Middleware...),
		SourceIP:      req.Source.IP,
		UserAgent:     req.Source.UserAgent,
		TrustLevel:    trustLevel,
		FlowType:      flowType,
		PhaseType:     phaseType,
		ZoneHints:     zoneHints,
	}

	return PipelineStageReport{
		Stage:  3,
		Name:   "context_middleware",
		Owner:  "middleware",
		Status: "completed",
		Notes:  []string{"Stage 3 resolves actor, route, device, and source context around the Z0 signal."},
		Result: result,
	}, result
}

func (h *PipelineHandler) buildStage4Report(req *PipelineAnalysisRequest) (PipelineStageReport, Stage4IntentResult) {
	route := analyzer.RouteInfo{
		URI:        req.Path,
		Methods:    []string{req.Method},
		Middleware: append([]string(nil), req.Middleware...),
		Name:       strings.TrimSpace(req.RouteName),
		Action: analyzer.ActionInfo{
			Controller: strings.TrimSpace(req.Controller),
			Method:     inferAction(req.Method),
			Type:       classifierActionType(req.Controller),
		},
	}

	analyzed := analyzer.AnalyzedRoute{}
	if h.routeAnalyzer != nil {
		analyzed = h.routeAnalyzer.AnalyzeRoute(route)
	}

	zoneCandidate := analyzed.ZoneKey
	if zoneCandidate == "" || analyzed.ZoneID == analyzer.UnassignedZoneID {
		zoneCandidate = inferZoneCandidateFromPath(req.Path)
	}

	intent, pattern, sensitivity, risk, confidence, traits := classifyIntent(req.Method, req.Path, zoneCandidate)

	result := Stage4IntentResult{
		Intent:           intent,
		Pattern:          pattern,
		ZoneCandidate:    zoneCandidate,
		Confidence:       confidence,
		Sensitivity:      sensitivity,
		Risk:             risk,
		Traits:           traits,
		AnalyzerZoneID:   analyzed.ZoneID,
		AnalyzerZoneKey:  analyzed.ZoneKey,
		AnalyzerZoneName: analyzed.ZoneName,
		MatchScore:       analyzed.Score,
		MatchDetails:     analyzed.ZoneAssignment.Details,
	}

	return PipelineStageReport{
		Stage:  4,
		Name:   "intent_pattern_classifier",
		Owner:  "analyzer",
		Status: "completed",
		Notes:  []string{"Stage 4 is separate from middleware and interprets what the request is trying to do."},
		Result: result,
	}, result
}

func (h *PipelineHandler) buildStage5Report(req *PipelineAnalysisRequest, stage3 Stage3ContextResult, stage4 Stage4IntentResult) (PipelineStageReport, Stage5CoherenceResult) {
	zoneDefinition, _ := h.lookupZoneDefinition(stage4, "")
	routeFingerprint := quickFingerprint(h.fingerprintAnalyzer, fingerprint.RouteData{
		URI:        req.Path,
		Controller: req.Controller,
		Method:     req.Method,
	})
	filterResult := fingerprint.FilterIntentFields(req.Payload, analyzerConfig(h.fingerprintAnalyzer))
	inputFingerprint := quickFingerprint(h.fingerprintAnalyzer, filterResult.Filtered)
	drift, syncStatus := compareFingerprints(h.fingerprintAnalyzer, routeFingerprint, inputFingerprint)

	authValid := !zoneDefinition.Rules.AuthRequired || stage3.Authenticated
	tiuPresent := headerPresent(req.Headers, "x-codex-tiu")
	tiuValid := !zoneDefinition.Rules.TIURequired || tiuPresent
	trustValid := trustLevelValid(zoneDefinition, stage3)
	methodValid := h.policyRegistry == nil || h.policyRegistry.IsMethodAllowed(zoneDefinition.ID, req.Method)
	executiveValid := !zoneDefinition.Rules.Executive || hasExecutiveAccess(stage3, req.Headers)
	mfaRequired := zoneDefinition.Rules.Executive && inferAction(req.Method) != "read"
	mfaValid := !mfaRequired || hasMFAProof(req.Headers, req.Payload)
	fingerprintEnforced := shouldEnforceFingerprint(zoneDefinition, stage4, filterResult) && authValid && tiuValid && trustValid && methodValid && executiveValid && mfaValid && !(zoneDefinition.Rules.Executive && mfaValid)

	requiredChecks := []string{"auth", "tiu", "trust", "method"}
	if zoneDefinition.Rules.Executive {
		requiredChecks = append(requiredChecks, "executive")
	}
	if mfaRequired {
		requiredChecks = append(requiredChecks, "mfa")
	}
	if fingerprintEnforced {
		requiredChecks = append(requiredChecks, "fingerprint")
	}

	failedChecks := []string{}
	rejectionReason := ""
	if !authValid {
		failedChecks = append(failedChecks, "auth")
		rejectionReason = firstReason(rejectionReason, "authentication required")
	}
	if !tiuValid {
		failedChecks = append(failedChecks, "tiu")
		rejectionReason = firstReason(rejectionReason, "TIU header missing")
	}
	if !trustValid {
		failedChecks = append(failedChecks, "trust")
		rejectionReason = firstReason(rejectionReason, "trust level insufficient")
	}
	if !methodValid {
		failedChecks = append(failedChecks, "method")
		rejectionReason = firstReason(rejectionReason, "method not allowed for candidate zone")
	}
	if !executiveValid {
		failedChecks = append(failedChecks, "executive")
		rejectionReason = firstReason(rejectionReason, "executive access required")
	}
	if !mfaValid {
		failedChecks = append(failedChecks, "mfa")
		rejectionReason = firstReason(rejectionReason, "multi-factor confirmation required for executive zone")
	}
	if fingerprintEnforced && syncStatus == fingerprint.SyncStatusError {
		failedChecks = append(failedChecks, "fingerprint")
		rejectionReason = firstReason(rejectionReason, "route and input fingerprints diverged")
	}

	result := Stage5CoherenceResult{
		Valid:            len(failedChecks) == 0,
		AuthValid:        authValid,
		TIUValid:         tiuValid,
		TrustValid:       trustValid,
		MethodValid:      methodValid,
		ExecutiveValid:   executiveValid,
		MFAValid:         mfaValid,
		FingerprintEnforced: fingerprintEnforced,
		Replay:           false,
		Drift:            drift,
		SyncStatus:       syncStatus,
		RouteFingerprint: routeFingerprint,
		InputFingerprint: inputFingerprint,
		CandidateZoneKey: zoneDefinition.Key,
		RequiredChecks:   requiredChecks,
		FailedChecks:     failedChecks,
		RejectionReason:  rejectionReason,
	}

	status := "completed"
	if !result.Valid {
		status = "rejected"
	}

	return PipelineStageReport{
		Stage:  5,
		Name:   "coherence_middleware",
		Owner:  "middleware",
		Status: status,
		Notes:  []string{"Stage 5 tests whether the classified signal is trustworthy enough to continue."},
		Result: result,
	}, result
}

func (h *PipelineHandler) buildStage6Report(stage1 Stage1IngressPatternResult, stage4 Stage4IntentResult, stage5 Stage5CoherenceResult) (PipelineStageReport, Stage6ZoneAlignmentResult) {
	zoneDefinition, source := h.lookupZoneDefinition(stage4, stage1.ProposedZoneID)
	alignmentReason := "resolved from classifier and policy registry"
	if stage1.QuarantineCandidate {
		zoneDefinition, _ = h.zoneByID(0)
		source = "stage1_quarantine"
		alignmentReason = "Stage 1 flagged the request for Z0 quarantine"
	} else if stage4.AnalyzerZoneID == analyzer.UnassignedZoneID && source == "stage1_mcp_hint" {
		alignmentReason = "Classifier was unassigned, so MCP hypothesis was used as a fallback"
	}

	result := Stage6ZoneAlignmentResult{
		ZoneID:            zoneDefinition.ID,
		ZoneCode:          policy.ZoneIDToCode(zoneDefinition.ID),
		ZoneKey:           zoneDefinition.Key,
		ZoneName:          zoneDefinition.Name,
		ZoneTier:          zoneDefinition.Tier,
		ZoneSource:        source,
		MeetsRequirements: stage1.QuarantineCandidate || stage5.Valid,
		AlignmentReason:   alignmentReason,
	}

	status := "completed"
	if !result.MeetsRequirements {
		status = "guarded"
	}

	return PipelineStageReport{
		Stage:  6,
		Name:   "zone_alignment_middleware",
		Owner:  "middleware",
		Status: status,
		Notes:  []string{"Stage 6 is the first point where the system is allowed to assign a realized zone."},
		Result: result,
	}, result
}

func (h *PipelineHandler) buildStage7Report(requestID string, stage1 Stage1IngressPatternResult, stage5 Stage5CoherenceResult, stage6 Stage6ZoneAlignmentResult) (PipelineStageReport, Stage7ExecutionResult) {
	accepted := !stage1.QuarantineCandidate && stage5.Valid && stage6.MeetsRequirements
	finalStatus := "accepted"
	rejectionReason := ""
	if stage1.QuarantineCandidate {
		finalStatus = "quarantined"
		rejectionReason = stage1.QuarantineReason
	} else if !accepted {
		finalStatus = "rejected"
		rejectionReason = stage5.RejectionReason
	}

	result := Stage7ExecutionResult{
		Accepted:       accepted,
		FinalStatus:    finalStatus,
		FinalStage:     7,
		FinalStageName: "execution_record_middleware",
		AnalysisOnly:   true,
		Recorded:       false,
		RejectionReason: rejectionReason,
		ResponseHeaders: map[string]string{
			"X-Codex-Request-ID": requestID,
			"X-Codex-Zone":       stage6.ZoneCode,
			"X-Codex-Zone-Key":   stage6.ZoneKey,
			"X-Codex-Result":     finalStatus,
		},
	}

	status := "completed"
	if !accepted {
		status = finalStatus
	}

	return PipelineStageReport{
		Stage:  7,
		Name:   "execution_record_middleware",
		Owner:  "middleware",
		Status: status,
		Notes:  []string{"Stage 7 reports execution gating and record state. This endpoint does not persist anything."},
		Result: result,
	}, result
}

func validatePipelineAnalysisRequest(req *PipelineAnalysisRequest) error {
	if strings.TrimSpace(req.Method) == "" {
		return fmt.Errorf("method is required")
	}
	if strings.TrimSpace(req.Path) == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func applyRequestDefaults(req *PipelineAnalysisRequest, headers map[string]string) {
	if strings.TrimSpace(req.Source.SourceType) == "" {
		req.Source.SourceType = "http"
	}
	if strings.TrimSpace(req.Source.Transport) == "" {
		req.Source.Transport = "http"
	}
	if strings.TrimSpace(req.Source.Service) == "" {
		req.Source.Service = "codexsecure-api"
	}
	if strings.TrimSpace(req.Source.UserAgent) == "" {
		req.Source.UserAgent = headers["user-agent"]
	}
	if strings.TrimSpace(req.Source.IP) == "" {
		req.Source.IP = firstNonEmpty(headers["x-forwarded-for"], headers["x-real-ip"])
	}
	if !req.Actor.Authenticated {
		req.Actor.Authenticated = headerPresent(headers, "authorization")
	}
	if strings.TrimSpace(req.Actor.SessionID) == "" {
		req.Actor.SessionID = firstNonEmpty(headers["x-session-id"], headers["x-codex-session"])
	}
	if strings.TrimSpace(req.Actor.DeviceID) == "" {
		req.Actor.DeviceID = headers["x-device-uid"]
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	if req.Query == nil {
		req.Query = map[string][]string{}
	}
	if req.Headers == nil {
		req.Headers = map[string]string{}
	}
	if req.Middleware == nil {
		req.Middleware = []string{}
	}
}

func normalizeHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return map[string]string{}
	}
	normalized := make(map[string]string, len(headers))
	for key, value := range headers {
		normalized[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return normalized
}

func normalizeMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return http.MethodGet
	}
	return method
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.ToLower(trimmed)
}

func normalizePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	normalized := make(map[string]any, len(payload))
	for key, value := range payload {
		normalized[strings.ToLower(strings.TrimSpace(key))] = normalizePayloadValue(value)
	}
	return normalized
}

func normalizePayloadValue(value any) any {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return normalizePayload(typed)
	case []string:
		result := make([]string, len(typed))
		for i, item := range typed {
			result[i] = strings.TrimSpace(item)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, item := range typed {
			result[i] = normalizePayloadValue(item)
		}
		return result
	default:
		return value
	}
}

func inferAction(method string) string {
	switch normalizeMethod(method) {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

func inferFlowType(path string, controller string) string {
	path = normalizePath(path)
	switch {
	case strings.Contains(path, "/admin") || strings.Contains(strings.ToLower(controller), "admin"):
		return "admin"
	case strings.Contains(path, "/auth") || strings.Contains(path, "/login") || strings.Contains(path, "/register"):
		return "identity"
	case strings.Contains(path, "/wallet") || strings.Contains(path, "/system"):
		return "system"
	case strings.Contains(path, "/api") || strings.Contains(path, "/sync"):
		return "api"
	default:
		return "public"
	}
}

func inferPhaseType(method string, path string) string {
	if strings.Contains(path, "/login") {
		return "login"
	}
	if strings.Contains(path, "/logout") {
		return "logout"
	}
	if strings.Contains(path, "/register") {
		return "register"
	}
	return inferAction(method)
}

func inferTrustLevel(actor PipelineAnalysisActor, headers map[string]string) int {
	if strings.EqualFold(actor.Role, "superadmin") || strings.EqualFold(actor.Role, "executive") {
		return 95
	}
	if strings.EqualFold(actor.Role, "admin") || headerPresent(headers, "x-executive-access") {
		return 85
	}
	if actor.Authenticated {
		return 65
	}
	if strings.TrimSpace(actor.DeviceID) != "" {
		return 35
	}
	return 10
}

func classifierActionType(controller string) string {
	if strings.TrimSpace(controller) == "" {
		return "closure"
	}
	return "controller"
}

// inferZoneCandidateFromPath returns a policy zone key for paths the route
// analyzer left unassigned.
func inferZoneCandidateFromPath(path string) string {
	path = normalizePath(path)
	switch {
	case strings.Contains(path, "/admin"):
		return "administration"
	case strings.Contains(path, "/auth") || strings.Contains(path, "/login") || strings.Contains(path, "/register"):
		return "identity"
	case strings.Contains(path, "/wallet"):
		return "payments"
	case strings.Contains(path, "/system"):
		return "processing"
	case strings.Contains(path, "/analytics"):
		return "analytics"
	case strings.Contains(path, "/sync"):
		return "synchronization"
	default:
		return "public"
	}
}

func classifyIntent(method string, path string, zoneCandidate string) (string, string, string, string, int, []string) {
	traits := []string{}
	action := inferAction(method)
	confidence := 72
	sensitivity := "medium"
	risk := "standard"
	intent := zoneCandidate
	pattern := action

	switch {
	case strings.Contains(path, "/login") || strings.Contains(path, "/auth"):
		intent = "identity"
		pattern = "auth_flow"
		sensitivity = "high"
		risk = "elevated"
		traits = append(traits, "auth_flow")
	case strings.Contains(path, "/admin"):
		intent = "admin"
		pattern = "admin_control"
		sensitivity = "critical"
		risk = "high"
		confidence = 85
		traits = append(traits, "admin_operation")
	case strings.Contains(path, "/wallet") || zoneCandidate == "processing":
		intent = "system"
		pattern = "system_execution"
		sensitivity = "high"
		risk = "elevated"
		confidence = 80
		traits = append(traits, "stateful")
	}

	if action != "read" {
		traits = append(traits, "mutation")
	}

	sort.Strings(traits)
	return intent, pattern, sensitivity, risk, confidence, traits
}

func (h *PipelineHandler) lookupZoneDefinition(stage4 Stage4IntentResult, stage1ProposedZoneID string) (policy.ZoneDefinition, string) {
	if h.policyRegistry == nil {
		return policy.DefaultZones()[1], "fallback_public"
	}

	if stage4.AnalyzerZoneID != analyzer.UnassignedZoneID {
		if zone, ok := h.policyRegistry.GetZone(stage4.AnalyzerZoneID); ok {
			return zone, "stage4_analyzer"
		}
	}

	if zone, ok := h.policyRegistry.GetZoneByKey(strings.TrimSpace(stage4.ZoneCandidate)); ok {
		return zone, "stage4_candidate"
	}

	if zoneID, ok := policy.ZoneCodeToID(strings.TrimSpace(stage1ProposedZoneID)); ok {
		if zone, ok := h.policyRegistry.GetZone(zoneID); ok {
			return zone, "stage1_mcp_hint"
		}
	}

	zone, _ := h.policyRegistry.GetZone(1)
	return zone, "fallback_public"
}

func (h *PipelineHandler) zoneByID(zoneID int) (policy.ZoneDefinition, bool) {
	if h.policyRegistry == nil {
		zone, ok := policy.DefaultZones()[zoneID]
		return zone, ok
	}
	return h.policyRegistry.GetZone(zoneID)
}

func analyzerConfig(fp *fingerprint.Analyzer) *fingerprint.Config {
	if fp == nil {
		return fingerprint.DefaultConfig()
	}
	return fp.GetConfig()
}

func quickFingerprint(fp *fingerprint.Analyzer, value any) string {
	if fp == nil {
		return fingerprint.NewAnalyzer(nil).QuickFingerprint(value)
	}
	return fp.QuickFingerprint(value)
}

func compareFingerprints(fp *fingerprint.Analyzer, routeFingerprint string, inputFingerprint string) (float64, string) {
	if fp == nil {
		fp = fingerprint.NewAnalyzer(nil)
	}
	result := fp.CompareFingerprints(routeFingerprint, inputFingerprint)
	return result.Drift, result.SyncStatus
}

func trustLevelValid(zone policy.ZoneDefinition, stage3 Stage3ContextResult) bool {
	switch zone.Tier {
	case policy.TierCritical:
		return stage3.TrustLevel >= 80
	case policy.TierBusiness:
		if zone.Rules.AuthRequired {
			return stage3.TrustLevel >= 50
		}
		return stage3.TrustLevel >= 25
	default:
		return true
	}
}

func firstPathSegment(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "root"
	}
	parts := strings.Split(trimmed, "/")
	return parts[0]
}

func headerPresent(headers map[string]string, key string) bool {
	value, ok := headers[strings.ToLower(strings.TrimSpace(key))]
	return ok && strings.TrimSpace(value) != ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstReason(existing string, candidate string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return candidate
}

func hasExecutiveAccess(stage3 Stage3ContextResult, headers map[string]string) bool {
	role := strings.ToLower(strings.TrimSpace(stage3.Role))
	if role == "admin" || role == "executive" || role == "superadmin" {
		return true
	}
	return headerPresent(headers, "x-executive-access")
}

func hasMFAProof(headers map[string]string, payload map[string]any) bool {
	for _, key := range []string{"x-codex-mfa", "x-mfa-verified", "x-auth-mfa"} {
		value := strings.ToLower(strings.TrimSpace(headers[key]))
		if value == "true" || value == "verified" || value == "passed" {
			return true
		}
	}

	for _, key := range []string{"mfa_token", "mfa_code", "otp", "totp", "verification_code"} {
		if value, ok := payload[key]; ok {
			if strings.TrimSpace(fmt.Sprint(value)) != "" {
				return true
			}
		}
	}

	return false
}

func shouldEnforceFingerprint(zone policy.ZoneDefinition, stage4 Stage4IntentResult, filterResult fingerprint.FilterResult) bool {
	if stage4.Pattern == "auth_flow" || stage4.Intent == "identity" {
		return false
	}

	if zone.Tier == policy.TierCritical {
		return true
	}

	// Intent defaults to the zone candidate, so Z12 traffic arrives as either label.
	if stage4.Intent == "admin" || stage4.Intent == "administration" {
		return true
	}

	if len(filterResult.IntentFields) >= 2 && !filterResult.UsedFallback {
		return true
	}

	return false
}
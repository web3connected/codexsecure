package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/schema"
)

const (
	HeaderPhaseLockScore      = "X-Codex-PhaseLock-Score"
	HeaderPhaseLockStrongLock = "X-Codex-PhaseLock-Strong"
)

type PhaseLockConfig struct {
	Enabled      bool
	MaxBodyBytes int64

	// EmitResponseHeaders optionally adds phase-lock summary headers.
	EmitResponseHeaders bool
}

func DefaultPhaseLockConfig() PhaseLockConfig {
	return PhaseLockConfig{
		Enabled:             strings.EqualFold(strings.TrimSpace(os.Getenv("PHASE_LOCK_ENABLED")), "true"),
		MaxBodyBytes:        64 * 1024,
		EmitResponseHeaders: strings.EqualFold(strings.TrimSpace(os.Getenv("PHASE_LOCK_HEADERS")), "true"),
	}
}

type phaseLockRole string

const (
	roleIdentifier phaseLockRole = "identifier"
	roleSecret     phaseLockRole = "secret"
)

type phaseLockEvent struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id"`

	ZoneKey string `json:"zone_key"`
	ZoneID  int    `json:"zone_id"`

	Method string `json:"method"`
	Path   string `json:"path"`

	// Application pattern (intent) signals.
	SchemaMatched    bool    `json:"schema_matched"`
	SchemaID         string  `json:"schema_id,omitempty"`
	SchemaRouteName  string  `json:"schema_route_name,omitempty"`
	SchemaFamily     string  `json:"schema_route_family,omitempty"`
	IntentType       string  `json:"intent_type,omitempty"`
	SchemaZoneHint   string  `json:"schema_zone_hint,omitempty"`
	SchemaConfidence float64 `json:"schema_confidence"`

	// User pattern signals (no raw values).
	ObservedRoles []string `json:"observed_roles"`
	ExpectedRoles []string `json:"expected_roles"`

	RouteFingerprint string  `json:"route_fingerprint"`
	UserFingerprint  string  `json:"user_fingerprint"`
	Score            float64 `json:"score"`
	StrongLock       bool    `json:"strong_lock"`
	StatusCode       int     `json:"status_code"`
}

// PhaseLockEvent is the normalized Phase Lock record emitted by middleware or handlers.
// It contains no raw user secrets; only redacted role/features and fingerprints.
type PhaseLockEvent = phaseLockEvent

// ComputePhaseLockEvent derives the Phase Lock record from a request snapshot.
// This is used by both middleware and explicit recording endpoints.
func ComputePhaseLockEvent(
	fp *fingerprint.Analyzer,
	reg *schema.Registry,
	requestID string,
	zoneKey string,
	zoneID int,
	method string,
	path string,
	handlerName string,
	payload map[string]any,
	statusCode int,
) PhaseLockEvent {
	if fp == nil {
		fp = fingerprint.NewAnalyzer(nil)
	}

	schemaMatch, schemaRecord := matchSchema(reg, method, path)
	intentType := ""
	schemaRouteName := ""
	schemaFamily := ""
	schemaID := ""
	schemaZoneHint := ""
	schemaConfidence := schemaMatch.Confidence
	if schemaRecord != nil {
		intentType = strings.TrimSpace(schemaRecord.Intent.Type)
		schemaRouteName = strings.TrimSpace(schemaRecord.Route.Name)
		schemaFamily = strings.TrimSpace(schemaRecord.Route.Family)
		schemaID = strings.TrimSpace(schemaRecord.SchemaID)
		schemaZoneHint = strings.TrimSpace(schemaRecord.Route.ZoneHint)
	}

	expectedRoles := expectedRolesForIntent(intentType, path)
	observedRoles, redacted := deriveObservedRoles(payload)

	routeFP := fp.GenerateRouteFingerprint(fingerprint.RouteData{URI: path, Method: method, Controller: handlerName})
	userFP := fp.QuickFingerprint(redacted)

	roleScore := jaccardScore(expectedRoles, observedRoles)
	score := clamp01(0.5*schemaConfidence + 0.5*roleScore)
	strong := score >= 0.90

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return phaseLockEvent{
		Timestamp:        time.Now().UTC(),
		RequestID:        requestID,
		ZoneKey:          zoneKey,
		ZoneID:           zoneID,
		Method:           method,
		Path:             path,
		SchemaMatched:    schemaMatch.Matched,
		SchemaID:         schemaID,
		SchemaRouteName:  schemaRouteName,
		SchemaFamily:     schemaFamily,
		IntentType:       intentType,
		SchemaZoneHint:   schemaZoneHint,
		SchemaConfidence: schemaConfidence,
		ObservedRoles:    observedRoles,
		ExpectedRoles:    expectedRoles,
		RouteFingerprint: routeFP,
		UserFingerprint:  userFP,
		Score:            score,
		StrongLock:       strong,
		StatusCode:       statusCode,
	}
}

// PhaseLockMiddleware records request patterns from two angles:
// 1) application intent (page schema / route intent), and
// 2) user-submitted pattern roles (identifier/secret) derived primarily from VALUES (field naming is optional).
//
// It is intentionally log-only by default.
func PhaseLockMiddleware(fp *fingerprint.Analyzer, reg *schema.Registry, cfg PhaseLockConfig) gin.HandlerFunc {
	if fp == nil {
		fp = fingerprint.NewAnalyzer(nil)
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = DefaultPhaseLockConfig().MaxBodyBytes
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Z2 group wiring is the primary gate.
		path := firstNonEmpty(c.FullPath(), c.Request.URL.Path)

		requestID := GetRequestID(c)
		if requestID == "" {
			requestID = c.GetHeader(HeaderRequestID)
		}

		bodyBytes, payload := readRequestPayloadForPhaseLock(c, cfg.MaxBodyBytes)
		_ = bodyBytes

		// Proceed through the chain/handler.
		c.Next()

		zoneKey := GetZoneKey(c)
		zoneID := GetZoneID(c)
		if zoneKey != "identity" && zoneID != 2 {
			return
		}

		event := ComputePhaseLockEvent(
			fp,
			reg,
			requestID,
			zoneKey,
			zoneID,
			c.Request.Method,
			path,
			c.HandlerName(),
			payload,
			c.Writer.Status(),
		)

		log.Info().
			Str("event", "phase_lock").
			Str("zone_key", zoneKey).
			Int("zone_id", zoneID).
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", path).
			Bool("schema_matched", event.SchemaMatched).
			Str("schema_id", event.SchemaID).
			Str("intent", event.IntentType).
			Float64("schema_confidence", event.SchemaConfidence).
			Strs("observed_roles", event.ObservedRoles).
			Strs("expected_roles", event.ExpectedRoles).
			Float64("score", event.Score).
			Bool("strong_lock", event.StrongLock).
			Int("status_code", event.StatusCode).
			Msg("Phase lock record")

		if cfg.EmitResponseHeaders {
			c.Header(HeaderPhaseLockScore, formatFloat(event.Score))
			c.Header(HeaderPhaseLockStrongLock, formatBool(event.StrongLock))
		}
	}
}

func matchSchema(reg *schema.Registry, method string, path string) (schema.MatchResult, *schema.Record) {
	if reg == nil {
		return schema.MatchResult{}, nil
	}
	return reg.MatchRequestWithRecord(method, path)
}

func readRequestPayloadForPhaseLock(c *gin.Context, maxBytes int64) ([]byte, map[string]any) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return nil, nil
	}

	// Avoid blocking on streaming/chunked request bodies.
	// If Content-Length is unknown (<0), do not attempt to read here.
	// (Downstream handlers may still read/stream the body as needed.)
	if c.Request.ContentLength < 0 {
		return nil, nil
	}
	if c.Request.ContentLength == 0 {
		return nil, nil
	}
	if maxBytes > 0 && c.Request.ContentLength > maxBytes {
		return nil, nil
	}

	mediaType := strings.TrimSpace(c.GetHeader("Content-Type"))
	if mediaType != "" {
		parsed, _, err := mime.ParseMediaType(mediaType)
		if err == nil {
			mediaType = parsed
		}
	}

	// Only attempt JSON bodies for now.
	if mediaType != "" && !strings.Contains(strings.ToLower(mediaType), "application/json") {
		return nil, nil
	}

	limited := io.LimitReader(c.Request.Body, maxBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil || len(bodyBytes) == 0 {
		// Restore an empty reader so downstream doesn't panic.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
		return nil, nil
	}

	// Restore body for downstream handlers.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload map[string]any
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return bodyBytes, nil
	}
	return bodyBytes, payload
}

var emailRegex = regexp.MustCompile(`(?i)^[^@\s]{1,128}@[a-z0-9.-]{1,128}\.[a-z]{2,24}$`)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,64}$`)

func deriveObservedRoles(payload map[string]any) ([]string, map[string]any) {
	roles := map[phaseLockRole]bool{}
	features := map[string]any{}
	if payload == nil {
		return []string{}, map[string]any{"roles": []string{}}
	}

	for key, value := range payload {
		keyLower := strings.ToLower(strings.TrimSpace(key))
		role, fieldFeatures := detectRoleFromField(keyLower, value)
		if role != "" {
			roles[role] = true
			features[string(role)] = mergeFeatureMaps(features[string(role)], fieldFeatures)
		}
	}

	roleList := make([]string, 0, len(roles))
	for role := range roles {
		roleList = append(roleList, string(role))
	}
	sort.Strings(roleList)

	redacted := map[string]any{
		"roles":  roleList,
		"fields": features,
	}

	return roleList, redacted
}

func detectRoleFromField(key string, value any) (phaseLockRole, map[string]any) {
	stringValue, ok := value.(string)
	if !ok {
		return "", nil
	}
	trimmed := strings.TrimSpace(stringValue)
	if trimmed == "" {
		return "", nil
	}

	lower := strings.ToLower(trimmed)
	lenBucket := bucketLen(len(trimmed))

	// Identifier: email by VALUE.
	if emailRegex.MatchString(lower) {
		return roleIdentifier, map[string]any{"kind": "email", "len": lenBucket}
	}

	// Secret: by key hint OR by value complexity.
	if strings.Contains(key, "pass") || strings.Contains(key, "pwd") || strings.Contains(key, "secret") {
		return roleSecret, map[string]any{"kind": "secret", "len": lenBucket}
	}
	if looksLikeSecret(trimmed) {
		return roleSecret, map[string]any{"kind": "secret", "len": lenBucket}
	}

	// Identifier: username by VALUE (fallback).
	if usernameRegex.MatchString(trimmed) {
		return roleIdentifier, map[string]any{"kind": "username", "len": lenBucket}
	}

	return "", nil
}

func looksLikeSecret(value string) bool {
	if len(value) < 8 {
		return false
	}
	// Too much whitespace is unlikely for a secret.
	if strings.ContainsAny(value, " \t\n\r") {
		return false
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := false
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	classes := 0
	if hasLower {
		classes++
	}
	if hasUpper {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSymbol {
		classes++
	}

	return classes >= 3
}

func expectedRolesForIntent(intentType string, path string) []string {
	intent := strings.ToLower(strings.TrimSpace(intentType))
	pathLower := strings.ToLower(path)

	// Schema-driven mappings.
	switch intent {
	case "auth.login", "auth.signin", "auth.authenticate":
		return []string{string(roleIdentifier), string(roleSecret)}
	case "auth.register", "auth.signup":
		return []string{string(roleIdentifier), string(roleSecret)}
	case "auth.password_reset", "auth.forgot_password":
		return []string{string(roleIdentifier)}
	}

	// Fallback path heuristics.
	if strings.Contains(pathLower, "login") || strings.Contains(pathLower, "signin") {
		return []string{string(roleIdentifier), string(roleSecret)}
	}
	if strings.Contains(pathLower, "register") || strings.Contains(pathLower, "signup") {
		return []string{string(roleIdentifier), string(roleSecret)}
	}
	if strings.Contains(pathLower, "reset") || strings.Contains(pathLower, "forgot") {
		return []string{string(roleIdentifier)}
	}

	return []string{}
}

func jaccardScore(expected []string, observed []string) float64 {
	if len(expected) == 0 && len(observed) == 0 {
		return 1
	}
	if len(expected) == 0 {
		return 0.5
	}

	setA := make(map[string]bool, len(expected))
	for _, item := range expected {
		setA[item] = true
	}
	setB := make(map[string]bool, len(observed))
	for _, item := range observed {
		setB[item] = true
	}

	intersection := 0
	union := 0
	for item := range setA {
		union++
		if setB[item] {
			intersection++
		}
	}
	for item := range setB {
		if !setA[item] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func bucketLen(n int) string {
	switch {
	case n <= 0:
		return "0"
	case n <= 4:
		return "1-4"
	case n <= 7:
		return "5-7"
	case n <= 15:
		return "8-15"
	case n <= 31:
		return "16-31"
	case n <= 63:
		return "32-63"
	default:
		return "64+"
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func mergeFeatureMaps(existing any, incoming map[string]any) map[string]any {
	result := map[string]any{}
	if typed, ok := existing.(map[string]any); ok {
		for k, v := range typed {
			result[k] = v
		}
	}
	for k, v := range incoming {
		result[k] = v
	}
	return result
}

func formatFloat(v float64) string {
	// Avoid fmt import just for formatting.
	buf, _ := json.Marshal(v)
	return strings.Trim(string(buf), "\"")
}

func formatBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

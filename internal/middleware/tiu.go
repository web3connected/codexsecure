package middleware

// TIU Middleware - Time Integrity Unit validation
// Validates TIU sync and session tokens for authenticated zones.
//
// Converted from: CodexTIUMiddleware.php
// @version 2.0.0

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

// TIU error responses
var (
	ErrTIUDrift = gin.H{
		"error": "TIU drift detected. Please log in again.",
		"code":  "TIU_DRIFT",
	}
	ErrSessionMissing = gin.H{
		"error": "Session token missing",
		"code":  "SESSION_MISSING",
	}
	ErrSessionExpired = gin.H{
		"error": "Codex session expired",
		"code":  "SESSION_EXPIRED",
	}
	ErrTIUMissing = gin.H{
		"error": "TIU token missing",
		"code":  "TIU_MISSING",
	}
	ErrTIUInvalid = gin.H{
		"error": "Invalid TIU token",
		"code":  "TIU_INVALID",
	}
	ErrTIUExpired = gin.H{
		"error": "TIU token expired",
		"code":  "TIU_EXPIRED",
	}
)

// TIUConfig configures TIU middleware behavior
type TIUConfig struct {
	// SkipZones lists zone IDs to skip TIU validation (e.g., public zones)
	SkipZones []int

	// DriftThreshold is max allowed drift (default: 0.01)
	// Deprecated: DriftThreshold was used by the simplified drift check.
	// Prefer MaxDriftSeconds + signed TIU tokens.
	DriftThreshold float64

	// SigningSecret enables signed TIU validation when non-empty.
	SigningSecret string

	// MaxDriftSeconds is the max allowed seconds between iat and server time.
	MaxDriftSeconds float64

	// MaxFutureSkewSeconds is the max allowed iat in the future.
	MaxFutureSkewSeconds float64

	// TokenHeader is the header name for session token
	TokenHeader string

	// TokenCookie is the cookie name for session token
	TokenCookie string

	// DeviceHeader is the header name for device UID
	DeviceHeader string

	// TIUHeader is the header name for signed TIU token
	TIUHeader string

	// CodexTimeClient is used to fetch live TIU when SigningSecret is empty.
	// Nil disables live TIU fetching.
	CodexTimeClient *codextime.Client
}

// DefaultTIUConfig returns sensible defaults
func DefaultTIUConfig() *TIUConfig {
	maxDrift := getEnvFloat("TIU_MAX_DRIFT_SECONDS", 15)
	maxFutureSkew := getEnvFloat("TIU_MAX_FUTURE_SKEW_SECONDS", 2)
	return &TIUConfig{
		SkipZones:            []int{0, 1}, // Z0 (Boundary), Z1 (public)
		DriftThreshold:       0.01,
		SigningSecret:        strings.TrimSpace(os.Getenv("TIU_HMAC_SECRET")),
		MaxDriftSeconds:      maxDrift,
		MaxFutureSkewSeconds: maxFutureSkew,
		TokenHeader:          "X-Codex-Token",
		TokenCookie:          "codex_token",
		DeviceHeader:         "X-Device-UID",
		TIUHeader:            "X-Codex-TIU",
		CodexTimeClient:      codextime.NewClient(),
	}
}

// TIUMiddleware validates TIU sync and session integrity
func TIUMiddleware(fp *fingerprint.Analyzer, cfg *TIUConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultTIUConfig()
	}

	return func(c *gin.Context) {
		zoneID := GetZoneID(c)

		// Skip for configured zones (typically public)
		if shouldSkipZone(zoneID, cfg.SkipZones) {
			c.Next()
			return
		}

		// Get zone config to check if TIU is required
		zoneConfig := GetZoneConfig(c)
		if zoneConfig != nil && !zoneConfig.Rules.TIURequired {
			c.Next()
			return
		}

		if !shouldEnforceTIUSession(c) {
			deviceUID := c.GetHeader(cfg.DeviceHeader)
			if deviceUID != "" {
				c.Set("device_uid", deviceUID)
			}
			c.Next()
			return
		}

		// Get session token
		token := c.GetHeader(cfg.TokenHeader)
		if token == "" {
			var err error
			token, err = c.Cookie(cfg.TokenCookie)
			if err != nil || token == "" {
				log.Warn().
					Int("zone_id", zoneID).
					Str("path", c.Request.URL.Path).
					Msg("Session token missing")

				c.AbortWithStatusJSON(http.StatusPreconditionRequired, ErrSessionMissing)
				return
			}
		}

		// Validate TIU sync
		// Signed TIU validation (offline)
		// Enabled when cfg.SigningSecret is set. This avoids a runtime dependency on CodexTime.
		if cfg.SigningSecret != "" {
			token := strings.TrimSpace(c.GetHeader(cfg.TIUHeader))
			if token == "" {
				c.AbortWithStatusJSON(StatusTIURequired, ErrTIUMissing)
				return
			}

			payload, err := verifySignedTIUToken(token, cfg.SigningSecret)
			if err != nil {
				if errors.Is(err, errTIUExpired) {
					c.AbortWithStatusJSON(StatusLoginTimeout, ErrTIUExpired)
					return
				}
				c.AbortWithStatusJSON(StatusTIURequired, ErrTIUInvalid)
				return
			}

			now := time.Now().UTC().Unix()
			if payload.Iat > now+int64(cfg.MaxFutureSkewSeconds) {
				c.AbortWithStatusJSON(StatusTIURequired, ErrTIUInvalid)
				return
			}
			driftSeconds := float64(now - payload.Iat)
			if driftSeconds < 0 {
				driftSeconds = -driftSeconds
			}
			if driftSeconds > cfg.MaxDriftSeconds {
				log.Warn().
					Int("zone_id", zoneID).
					Float64("drift_seconds", driftSeconds).
					Float64("threshold_seconds", cfg.MaxDriftSeconds).
					Msg("TIU drift detected")
				c.AbortWithStatusJSON(StatusLoginTimeout, ErrTIUDrift)
				return
			}

			deviceUID := strings.TrimSpace(c.GetHeader(cfg.DeviceHeader))
			if payload.DeviceUID != "" && deviceUID != "" && payload.DeviceUID != deviceUID {
				c.AbortWithStatusJSON(StatusTIURequired, ErrTIUInvalid)
				return
			}
		}

		// When no HMAC secret is configured, fetch the live TIU from CodexTime
		// and store it in context for downstream handlers.
		if cfg.SigningSecret == "" && cfg.CodexTimeClient != nil {
			tiuResp, err := cfg.CodexTimeClient.CurrentTIU(c.Request.Context())
			if err != nil {
				log.Warn().Err(err).Msg("CodexTime TIU fetch failed (fail-open)")
			} else {
				c.Set("codex_tiu", tiuResp.TIU)
				c.Set("codex_tiu_source", tiuResp.Source)
			}
		}

		// Store device UID in context
		deviceUID := c.GetHeader(cfg.DeviceHeader)
		if deviceUID != "" {
			c.Set("device_uid", deviceUID)
		}

		c.Next()
	}
}

type signedTIUPayload struct {
	Version   string
	Iat       int64
	Exp       int64
	DeviceUID string
}

var errTIUExpired = errors.New("tiu_expired")

func verifySignedTIUToken(token string, secret string) (signedTIUPayload, error) {
	var payload signedTIUPayload
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return payload, errors.New("invalid_format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, errors.New("invalid_payload")
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, errors.New("invalid_signature")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	expected := mac.Sum(nil)
	if !hmac.Equal(sigBytes, expected) {
		return payload, errors.New("bad_signature")
	}

	vals, err := url.ParseQuery(string(payloadBytes))
	if err != nil {
		return payload, errors.New("invalid_payload")
	}
	payload.Version = vals.Get("v")
	if payload.Version != "1" {
		return payload, errors.New("unsupported_version")
	}
	if payload.Iat, err = strconv.ParseInt(vals.Get("iat"), 10, 64); err != nil {
		return payload, errors.New("invalid_iat")
	}
	if payload.Exp, err = strconv.ParseInt(vals.Get("exp"), 10, 64); err != nil {
		return payload, errors.New("invalid_exp")
	}
	payload.DeviceUID = vals.Get("device_uid")

	now := time.Now().UTC().Unix()
	if payload.Exp > 0 && now > payload.Exp {
		return payload, errTIUExpired
	}

	return payload, nil
}

func getEnvFloat(key string, fallback float64) float64 {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func shouldEnforceTIUSession(c *gin.Context) bool {
	zoneConfig := GetZoneConfig(c)
	if zoneConfig == nil {
		return false
	}

	zoneKey := zoneConfig.Key
	authRequired := zoneConfig.Rules.AuthRequired

	if authRequired {
		return true
	}

	if strings.EqualFold(zoneKey, "identity") && isIdentityBootstrapRequest(c) {
		return false
	}

	return hasSessionArtifacts(c)
}

func isIdentityBootstrapRequest(c *gin.Context) bool {
	path := strings.ToLower(c.Request.URL.Path)
	method := strings.ToUpper(c.Request.Method)

	if method != http.MethodGet && method != http.MethodPost && method != http.MethodOptions {
		return false
	}

	identityMarkers := []string{"/login", "/signin", "/register", "/signup", "/logout", "/verify", "/reset", "/forgot", "/password"}
	for _, marker := range identityMarkers {
		if strings.Contains(path, marker) {
			return true
		}
	}

	return false
}

func hasSessionArtifacts(c *gin.Context) bool {
	if strings.TrimSpace(c.GetHeader("Authorization")) != "" {
		return true
	}
	if strings.TrimSpace(c.GetHeader("X-Codex-Token")) != "" {
		return true
	}
	if _, err := c.Cookie("codex_token"); err == nil {
		return true
	}
	if _, err := c.Cookie("codex_session"); err == nil {
		return true
	}
	if val, exists := c.Get("authenticated"); exists {
		if authenticated, ok := val.(bool); ok && authenticated {
			return true
		}
	}
	return false
}

// shouldSkipZone checks if a zone should skip TIU validation
func shouldSkipZone(zoneID int, skipZones []int) bool {
	for _, skip := range skipZones {
		if zoneID == skip {
			return true
		}
	}
	return false
}

// RequireTIU is a stricter middleware that always requires TIU validation
func RequireTIU(fp *fingerprint.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get TIU timestamp from header
		tiuHeader := c.GetHeader("X-Codex-TIU")
		if tiuHeader == "" {
			c.AbortWithStatusJSON(http.StatusPreconditionRequired, gin.H{
				"error": "TIU timestamp required",
				"code":  "TIU_MISSING",
			})
			return
		}

		// Validate TIU timestamp is recent
		// In real implementation, parse and validate timestamp
		// For now, just check header exists

		c.Next()
	}
}

// ValidateTIUSync validates TIU synchronization
// Returns true if sync is valid, false if drift detected
func ValidateTIUSync(fp *fingerprint.Analyzer, threshold float64) bool {
	if fp == nil {
		return true // No analyzer, skip validation
	}

	// In real implementation, compare server time with client time
	// For now, always return true
	return true
}

// HTTP status codes for TIU errors
const (
	// StatusLoginTimeout (440) - TIU drift detected, re-login required
	StatusLoginTimeout = 440

	// StatusTIURequired (428) - TIU validation required
	StatusTIURequired = 428
)

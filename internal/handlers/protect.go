package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

const (
	// protectMaxBodyBytes bounds the communication envelope.
	protectMaxBodyBytes = 1 << 20

	// CodexHash payload hash: codexhash-api POST /api/hash/payload (stateless) with SHA-512.
	protectHashAlgo = "sha512"
	protectHashSize = 512
)

var sha512Hex = regexp.MustCompile(`^[0-9a-f]{128}$`)

// payloadHasher is the CodexHash operation used by Protect (codexhash.Client).
type payloadHasher interface {
	PayloadHash(ctx context.Context, data string, algo string, hashSize int) (codexhash.PayloadHashResult, error)
}

// tiuSource is the CodexTime operation used by Protect (codextime.Client).
type tiuSource interface {
	CurrentTIU(ctx context.Context) (codextime.TIUResponse, error)
}

// ProtectHandler serves the Stage 4 (Protect) CodexHash and TIU operations
// requested by the Laravel SDK. It does not run a pipeline or record events itself.
type ProtectHandler struct {
	hasher payloadHasher
	time   tiuSource
}

// NewProtectHandler creates a Protect handler from the existing CodexHash and CodexTime clients.
func NewProtectHandler(hasher payloadHasher, time tiuSource) *ProtectHandler {
	return &ProtectHandler{hasher: hasher, time: time}
}

// GenerateCodexHash returns the CodexHash of the communication payload.
// POST /api/v1/protect/codexhash
func (h *ProtectHandler) GenerateCodexHash(c *gin.Context) {
	canonical, ok := bindProtectPayload(c)
	if !ok {
		return
	}

	result, err := h.hasher.PayloadHash(c.Request.Context(), canonical, protectHashAlgo, protectHashSize)
	if err != nil {
		log.Warn().Str("operation", "protect.codexhash").Msg("CodexHash unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "codexhash_unavailable",
			"message": "CodexHash could not generate a hash",
		})
		return
	}
	if result.Algo != protectHashAlgo || !sha512Hex.MatchString(result.PayloadHash) {
		log.Warn().Str("operation", "protect.codexhash").Str("algo", result.Algo).Msg("CodexHash returned an unexpected result")
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "invalid_service_response",
			"message": "CodexHash returned an unexpected result",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"codexhash":      result.PayloadHash,
		"algorithm":      result.Algo,
		"representation": protect.Representation,
	})
}

// IssueTIU returns the current CodexTime TIU for the communication.
// POST /api/v1/protect/tiu
func (h *ProtectHandler) IssueTIU(c *gin.Context) {
	if _, ok := bindProtectPayload(c); !ok {
		return
	}

	result, err := h.time.CurrentTIU(c.Request.Context())
	// The CodexTime client fails open with TIU 0 / source "unavailable"; that is never a TIU.
	if err != nil || result.Source != "api" || result.TIU <= 0 {
		log.Warn().Str("operation", "protect.tiu").Str("source", result.Source).Msg("CodexTime TIU unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "tiu_unavailable",
			"message": "CodexTime could not provide a TIU",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tiu":       result.TIU,
		"timestamp": result.Timestamp,
		"epoch":     result.Epoch,
		"source":    "codextime",
	})
}

// bindProtectPayload reads {"payload": {...}} and returns the canonical payload.
// Error responses never echo payload contents.
func bindProtectPayload(c *gin.Context) (string, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, protectMaxBodyBytes)

	var req struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "payload_too_large",
				"message": "Request body exceeds 1 MiB",
			})
			return "", false
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Request body must be a JSON object with a payload field",
		})
		return "", false
	}
	if len(req.Payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "payload is required",
		})
		return "", false
	}

	canonical, err := protect.CanonicalPayload(req.Payload)
	switch {
	case err == nil:
		return canonical, true
	case errors.Is(err, protect.ErrReservedField):
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "reserved_field",
			"message": "payload must not contain codexhash or tiu",
		})
	case errors.Is(err, protect.ErrDuplicateKey):
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "duplicate_key",
			"message": "payload contains a duplicate object key",
		})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "invalid_payload",
			"message": "payload must be a non-empty JSON object",
		})
	}
	return "", false
}

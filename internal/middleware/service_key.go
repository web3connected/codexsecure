package middleware

// ServiceKeyMiddleware - inter-service authentication
//
// Protects zone detection and analyzer endpoints from unauthenticated callers.
// Clients must supply the shared secret in the X-Service-Key header.
//
// Configuration:
//   SERVICE_API_KEY — shared secret for all authorised Codex services.
//                     If unset the middleware is a no-op (dev convenience).

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ServiceKeyMiddleware returns a Gin middleware that validates X-Service-Key.
func ServiceKeyMiddleware() gin.HandlerFunc {
	key := strings.TrimSpace(os.Getenv("SERVICE_API_KEY"))

	if key == "" {
		log.Warn().Msg("SERVICE_API_KEY is not set — service key auth is DISABLED")
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		provided := strings.TrimSpace(c.GetHeader("X-Service-Key"))
		if provided != key {
			log.Warn().
				Str("path", c.Request.URL.Path).
				Str("remote_addr", c.ClientIP()).
				Msg("rejected: invalid or missing X-Service-Key")

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or missing service key",
				"code":  "SERVICE_KEY_INVALID",
			})
			return
		}
		c.Next()
	}
}

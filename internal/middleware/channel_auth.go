package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ChannelAuth reuses the communication authentication: the service key plus the
// trusted gateway principal installed by SetCommunicationPrincipal. Channel
// operations are application operations, so they need a ChannelSites grant for
// {site} but no administrator actor and no Idempotency-Key. Like CommunicationAuth,
// a missing SERVICE_API_KEY or gateway principal always denies access.
func ChannelAuth() gin.HandlerFunc {
	key := strings.TrimSpace(os.Getenv("SERVICE_API_KEY"))
	return func(c *gin.Context) {
		fail := func(status int, code, message string) {
			c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
		}
		if key == "" || subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Service-Key")), []byte(key)) != 1 {
			fail(http.StatusUnauthorized, "SERVICE_KEY_INVALID", "invalid or missing service key")
			return
		}
		value, ok := c.Get(communicationPrincipalKey)
		p, valid := value.(CommunicationPrincipal)
		if !ok || !valid || !p.GatewayAuthenticated {
			fail(http.StatusForbidden, "ADMIN_UNAUTHORIZED", "trusted gateway identity is required")
			return
		}
		if !p.ChannelSites[c.Param("site")] {
			fail(http.StatusForbidden, "ADMIN_UNAUTHORIZED", "channel access denied")
			return
		}
		c.Next()
	}
}

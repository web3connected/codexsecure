package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CommunicationPrincipal must be installed by trusted authentication middleware,
// after verifying gateway identity and site-specific administrator authority.
// No unverified HTTP header can create this principal. GatewayAssertion verifies
// the gateway signature and request binding before installing site-specific grants.
type CommunicationPrincipal struct {
	GatewayAuthenticated bool
	Actor                string
	ReadSites            map[string]bool
	AdminSites           map[string]bool
	DiscoverySites       map[string]bool
	// ChannelSites authorizes communication channel operations (ChannelAuth).
	ChannelSites map[string]bool
}

const communicationPrincipalKey = "codexsecure.communication.principal"

func SetCommunicationPrincipal(c *gin.Context, p CommunicationPrincipal) {
	// Freeze request-local grants; the authentication provider's maps must not
	// change this request's authority after the principal is installed.
	p.ReadSites = copyCommunicationGrants(p.ReadSites)
	p.AdminSites = copyCommunicationGrants(p.AdminSites)
	p.DiscoverySites = copyCommunicationGrants(p.DiscoverySites)
	p.ChannelSites = copyCommunicationGrants(p.ChannelSites)
	c.Set(communicationPrincipalKey, p)
}
func copyCommunicationGrants(grants map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(grants))
	for site, allowed := range grants {
		copy[site] = allowed
	}
	return copy
}

// CommunicationActor returns the verified audit actor, never a client assertion.
func CommunicationActor(c *gin.Context) string {
	value, ok := c.Get(communicationPrincipalKey)
	p, valid := value.(CommunicationPrincipal)
	if !ok || !valid || !p.GatewayAuthenticated {
		return ""
	}
	return p.Actor
}

// CommunicationAuth is isolated from the legacy optional ServiceKeyMiddleware.
// A missing SERVICE_API_KEY always denies access, including configuration reads.
func CommunicationAuth() gin.HandlerFunc {
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
		site := c.Param("site")
		if c.Request.Method == http.MethodGet {
			if !p.ReadSites[site] && !p.AdminSites[site] {
				fail(http.StatusForbidden, "ADMIN_UNAUTHORIZED", "site access denied")
				return
			}
		} else {
			ingestion := strings.HasSuffix(c.FullPath(), "/discovery") || strings.Contains(c.FullPath(), "/discovery/batches")
			if ingestion {
				if !p.DiscoverySites[site] {
					fail(http.StatusForbidden, "ADMIN_UNAUTHORIZED", "discovery access denied")
					return
				}
			} else {
				actor := c.GetHeader("X-CodexSecure-Actor")
				if !p.AdminSites[site] || actor == "" || strings.TrimSpace(actor) != actor || utf8.RuneCountInString(actor) > 128 || strings.IndexFunc(actor, unicode.IsControl) >= 0 || actor != p.Actor {
					fail(http.StatusForbidden, "ADMIN_UNAUTHORIZED", "administrator authority is required")
					return
				}
			}
			replay := c.GetHeader("Idempotency-Key")
			parsed, err := uuid.Parse(replay)
			if err != nil || parsed == uuid.Nil {
				fail(http.StatusUnprocessableEntity, "CONFIGURATION_INVALID", "a UUID Idempotency-Key is required")
				return
			}
		}
		c.Next()
	}
}

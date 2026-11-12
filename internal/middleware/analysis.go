package middleware

// Analysis Middleware - Zone analysis enforcement
// Enforces zone access rules and validates zone configuration.
//
// Converted from: CodexZoneAnalysisMiddleware.php
// @version 2.0.0

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// AnalysisConfig configures analysis middleware behavior
type AnalysisConfig struct {
	// RequireZoneConfig aborts if zone config is missing
	RequireZoneConfig bool

	// EnforceAuth checks authentication requirements
	EnforceAuth bool

	// EnforceExecutive checks executive access for critical zones
	EnforceExecutive bool

	// SkipZones lists zones to skip analysis (e.g., health endpoints)
	SkipZones []int
}

// DefaultAnalysisConfig returns sensible defaults
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		RequireZoneConfig: true,
		EnforceAuth:       true,
		EnforceExecutive:  true,
		SkipZones:         []int{}, // Don't skip any by default
	}
}

// AnalysisMiddleware performs zone analysis and enforcement
func AnalysisMiddleware(cfg *AnalysisConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultAnalysisConfig()
	}

	return func(c *gin.Context) {
		zoneID := GetZoneID(c)

		// Skip analysis for configured zones
		for _, skip := range cfg.SkipZones {
			if zoneID == skip {
				c.Next()
				return
			}
		}

		// Check for unassigned zone
		if zoneID == analyzer.UnassignedZoneID {
			if cfg.RequireZoneConfig {
				log.Warn().
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("Unauthorized zone access - config missing")

				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Unauthorized Zone Access (ZoneConfig Missing)",
					"code":  "ZONE_CONFIG_MISSING",
					"path":  c.Request.URL.Path,
				})
				return
			}
		}

		// Get zone config for further checks
		zoneConfig := GetZoneConfig(c)
		if zoneConfig == nil && cfg.RequireZoneConfig {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Zone configuration not found",
				"code":  "ZONE_NOT_FOUND",
			})
			return
		}

		// Enforce authentication requirement
		if cfg.EnforceAuth && zoneConfig != nil && zoneConfig.Rules.AuthRequired {
			if !isAuthenticated(c) {
				log.Warn().
					Int("zone_id", zoneID).
					Str("zone_key", zoneConfig.Key).
					Msg("Authentication required for zone")

				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required",
					"code":  "AUTH_REQUIRED",
					"zone":  zoneConfig.Key,
				})
				return
			}
		}

		// Enforce executive access for critical zones
		if cfg.EnforceExecutive && zoneConfig != nil && zoneConfig.Rules.Executive {
			if !hasExecutiveAccess(c) {
				log.Warn().
					Int("zone_id", zoneID).
					Str("zone_key", zoneConfig.Key).
					Msg("Executive access required for zone")

				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Executive access required",
					"code":  "EXECUTIVE_REQUIRED",
					"zone":  zoneConfig.Key,
				})
				return
			}
		}

		c.Next()
	}
}

// isAuthenticated checks if user is authenticated
// In real implementation, check session/JWT token
func isAuthenticated(c *gin.Context) bool {
	// Check for auth token
	token := c.GetHeader("Authorization")
	if token != "" {
		return true
	}

	// Check for session cookie
	if _, err := c.Cookie("codex_session"); err == nil {
		return true
	}

	// Check context (may be set by auth middleware)
	if val, exists := c.Get("authenticated"); exists {
		if auth, ok := val.(bool); ok && auth {
			return true
		}
	}

	return false
}

// hasExecutiveAccess checks for executive/admin privileges
func hasExecutiveAccess(c *gin.Context) bool {
	// Check for executive role in context
	if val, exists := c.Get("user_role"); exists {
		if role, ok := val.(string); ok {
			return role == "admin" || role == "executive" || role == "superadmin"
		}
	}

	// Check for executive header (for service-to-service)
	if c.GetHeader("X-Executive-Access") != "" {
		return true
	}

	return false
}

// RequireAuth is a simple middleware that requires authentication
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthenticated(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			return
		}
		c.Next()
	}
}

// RequireExecutive is a simple middleware that requires executive access
func RequireExecutive() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasExecutiveAccess(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Executive access required",
				"code":  "FORBIDDEN",
			})
			return
		}
		c.Next()
	}
}

// RequireZone creates middleware that requires a specific zone
func RequireZone(requiredZoneID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		zoneID := GetZoneID(c)
		if zoneID != requiredZoneID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":         "Zone access denied",
				"code":          "ZONE_MISMATCH",
				"required_zone": requiredZoneID,
				"current_zone":  zoneID,
			})
			return
		}
		c.Next()
	}
}

// RequireZoneTier creates middleware that requires a minimum zone tier
func RequireZoneTier(minTier string) gin.HandlerFunc {
	tierPriority := map[string]int{
		policy.TierPublic:   1,
		policy.TierBusiness: 2,
		policy.TierCritical: 3,
	}

	return func(c *gin.Context) {
		currentTier := GetZoneTier(c)

		currentPriority, ok1 := tierPriority[currentTier]
		requiredPriority, ok2 := tierPriority[minTier]

		if !ok1 || !ok2 || currentPriority < requiredPriority {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":         "Zone tier access denied",
				"code":          "TIER_INSUFFICIENT",
				"required_tier": minTier,
				"current_tier":  currentTier,
			})
			return
		}
		c.Next()
	}
}

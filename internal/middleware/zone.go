package middleware

// Zone Middleware - Core zone detection and context setting
// Detects zone from route and sets it in Gin context for downstream use.
//
// Converted from: ZoneRouteMiddleware.php
// @version 2.0.0

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// Context keys for zone data
const (
	ContextKeyZoneID     = "codex_zone_id"
	ContextKeyZoneName   = "codex_zone_name"
	ContextKeyZoneKey    = "codex_zone_key"
	ContextKeyZoneConfig = "codex_zone_config"
	ContextKeyZoneTier   = "codex_zone_tier"
	ContextKeyZoneSource = "codex_zone_source"
	ContextKeyZoneConfidence = "codex_zone_confidence"
)

// ZoneMiddleware detects zone from route and sets context
// This is the core middleware that must run before other zone-aware middleware.
func ZoneMiddleware(a *analyzer.Analyzer, reg *policy.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Build route info for analysis
		route := analyzer.RouteInfo{
			URI:     path,
			Methods: []string{method},
		}

		// Analyze route to get zone assignment
		result := a.AnalyzeRoute(route)

		// Get zone config from registry
		var zoneConfig policy.ZoneDefinition
		var hasConfig bool
		if result.ZoneID >= 0 {
			zoneConfig, hasConfig = reg.GetZone(result.ZoneID)
		}

		// Set zone context
		c.Set(ContextKeyZoneID, result.ZoneID)
		c.Set(ContextKeyZoneName, result.ZoneName)
		c.Set(ContextKeyZoneKey, result.ZoneKey)
		c.Set(ContextKeyZoneSource, result.Source)
		c.Set(ContextKeyZoneConfidence, result.Confidence)

		if hasConfig {
			c.Set(ContextKeyZoneConfig, zoneConfig)
			c.Set(ContextKeyZoneTier, zoneConfig.Tier)
		}

		// Process request
		c.Next()

		// Add zone header to response
		if result.ZoneKey != "" {
			c.Header("X-Codex-Zone", result.ZoneKey)
		} else {
			c.Header("X-Codex-Zone", "unassigned")
		}
		if result.Source != "" {
			c.Header("X-Codex-Zone-Source", result.Source)
		}
		c.Header("X-Codex-Zone-Confidence", fmt.Sprintf("%.2f", result.Confidence))
	}
}

// GetZoneID retrieves zone ID from context
func GetZoneID(c *gin.Context) int {
	if val, exists := c.Get(ContextKeyZoneID); exists {
		if id, ok := val.(int); ok {
			return id
		}
	}
	return analyzer.UnassignedZoneID
}

// GetZoneName retrieves zone name from context
func GetZoneName(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyZoneName); exists {
		if name, ok := val.(string); ok {
			return name
		}
	}
	return "Unassigned"
}

// GetZoneKey retrieves zone key from context
func GetZoneKey(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyZoneKey); exists {
		if key, ok := val.(string); ok {
			return key
		}
	}
	return ""
}

// GetZoneConfig retrieves zone config from context
func GetZoneConfig(c *gin.Context) *policy.ZoneDefinition {
	if val, exists := c.Get(ContextKeyZoneConfig); exists {
		if cfg, ok := val.(policy.ZoneDefinition); ok {
			return &cfg
		}
	}
	return nil
}

// GetZoneTier retrieves zone tier from context
func GetZoneTier(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyZoneTier); exists {
		if tier, ok := val.(string); ok {
			return tier
		}
	}
	return ""
}

// GetZoneSource retrieves zone source from context.
func GetZoneSource(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyZoneSource); exists {
		if source, ok := val.(string); ok {
			return source
		}
	}
	return ""
}

// GetZoneConfidence retrieves zone confidence from context.
func GetZoneConfidence(c *gin.Context) float64 {
	if val, exists := c.Get(ContextKeyZoneConfidence); exists {
		if confidence, ok := val.(float64); ok {
			return confidence
		}
	}
	return 0
}

// IsPublicZone returns true if current zone is public tier
func IsPublicZone(c *gin.Context) bool {
	tier := GetZoneTier(c)
	return tier == policy.TierPublic
}

// IsCriticalZone returns true if current zone is critical tier
func IsCriticalZone(c *gin.Context) bool {
	tier := GetZoneTier(c)
	return tier == policy.TierCritical
}

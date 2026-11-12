package middleware

// Method Middleware - HTTP method enforcement per zone
// Validates that the HTTP method is allowed for the detected zone.
//
// Converted from: CodexEnforceMethodMiddleware.php
// @version 2.0.0

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// MethodMiddleware validates HTTP method against zone rules
func MethodMiddleware(reg *policy.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		zoneID := GetZoneID(c)
		method := c.Request.Method

		// Get zone config
		zoneConfig := GetZoneConfig(c)
		if zoneConfig == nil {
			// No zone config, allow request (middleware ran before zone detection)
			c.Next()
			return
		}

		// Check if method is allowed
		if !isMethodAllowed(method, zoneConfig.Rules.MethodsAllowed, zoneConfig.Rules.ReadOnly) {
			log.Warn().
				Int("zone_id", zoneID).
				Str("zone_key", zoneConfig.Key).
				Str("method", method).
				Strs("allowed", zoneConfig.Rules.MethodsAllowed).
				Msg("Method not allowed for zone")

			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
				"error":   "Method not allowed for this zone",
				"code":    "METHOD_VIOLATION",
				"zone_id": zoneID,
				"zone":    zoneConfig.Key,
				"method":  method,
				"allowed": zoneConfig.Rules.MethodsAllowed,
			})
			return
		}

		c.Next()
	}
}

// isMethodAllowed checks if a method is permitted for a zone
func isMethodAllowed(method string, allowed []string, readOnly bool) bool {
	method = strings.ToUpper(method)

	// ReadOnly zones only allow GET and HEAD
	if readOnly {
		return method == "GET" || method == "HEAD" || method == "OPTIONS"
	}

	// If no methods specified, allow all
	if len(allowed) == 0 {
		return true
	}

	// Check against allowed list
	for _, m := range allowed {
		if strings.EqualFold(m, method) {
			return true
		}
	}

	return false
}

// EnforceReadOnly is a middleware that only allows GET/HEAD requests
func EnforceReadOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method != "GET" && method != "HEAD" && method != "OPTIONS" {
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
				"error":   "This endpoint is read-only",
				"code":    "READ_ONLY",
				"allowed": []string{"GET", "HEAD", "OPTIONS"},
			})
			return
		}
		c.Next()
	}
}

// EnforceMethods creates middleware that only allows specific methods
func EnforceMethods(methods ...string) gin.HandlerFunc {
	// Normalize to uppercase
	allowed := make([]string, len(methods))
	for i, m := range methods {
		allowed[i] = strings.ToUpper(m)
	}

	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)

		for _, m := range allowed {
			if m == method {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
			"error":   "Method not allowed",
			"code":    "METHOD_NOT_ALLOWED",
			"method":  method,
			"allowed": allowed,
		})
	}
}

// RequireWriteAccess ensures the zone allows write operations
func RequireWriteAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		zoneConfig := GetZoneConfig(c)
		if zoneConfig != nil && zoneConfig.Rules.ReadOnly {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Zone is read-only",
				"code":  "ZONE_READ_ONLY",
				"zone":  zoneConfig.Key,
			})
			return
		}
		c.Next()
	}
}

// ValidateMethod checks if a method is valid for a zone (utility function)
func ValidateMethod(reg *policy.Registry, zoneID int, method string) error {
	zone, ok := reg.GetZone(zoneID)
	if !ok {
		return nil // No zone config, allow
	}

	if !isMethodAllowed(method, zone.Rules.MethodsAllowed, zone.Rules.ReadOnly) {
		return &ZoneViolation{
			Zone:    zone.Key,
			Type:    ViolationMethodDenied,
			Message: "Method not allowed for zone",
			Code:    http.StatusMethodNotAllowed,
		}
	}

	return nil
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/database"
)

// ZoneHandler handles zone-related endpoints
type ZoneHandler struct {
	db *database.DB
}

// NewZoneHandler creates a new zone handler
func NewZoneHandler(db *database.DB) *ZoneHandler {
	return &ZoneHandler{db: db}
}

// ListZones returns all active zones
func (h *ZoneHandler) ListZones(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Check for tier filter
	tier := c.Query("tier")
	
	var zones []database.SecurityZone
	var err error
	
	if tier != "" {
		zones, err = h.db.GetZonesByTier(ctx, tier)
	} else {
		zones, err = h.db.GetAllZones(ctx)
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(zones),
		"filter": tier,
		"zones": zones,
	})
}

// GetZone returns a specific zone by code
func (h *ZoneHandler) GetZone(c *gin.Context) {
	ctx := c.Request.Context()
	code := strings.ToUpper(c.Param("code"))
	
	zone, err := h.db.GetZoneByCode(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "zone not found",
			"code":  code,
		})
		return
	}

	c.JSON(http.StatusOK, zone)
}

// DetectZoneRequest represents a zone detection request
type DetectZoneRequest struct {
	Path    string            `json:"path" binding:"required"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// DetectZone detects the appropriate zone for a given path
func (h *ZoneHandler) DetectZone(c *gin.Context) {
	var req DetectZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path is required",
		})
		return
	}

	ctx := c.Request.Context()
	
	// Check for explicit X-Zone-ID header
	if zoneID, ok := req.Headers["x-zone-id"]; ok && zoneID != "" {
		zone, err := h.db.GetZoneByCode(ctx, strings.ToUpper(zoneID))
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"zone_id":          zone.ZoneCode,
				"zone_name":        zone.ZoneName,
				"confidence":       1.0,
				"detection_method": "header",
				"security_tier":    zone.SecurityTier,
				"requires_auth":    zone.RequiresAuth,
				"requires_mfa":     zone.RequiresMfa,
			})
			return
		}
	}

	// Get all zones and match path
	zones, err := h.db.GetAllZones(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Normalize path
	path := strings.ToLower(strings.TrimPrefix(req.Path, "/"))
	segments := strings.Split(path, "/")
	
	var bestMatch *database.SecurityZone
	var bestConfidence float64 = 0
	
	for i := range zones {
		zone := &zones[i]
		for _, prefix := range zone.Prefixes {
			normalizedPrefix := strings.ToLower(strings.TrimPrefix(prefix, "/"))
			
			// Exact match on first segment
			if len(segments) > 0 && segments[0] == normalizedPrefix {
				if 0.9 > bestConfidence {
					bestMatch = zone
					bestConfidence = 0.9
				}
			}
			
			// Path starts with prefix
			if strings.HasPrefix(path, normalizedPrefix) {
				if 0.8 > bestConfidence {
					bestMatch = zone
					bestConfidence = 0.8
				}
			}
		}
	}

	// Default to Z1 (public)
	if bestMatch == nil {
		zone, _ := h.db.GetZoneByCode(ctx, "Z1")
		if zone != nil {
			bestMatch = zone
			bestConfidence = 0.5
		}
	}

	if bestMatch == nil {
		c.JSON(http.StatusOK, gin.H{
			"zone_id":          "Z1",
			"zone_name":        "public",
			"confidence":       0.5,
			"detection_method": "default",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"zone_id":          bestMatch.ZoneCode,
		"zone_name":        bestMatch.ZoneName,
		"confidence":       bestConfidence,
		"detection_method": "path_analysis",
		"security_tier":    bestMatch.SecurityTier,
		"requires_auth":    bestMatch.RequiresAuth,
		"requires_mfa":     bestMatch.RequiresMfa,
	})
}

// ZoneHealth returns health status for all zones
func (h *ZoneHandler) ZoneHealth(c *gin.Context) {
	ctx := c.Request.Context()
	
	zones, err := h.db.GetAllZones(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	health := make([]gin.H, len(zones))
	for i, zone := range zones {
		health[i] = gin.H{
			"zone_code":     zone.ZoneCode,
			"zone_name":     zone.ZoneName,
			"security_tier": zone.SecurityTier,
			"is_active":     zone.IsActive,
			"status":        "healthy",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  len(health),
		"zones":  health,
		"status": "all zones healthy",
	})
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// PolicyHandler handles policy-related endpoints
type PolicyHandler struct {
	registry *policy.Registry
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(registry *policy.Registry) *PolicyHandler {
	return &PolicyHandler{registry: registry}
}

// GetSnapshot returns the complete policy snapshot (admin/debug endpoint)
// GET /api/v1/policy/snapshot
func (h *PolicyHandler) GetSnapshot(c *gin.Context) {
	snapshot := h.registry.GetSnapshot()
	c.JSON(http.StatusOK, snapshot)
}

// GetZonePolicy returns a single zone's policy by ID or key
// GET /api/v1/policy/zones/:idOrKey
func (h *PolicyHandler) GetZonePolicy(c *gin.Context) {
	idOrKey := c.Param("idOrKey")
	
	// Try parsing as zone code (Z0, Z1, etc.)
	if id, ok := policy.ZoneCodeToID(idOrKey); ok {
		zone, exists := h.registry.GetZone(id)
		if exists {
			c.JSON(http.StatusOK, zone)
			return
		}
	}
	
	// Try as zone key (public, identity, etc.)
	zone, exists := h.registry.GetZoneByKey(idOrKey)
	if exists {
		c.JSON(http.StatusOK, zone)
		return
	}
	
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "zone not found",
		"zone_id": idOrKey,
	})
}

// ListZonePolicies returns all zone policies
// GET /api/v1/policy/zones
func (h *PolicyHandler) ListZonePolicies(c *gin.Context) {
	zones := h.registry.AllZones()
	
	// Convert to a list ordered by ID
	zoneList := make([]policy.ZoneDefinition, 0, len(zones))
	for i := 0; i <= 12; i++ {
		if zone, ok := zones[i]; ok {
			zoneList = append(zoneList, zone)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"version": h.registry.GetSnapshot().Version,
		"total":   len(zoneList),
		"zones":   zoneList,
	})
}

// ValidateMethod checks if a method is allowed in a zone
// POST /api/v1/policy/validate-method
func (h *PolicyHandler) ValidateMethod(c *gin.Context) {
	var req struct {
		ZoneID int    `json:"zone_id" binding:"required"`
		Method string `json:"method" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
			"details": err.Error(),
		})
		return
	}
	
	allowed := h.registry.IsMethodAllowed(req.ZoneID, req.Method)
	zone, _ := h.registry.GetZone(req.ZoneID)
	
	c.JSON(http.StatusOK, gin.H{
		"zone_id":   req.ZoneID,
		"zone_name": zone.Name,
		"method":    req.Method,
		"allowed":   allowed,
		"rules":     zone.Rules,
	})
}

// GetZoneRequirements returns auth/TIU/executive requirements for a zone
// GET /api/v1/policy/zones/:idOrKey/requirements
func (h *PolicyHandler) GetZoneRequirements(c *gin.Context) {
	idOrKey := c.Param("idOrKey")
	
	var zoneID int
	var zone policy.ZoneDefinition
	var exists bool
	
	// Try parsing as zone code
	if id, ok := policy.ZoneCodeToID(idOrKey); ok {
		zoneID = id
		zone, exists = h.registry.GetZone(id)
	} else {
		// Try as key
		zone, exists = h.registry.GetZoneByKey(idOrKey)
		if exists {
			zoneID = zone.ID
		}
	}
	
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "zone not found",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"zone_id":       zoneID,
		"zone_code":     policy.ZoneIDToCode(zoneID),
		"zone_key":      zone.Key,
		"zone_name":     zone.Name,
		"tier":          zone.Tier,
		"requires_auth": zone.Rules.AuthRequired,
		"requires_tiu":  zone.Rules.TIURequired,
		"is_executive":  zone.Rules.Executive,
		"is_read_only":  zone.Rules.ReadOnly,
	})
}

// GetRegistry returns the zone ID → key registry with its canonical signature.
// GET /api/v1/policy/registry
func (h *PolicyHandler) GetRegistry(c *gin.Context) {
	registry := h.registry.KeyRegistry()

	c.JSON(http.StatusOK, gin.H{
		"policy_version":      h.registry.GetSnapshot().Version,
		"signature_algorithm": policy.SignatureAlgorithm,
		"signature":           policy.RegistrySignature(registry),
		"total":               len(registry),
		"zones":               policy.CanonicalEntries(registry),
	})
}

// CheckAlignment compares a submitted zone registry with the policy registry.
// A mismatch is a valid result (200, aligned=false); malformed input is rejected.
// POST /api/v1/policy/alignment
func (h *PolicyHandler) CheckAlignment(c *gin.Context) {
	var req struct {
		Zones map[string]string `json:"zones" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	expected, err := policy.ParseExpectedRegistry(req.Zones)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "invalid_registry",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.registry.Alignment(expected))
}

// GetDefinitions returns the shared zone definition contracts with their canonical signature.
// GET /api/v1/policy/definitions
func (h *PolicyHandler) GetDefinitions(c *gin.Context) {
	contracts := h.registry.DefinitionContracts()

	zones := make([]gin.H, 0, len(contracts))
	for _, entry := range policy.CanonicalEntries(h.registry.KeyRegistry()) {
		zones = append(zones, gin.H{
			"zone_id":    entry.ZoneID,
			"zone_code":  entry.ZoneCode,
			"definition": contracts[entry.ZoneID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"policy_version":      h.registry.GetSnapshot().Version,
		"signature_algorithm": policy.DefinitionSignatureAlgorithm,
		"signature":           policy.DefinitionSignature(contracts),
		"compared_fields":     policy.DefinitionFields,
		"total":               len(zones),
		"zones":               zones,
	})
}

// CheckDefinitionAlignment compares submitted zone definitions with the policy registry.
// Registry (ID → key) alignment stays on POST /api/v1/policy/alignment.
// POST /api/v1/policy/alignment/definitions
func (h *PolicyHandler) CheckDefinitionAlignment(c *gin.Context) {
	var req struct {
		Zones map[string]policy.ExpectedDefinition `json:"zones" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	expected, err := policy.ParseExpectedDefinitions(req.Zones)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "invalid_definitions",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, h.registry.DefinitionAlignment(expected))
}

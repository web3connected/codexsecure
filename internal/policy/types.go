// Package policy defines the zone policy registry and rule definitions.
// This is the Go equivalent of CodexRouteServer.php - a policy catalog,
// NOT an HTTP server.
package policy

// Version is the current policy schema version
const Version = "3.1.0"

// RouteConfig defines system-level route configuration
type RouteConfig struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Method       string         `json:"method"`
	Path         string         `json:"path"`
	AuthRequired bool           `json:"auth_required"`
	Meta         map[string]any `json:"meta,omitempty"`
}

// ZoneRules defines the enforcement rules for a zone
type ZoneRules struct {
	AuthRequired   bool     `json:"auth_required"`
	TIURequired    bool     `json:"tiu_required"`    // Time Integrity Unit sync required
	ReadOnly       bool     `json:"read_only"`       // If true, only GET allowed
	Executive      bool     `json:"executive"`       // If true, requires privileged access
	MethodsAllowed []string `json:"methods_allowed"` // HTTP methods permitted
}

// ZoneDefinition defines a complete zone with its rules and metadata
type ZoneDefinition struct {
	ID          int            `json:"id"`           // Technical identifier: 0 (Boundary) or 1-12 (operational zones)
	Key         string         `json:"key"`          // Machine-readable key (e.g., "public", "identity")
	Aliases     []string       `json:"aliases,omitempty"` // Earlier keys for this same zone; never another traffic class
	Name        string         `json:"name"`         // Display name (e.g., "Public", "Identity")
	Description string         `json:"description"`  // Human-readable description
	Role        string         `json:"role,omitempty"` // Structural role (set for the Z0 Boundary)
	Rules       ZoneRules      `json:"rules"`        // Enforcement rules
	Tier        string         `json:"tier"`         // Security tier: public, business, critical
	Subzones    map[string]any `json:"subzones,omitempty"` // Future: typed subzone model
}

// PolicySnapshot represents the complete policy state at a point in time
type PolicySnapshot struct {
	Version   string                  `json:"version"`
	System    RouteConfig             `json:"system"`
	Zones     map[int]ZoneDefinition  `json:"zones"`
	LoadedAt  string                  `json:"loaded_at,omitempty"`
}

// ZoneTier constants for security classification
const (
	TierPublic   = "public"   // Z0 Boundary and Z1: Low trust, high filtering
	TierBusiness = "business" // Z2-Z10: Standard auth/validation
	TierCritical = "critical" // Z11-Z12: Executive access only
)

// HTTPErrors maps HTTP status codes to descriptions
type HTTPErrors map[string]string

// DefaultHTTPErrors returns standard HTTP error mappings
func DefaultHTTPErrors() HTTPErrors {
	return HTTPErrors{
		"303": "See Other",
		"400": "Bad Request",
		"401": "Unauthorized",
		"402": "Payment Required",
		"403": "Forbidden",
		"404": "Not Found",
		"405": "Method Not Allowed",
		"500": "Internal Server Error",
	}
}

// Registry is the in-memory policy store
type Registry struct {
	snapshot *PolicySnapshot
}

// NewRegistry creates a new policy registry with default zones
func NewRegistry() *Registry {
	return &Registry{
		snapshot: DefaultPolicySnapshot(),
	}
}

// GetSnapshot returns the current policy snapshot
func (r *Registry) GetSnapshot() *PolicySnapshot {
	return r.snapshot
}

// GetZone returns a zone definition by ID
func (r *Registry) GetZone(id int) (ZoneDefinition, bool) {
	zone, ok := r.snapshot.Zones[id]
	return zone, ok
}

// HasKey reports whether key names this zone, by its canonical key or an alias.
func (z ZoneDefinition) HasKey(key string) bool {
	key = NormalizeZoneKey(key)
	if key == "" {
		return false
	}
	if NormalizeZoneKey(z.Key) == key {
		return true
	}
	for _, alias := range z.Aliases {
		if NormalizeZoneKey(alias) == key {
			return true
		}
	}
	return false
}

// GetZoneByKey returns a zone definition by its canonical key or an alias
// (e.g. "administration" or "admin" for Z12).
func (r *Registry) GetZoneByKey(key string) (ZoneDefinition, bool) {
	for _, zone := range r.snapshot.Zones {
		if zone.HasKey(key) {
			return zone, true
		}
	}
	return ZoneDefinition{}, false
}

// AllZones returns all zone definitions
func (r *Registry) AllZones() map[int]ZoneDefinition {
	return r.snapshot.Zones
}

// IsMethodAllowed checks if an HTTP method is allowed in a zone
func (r *Registry) IsMethodAllowed(zoneID int, method string) bool {
	zone, ok := r.GetZone(zoneID)
	if !ok {
		return false
	}
	
	// If read_only, only GET is allowed regardless of methods_allowed
	if zone.Rules.ReadOnly && method != "GET" && method != "OPTIONS" {
		return false
	}
	
	for _, allowed := range zone.Rules.MethodsAllowed {
		if allowed == method {
			return true
		}
	}
	return false
}

// RequiresAuth checks if a zone requires authentication
func (r *Registry) RequiresAuth(zoneID int) bool {
	zone, ok := r.GetZone(zoneID)
	if !ok {
		return true // Safe default
	}
	return zone.Rules.AuthRequired
}

// RequiresTIU checks if a zone requires Time Integrity Unit sync
func (r *Registry) RequiresTIU(zoneID int) bool {
	zone, ok := r.GetZone(zoneID)
	if !ok {
		return true // Safe default
	}
	return zone.Rules.TIURequired
}

// IsExecutive checks if a zone requires executive privileges
func (r *Registry) IsExecutive(zoneID int) bool {
	zone, ok := r.GetZone(zoneID)
	if !ok {
		return true // Safe default
	}
	return zone.Rules.Executive
}

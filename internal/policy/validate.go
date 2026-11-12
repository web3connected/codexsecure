package policy

import (
	"fmt"
	"strings"
)

// ValidationError represents a policy validation failure
type ValidationError struct {
	ZoneID  int
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.ZoneID >= 0 {
		return fmt.Sprintf("zone %d (%s): %s", e.ZoneID, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds all validation errors
type ValidationResult struct {
	Errors []ValidationError
}

func (r *ValidationResult) AddError(zoneID int, field, message string) {
	r.Errors = append(r.Errors, ValidationError{
		ZoneID:  zoneID,
		Field:   field,
		Message: message,
	})
}

func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

func (r *ValidationResult) Error() string {
	if len(r.Errors) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// ValidMethods defines all valid HTTP methods
var ValidMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"OPTIONS": true,
	"HEAD":    true,
}

// ValidateZoneDefinition validates a single zone definition
func ValidateZoneDefinition(zone ZoneDefinition) *ValidationResult {
	result := &ValidationResult{}

	// Validate ID range
	if zone.ID < 0 || zone.ID > 12 {
		result.AddError(zone.ID, "id", fmt.Sprintf("zone ID must be 0-12, got %d", zone.ID))
	}

	// Validate key is set
	if zone.Key == "" {
		result.AddError(zone.ID, "key", "zone key cannot be empty")
	}

	// Validate name is set
	if zone.Name == "" {
		result.AddError(zone.ID, "name", "zone name cannot be empty")
	}

	// Validate tier
	switch zone.Tier {
	case TierPublic, TierBusiness, TierCritical:
		// Valid
	case "":
		result.AddError(zone.ID, "tier", "zone tier cannot be empty")
	default:
		result.AddError(zone.ID, "tier", fmt.Sprintf("invalid tier '%s', must be public/business/critical", zone.Tier))
	}

	// Validate methods_allowed
	if len(zone.Rules.MethodsAllowed) == 0 {
		result.AddError(zone.ID, "methods_allowed", "at least one HTTP method must be allowed")
	}

	for _, method := range zone.Rules.MethodsAllowed {
		if !ValidMethods[method] {
			result.AddError(zone.ID, "methods_allowed", fmt.Sprintf("invalid HTTP method '%s'", method))
		}
	}

	// Validate read_only consistency
	if zone.Rules.ReadOnly {
		for _, method := range zone.Rules.MethodsAllowed {
			if method != "GET" && method != "OPTIONS" && method != "HEAD" {
				result.AddError(zone.ID, "read_only",
					fmt.Sprintf("read_only=true but method '%s' is allowed", method))
			}
		}
	}

	// Validate executive zones have auth
	if zone.Rules.Executive && !zone.Rules.AuthRequired {
		result.AddError(zone.ID, "executive",
			"executive=true requires auth_required=true")
	}

	// Validate critical tier zones are protected
	if zone.Tier == TierCritical {
		if !zone.Rules.AuthRequired {
			result.AddError(zone.ID, "tier",
				"critical tier zones must have auth_required=true")
		}
		if !zone.Rules.TIURequired {
			result.AddError(zone.ID, "tier",
				"critical tier zones must have tiu_required=true")
		}
	}

	return result
}

// ValidatePolicySnapshot validates the entire policy snapshot
func ValidatePolicySnapshot(snapshot *PolicySnapshot) *ValidationResult {
	result := &ValidationResult{}

	// Validate version
	if snapshot.Version == "" {
		result.AddError(-1, "version", "policy version cannot be empty")
	}

	// Validate system config
	if snapshot.System.Path == "" {
		result.AddError(-1, "system.path", "system route path cannot be empty")
	}

	// Validate all zones
	if len(snapshot.Zones) == 0 {
		result.AddError(-1, "zones", "at least one zone must be defined")
	}

	// Require the Z0 Boundary and the twelve operational zones
	for i := BoundaryZoneID; i <= OperationalZoneCount; i++ {
		if _, ok := snapshot.Zones[i]; !ok {
			result.AddError(-1, "zones", fmt.Sprintf("zone %d is missing from policy", i))
		}
	}

	// Validate each zone
	for id, zone := range snapshot.Zones {
		// Check ID matches map key
		if zone.ID != id {
			result.AddError(id, "id", fmt.Sprintf("zone ID %d doesn't match map key %d", zone.ID, id))
		}

		// Run zone-level validation
		zoneResult := ValidateZoneDefinition(zone)
		result.Errors = append(result.Errors, zoneResult.Errors...)
	}

	// Validate zone key uniqueness; aliases share the namespace so a name resolves to one zone
	keysSeen := make(map[string]int)
	for id, zone := range snapshot.Zones {
		for _, key := range append([]string{zone.Key}, zone.Aliases...) {
			key = NormalizeZoneKey(key)
			if existingID, exists := keysSeen[key]; exists {
				result.AddError(id, "key",
					fmt.Sprintf("duplicate zone key '%s' (also used by zone %d)", key, existingID))
			}
			keysSeen[key] = id
		}
	}

	return result
}

// ValidateAndPanic validates the policy and panics if invalid
// Use at startup to fail fast on bad configuration
func ValidateAndPanic(snapshot *PolicySnapshot) {
	result := ValidatePolicySnapshot(snapshot)
	if !result.IsValid() {
		panic(fmt.Sprintf("invalid policy configuration: %s", result.Error()))
	}
}

// MustValidate validates the policy and returns an error if invalid
func MustValidate(snapshot *PolicySnapshot) error {
	result := ValidatePolicySnapshot(snapshot)
	if !result.IsValid() {
		return fmt.Errorf("policy validation failed: %s", result.Error())
	}
	return nil
}

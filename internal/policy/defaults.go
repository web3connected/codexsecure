package policy

import (
	"fmt"
	"time"
)

// DefaultSystemRouteConfig returns the system-level route configuration
func DefaultSystemRouteConfig() RouteConfig {
	return RouteConfig{
		Name:         "System Route Config",
		Description:  "System-level route configuration.",
		Method:       "GET",
		Path:         "/system/route-config",
		AuthRequired: false,
		Meta: map[string]any{
			"http_errors": DefaultHTTPErrors(),
		},
	}
}

// DefaultZones returns the canonical zone registry: the Z0 Boundary plus the twelve
// operational zones Z1–Z12. This is the single source of zone identity in Go;
// analyzer keys, alignment and pipeline lookups all resolve against it.
//
// Zones are classes of internet traffic, not user roles or site departments.
// Z0 is not a zone: it is the entry boundary and unassigned state for
// communication without an operational zone or a valid zone relationship.
// Quarantine and rejection are separate security outcomes, not Z0's meaning.
//
// Aliases are earlier names for the same zone and traffic class only; zones
// whose responsibility changed (Z8, Z10, Z11) have none.
func DefaultZones() map[int]ZoneDefinition {
	return map[int]ZoneDefinition{
		0: {
			ID:          0,
			Key:         "boundary",
			Aliases:     []string{"lobby", "negative"}, // Web3Codex SDK and the 2.0.0 registry
			Name:        "Boundary",
			Description: "Entry boundary and unassigned state for communication that has not been assigned to an operational zone or has not established a valid zone relationship.",
			Role:        "Outside the twelve operational zones; not an operational destination. Quarantine and rejection are separate security outcomes.",
			Tier:        TierPublic,
			Rules: ZoneRules{
				AuthRequired:   false,
				TIURequired:    false,
				ReadOnly:       true,
				Executive:      false,
				MethodsAllowed: []string{"GET"}, // No write operations at the boundary
			},
		},
		1: {
			ID:          1,
			Key:         "public",
			Name:        "Public",
			Description: "Open information exchange, public content, and guest-facing communications.",
			Tier:        TierPublic,
			Rules: ZoneRules{
				AuthRequired:   false,
				TIURequired:    false,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "OPTIONS"},
			},
		},
		2: {
			ID:          2,
			Key:         "identity",
			Name:        "Identity",
			Description: "Authentication, credentials, identity verification, and identity lifecycle communications.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   false, // Login forms don't require auth
				TIURequired:    true,  // Codex secure sync required
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "OPTIONS"},
			},
		},
		3: {
			ID:          3,
			Key:         "operations",
			Aliases:     []string{"ops"},
			Name:        "Operations",
			Description: "Operational commands, account-owned resource management, and delegated control actions by authorized representatives or services.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			},
		},
		4: {
			ID:          4,
			Key:         "protected",
			Name:        "Protected Data",
			Description: "Controlled exchange of confidential or sensitive data and protected resources.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"}, // No DELETE
			},
		},
		5: {
			ID:          5,
			Key:         "processing",
			Aliases:     []string{"system"},
			Name:        "Processing",
			Description: "Execution requests, background jobs, worker tasks, scheduling, and workflows.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			},
		},
		6: {
			ID:          6,
			Key:         "logging",
			Aliases:     []string{"eventlogs"},
			Name:        "Logging",
			Description: "Audit events, diagnostic logs, traces, and operational telemetry.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    false,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "OPTIONS"}, // No PUT/PATCH/DELETE - logs shouldn't be editable
			},
		},
		7: {
			ID:          7,
			Key:         "communications",
			Aliases:     []string{"comms"},
			Name:        "Communications",
			Description: "Messages, notifications, alerts, and interactive communications. Distinct from CodexSecure Communication Objects, which govern approved interfaces across zones.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    false,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "OPTIONS"}, // Fire-and-forget
			},
		},
		8: {
			ID:          8,
			Key:         "payments",
			Name:        "Payments",
			Description: "Financial transaction traffic, including checkout, billing, subscriptions, refunds, and settlement interactions.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"},
			},
		},
		9: {
			ID:          9,
			Key:         "analytics",
			Name:        "Analytics",
			Description: "Read-oriented reports, measurements, statistics, and visualized metrics.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    false,
				ReadOnly:       true, // This zone never modifies data
				Executive:      false,
				MethodsAllowed: []string{"GET", "OPTIONS"},
			},
		},
		10: {
			ID:          10,
			Key:         "streaming",
			Name:        "Streaming",
			Description: "Continuous audio, video, live broadcasts, and media-stream delivery.",
			Tier:        TierBusiness,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      false,
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"}, // No DELETE
			},
		},
		11: {
			ID:          11,
			Key:         "synchronization",
			Name:        "Synchronization",
			Description: "Distributed state exchange, replication, peer coordination, and consistency-oriented synchronization traffic.",
			Tier:        TierCritical,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      true, // Only privileged actors can modify
				MethodsAllowed: []string{"GET", "POST", "PATCH", "OPTIONS"},
			},
		},
		12: {
			ID:          12,
			Key:         "administration",
			Aliases:     []string{"admin"},
			Name:        "Administration",
			Description: "Privileged platform-wide configuration, oversight, and administrative control communications.",
			Tier:        TierCritical,
			Rules: ZoneRules{
				AuthRequired:   true,
				TIURequired:    true,
				ReadOnly:       false,
				Executive:      true, // Only executive/programmatic control
				MethodsAllowed: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			},
		},
	}
}

// DefaultPolicySnapshot creates a complete policy snapshot with defaults
func DefaultPolicySnapshot() *PolicySnapshot {
	return &PolicySnapshot{
		Version:  Version,
		System:   DefaultSystemRouteConfig(),
		Zones:    DefaultZones(),
		LoadedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// ZoneKeyMap returns a map of zone keys to zone IDs for fast lookup.
// Derived from DefaultZones so keys have a single definition.
func ZoneKeyMap() map[string]int {
	zones := DefaultZones()
	keys := make(map[string]int, len(zones))
	for id, zone := range zones {
		keys[zone.Key] = id
	}
	return keys
}

// BoundaryZoneID is the Z0 technical identifier: the entry boundary, not a zone.
const BoundaryZoneID = 0

// OperationalZoneCount is the number of zones (Z1–Z12). The registry also holds Z0.
const OperationalZoneCount = 12

// IsOperationalZone reports whether id is one of the twelve operational zones.
func IsOperationalZone(id int) bool {
	return id >= 1 && id <= OperationalZoneCount
}

// ZoneIDToCode returns the zone code (Z0, Z1, etc.) for a zone ID
func ZoneIDToCode(id int) string {
	if id < 0 || id > 12 {
		return "Z?"
	}
	return fmt.Sprintf("Z%d", id)
}

// ZoneCodeToID parses a zone code (Z0, Z1, etc.) to a zone ID
func ZoneCodeToID(code string) (int, bool) {
	if len(code) < 2 || code[0] != 'Z' {
		return -1, false
	}
	
	idStr := code[1:]
	var id int
	for _, c := range idStr {
		if c < '0' || c > '9' {
			return -1, false
		}
		id = id*10 + int(c-'0')
	}
	
	if id < 0 || id > 12 {
		return -1, false
	}
	return id, true
}

package analyzer

// Analyzer Types
// Structs for route analysis and zone assignment scoring.
//
// Converted from: CodexRouteAnalyzer.php
// @version 2.0.0

const (
	// SystemZoneID is the default zone for system routes
	SystemZoneID = 5

	// UnassignedZoneID indicates no zone was matched
	UnassignedZoneID = -1

	// MaxUnassignedWarnings limits log spam for unassigned routes
	MaxUnassignedWarnings = 10
)

// ActionInfo represents a parsed route action (controller or closure)
type ActionInfo struct {
	Controller string `json:"controller"`
	Method     string `json:"method"`
	Type       string `json:"type"` // "controller" or "closure"
	Raw        string `json:"raw,omitempty"`
}

// MatchScoreDetails tracks which matching strategies succeeded
type MatchScoreDetails struct {
	URIMatch        bool `json:"uri_match"`
	ControllerMatch bool `json:"controller_match"`
	MiddlewareMatch bool `json:"middleware_match"`
}

// RouteInfo represents a route to be analyzed
type RouteInfo struct {
	URI        string            `json:"uri"`
	Methods    []string          `json:"methods"`
	Action     ActionInfo        `json:"action"`
	Middleware []string          `json:"middleware"`
	Name       string            `json:"name,omitempty"`
	Details    MatchScoreDetails `json:"match_score_details"`
}

// ZoneAssignment is the result of zone detection for a route
type ZoneAssignment struct {
	ZoneID   int               `json:"zone_id"`
	ZoneName string            `json:"zone_name"`
	ZoneKey  string            `json:"zone_key"`
	Source   string            `json:"source"`
	Confidence float64         `json:"confidence"`
	Score    int               `json:"score"`
	Details  MatchScoreDetails `json:"details"`
}

// AnalyzedRoute combines RouteInfo with its ZoneAssignment
type AnalyzedRoute struct {
	RouteInfo
	ZoneAssignment
}

// ZoneGroup groups routes by their assigned zone
type ZoneGroup struct {
	ZoneID   int             `json:"zone_id"`
	ZoneName string          `json:"zone_name"`
	ZoneKey  string          `json:"zone_key"`
	Routes   []AnalyzedRoute `json:"routes"`
	Count    int             `json:"count"`
}

// AnalysisStats provides statistics about the analysis
type AnalysisStats struct {
	TotalRoutes      int     `json:"total_routes"`
	AssignedRoutes   int     `json:"assigned_routes"`
	UnassignedRoutes int     `json:"unassigned_routes"`
	AssignmentRate   float64 `json:"assignment_rate"`
	ZonesUsed        int     `json:"zones_used"` // operational zones (Z1–Z12) with routes; excludes the Z0 Boundary
}

// AnalysisResult is the complete output of route analysis
type AnalysisResult struct {
	Routes []AnalyzedRoute      `json:"routes"`
	Zones  map[string]ZoneGroup `json:"zones"`
	Stats  AnalysisStats        `json:"stats"`
}

// AnalyzeRequest is the input for batch route analysis
type AnalyzeRequest struct {
	Routes []RouteInfo `json:"routes"`
}

// AnalyzeRouteRequest is the input for single route analysis
type AnalyzeRouteRequest struct {
	URI        string   `json:"uri" binding:"required"`
	Methods    []string `json:"methods"`
	Controller string   `json:"controller,omitempty"`
	Method     string   `json:"method,omitempty"`
	Middleware []string `json:"middleware,omitempty"`
}

// MatchStrategies holds all matching strategy configurations
type MatchStrategies struct {
	URI        URIStrategies        `json:"uri"`
	Controller ControllerStrategies `json:"controller"`
	Middleware MiddlewareStrategies `json:"middleware"`
}

// URIStrategies maps URI patterns to zone IDs
type URIStrategies struct {
	Patterns map[string]int `json:"patterns"` // pattern -> zone_id
}

// ControllerStrategies maps controller name patterns to zones
type ControllerStrategies struct {
	Patterns map[string]int `json:"patterns"` // pattern -> zone_id
}

// MiddlewareStrategies maps middleware combinations to zones
type MiddlewareStrategies struct {
	ZoneMiddleware map[string][]string `json:"zone_middleware"` // zone_name -> []middleware
}

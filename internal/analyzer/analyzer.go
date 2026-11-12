package analyzer

// Analyzer - Route zone assignment engine
// Analyzes routes to determine their zone assignments based on URI, controller, and middleware.
//
// Converted from: CodexRouteAnalyzer.php
// @version 2.0.0

import (
	"log"
	"strings"
	"sync"

	"github.com/web3connected/codexsecure-api/internal/policy"
)

// Analyzer determines zone assignments for routes
type Analyzer struct {
	registry           *policy.Registry
	strategies         *MatchStrategies
	uriStrategies      []ZoneKeyword
	vocabularies       []ZoneVocabulary  // canonical zone vocabulary registry
	unassignedWarnings int
	mu                 sync.RWMutex
}

// NewAnalyzer creates a new route analyzer
func NewAnalyzer(registry *policy.Registry) *Analyzer {
	return &Analyzer{
		registry:      registry,
		strategies:    DefaultMatchStrategies(registry),
		uriStrategies: DefaultURIStrategies(),
		vocabularies:  DefaultVocabularies(),
	}
}

// AnalyzeRoute determines the zone assignment for a single route
func (a *Analyzer) AnalyzeRoute(route RouteInfo) AnalyzedRoute {
	assignment := a.determineZoneAssignment(route)

	// Log warning for unassigned routes (limited)
	if assignment.ZoneID == UnassignedZoneID {
		a.mu.Lock()
		if a.unassignedWarnings < MaxUnassignedWarnings {
			log.Printf("WARN: Zone not assigned for route: %s", route.URI)
			a.unassignedWarnings++
		}
		a.mu.Unlock()
	}

	return AnalyzedRoute{
		RouteInfo:      route,
		ZoneAssignment: assignment,
	}
}

// AnalyzeRoutes analyzes multiple routes and returns complete analysis
func (a *Analyzer) AnalyzeRoutes(routes []RouteInfo) AnalysisResult {
	analyzed := make([]AnalyzedRoute, 0, len(routes))

	for _, route := range routes {
		analyzed = append(analyzed, a.AnalyzeRoute(route))
	}

	return AnalysisResult{
		Routes: analyzed,
		Zones:  a.groupRoutesByZone(analyzed),
		Stats:  a.generateStats(analyzed),
	}
}

// determineZoneAssignment finds the best matching zone for a route.
//
// Stage 1 — Vocabulary URI scoring (primary, replaces flat keyword matching):
//   ScoreURIWithVocabulary() walks segments left-to-right.
//   Governing keywords (admin, auth, api…) lock the zone immediately.
//   Non-governing words are scored against each zone's primary vocabulary.
//   First-segment matches score highest (200+), child segments score lower (100+).
//
// Stage 2 — Controller pattern matching (unchanged).
// Stage 3 — Middleware stack matching (unchanged).
//
// If a governing keyword locked Stage 1, Stages 2+3 add confirmatory detail
// but cannot change the zone assignment.
func (a *Analyzer) determineZoneAssignment(route RouteInfo) ZoneAssignment {
	if compositeAssignment, ok := a.detectCompositeAssignment(route); ok {
		return compositeAssignment
	}

	zones := a.registry.GetSnapshot().Zones
	middlewareStrategies := DefaultMiddlewareStrategies()
	controllerPatterns := a.strategies.Controller.Patterns

	// ── Stage 1: Vocabulary URI scoring ─────────────────────────────────────
	vocabResult := ScoreURIWithVocabulary(route.URI, a.vocabularies)

	// If a governing keyword locked the zone, return immediately.
	// Controller + middleware details are appended for observability but zone is final.
	if vocabResult.Governed && vocabResult.ZoneID != UnassignedZoneID {
		zone, ok := a.registry.GetZone(vocabResult.ZoneID)
		if !ok {
			// Zone ID from vocabulary has no policy definition — fall through to scoring
			goto scoringFallback
		}
		details := MatchScoreDetails{URIMatch: true}

		// Collect Stage 2+3 signals as metadata (zone already locked)
		if route.Action.Controller != "" {
			if _, matched := MatchController(route.Action.Controller, controllerPatterns); matched {
				details.ControllerMatch = true
			}
		}
		if zoneMiddleware, ok := middlewareStrategies[zone.Key]; ok {
			if MatchMiddleware(route.Middleware, zoneMiddleware) {
				details.MiddlewareMatch = true
			}
		}

		return ZoneAssignment{
			ZoneID:     zone.ID,
			ZoneName:   zone.Name,
			ZoneKey:    zone.Key,
			Source:     "vocabulary_governed",
			Confidence: scoreConfidence(vocabResult.Score),
			Score:      vocabResult.Score,
			Details:    details,
		}
	}

scoringFallback:
	// ── Stages 1+2+3: Combined scoring (no governing lock) ───────────────────
	best := ZoneAssignment{
		ZoneID:     UnassignedZoneID,
		ZoneName:   "unassigned",
		ZoneKey:    "unassigned",
		Source:     "unassigned",
		Confidence: 0,
		Score:      0,
		Details:    route.Details,
	}

	for _, zone := range zones {
		score := 0
		details := MatchScoreDetails{}

		// Stage 1: vocabulary URI score — use result computed above if zone matches
		if vocabResult.ZoneID == zone.ID && vocabResult.Score > 0 {
			score += vocabResult.Score
			details.URIMatch = true
		}

		// Stage 2: controller name matching
		if route.Action.Controller != "" {
			if matchedZone, ok := MatchController(route.Action.Controller, controllerPatterns); ok && matchedZone == zone.ID {
				score += 30
				details.ControllerMatch = true
			}
		}

		// Stage 3: middleware stack matching
		if zoneMiddleware, ok := middlewareStrategies[zone.Key]; ok {
			if MatchMiddleware(route.Middleware, zoneMiddleware) {
				score += 20
				details.MiddlewareMatch = true
			}
		}

		if score > best.Score {
			best = ZoneAssignment{
				ZoneID:     zone.ID,
				ZoneName:   zone.Name,
				ZoneKey:    zone.Key,
				Source:     assignmentSource(details),
				Confidence: scoreConfidence(score),
				Score:      score,
				Details:    details,
			}
		}
	}

	return best
}

func (a *Analyzer) detectCompositeAssignment(route RouteInfo) (ZoneAssignment, bool) {
	normalizedPath := strings.ToLower(strings.Trim(route.URI, "/"))
	if normalizedPath == "" {
		return ZoneAssignment{}, false
	}

	segments := strings.Split(normalizedPath, "/")

	// Explicit zone module prefix: allow optional leading segments like /api/v1.
	// Examples:
	// - /zones/z2/login
	// - /api/v1/zones/z2/login
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] != "zones" {
			continue
		}
		if zoneID, ok := policy.ZoneCodeToID(strings.ToUpper(segments[i+1])); ok {
			zone, ok := a.registry.GetZone(zoneID)
			if !ok {
				return ZoneAssignment{}, false
			}

			return ZoneAssignment{
				ZoneID:     zone.ID,
				ZoneName:   zone.Name,
				ZoneKey:    zone.Key,
				Source:     "route_module",
				Confidence: 1.0,
				Score:      1000,
				Details: MatchScoreDetails{
					URIMatch: true,
				},
			}, true
		}
	}

	return ZoneAssignment{}, false
}

func assignmentSource(details MatchScoreDetails) string {
	switch {
	case details.URIMatch && details.ControllerMatch && details.MiddlewareMatch:
		return "composite_match"
	case details.URIMatch && details.ControllerMatch:
		return "uri_controller_match"
	case details.URIMatch:
		return "uri_keyword"
	case details.ControllerMatch:
		return "controller_pattern"
	case details.MiddlewareMatch:
		return "middleware_pattern"
	default:
		return "heuristic"
	}
}

func scoreConfidence(score int) float64 {
	if score <= 0 {
		return 0
	}
	confidence := float64(score) / 250.0
	if confidence > 1 {
		return 1
	}
	return confidence
}

// DetectRouteZone is a quick zone detection (returns zone key)
func (a *Analyzer) DetectRouteZone(uri, controller string) string {
	// Fast path for root URL
	if uri == "/" {
		return "public"
	}

	// URI-based detection
	if zoneID, zoneKey, found := FindKeywordZone(uri, a.uriStrategies); found {
		_ = zoneID
		return zoneKey
	}

	// Controller-based fallback
	if controller != "" {
		if zoneID, matched := MatchController(controller, a.strategies.Controller.Patterns); matched {
			if zone, ok := a.registry.GetZone(zoneID); ok {
				return zone.Key
			}
		}
	}

	return "unknown"
}

// groupRoutesByZone groups analyzed routes by their zone
func (a *Analyzer) groupRoutesByZone(routes []AnalyzedRoute) map[string]ZoneGroup {
	grouped := make(map[string]ZoneGroup)

	for _, route := range routes {
		zoneName := route.ZoneName
		if zoneName == "" {
			zoneName = "unassigned"
		}

		group, exists := grouped[zoneName]
		if !exists {
			group = ZoneGroup{
				ZoneID:   route.ZoneID,
				ZoneName: route.ZoneName,
				ZoneKey:  route.ZoneKey,
				Routes:   []AnalyzedRoute{},
				Count:    0,
			}
		}

		group.Routes = append(group.Routes, route)
		group.Count++
		grouped[zoneName] = group
	}

	return grouped
}

// generateStats calculates analysis statistics
func (a *Analyzer) generateStats(routes []AnalyzedRoute) AnalysisStats {
	total := len(routes)
	assigned := 0
	zonesUsed := make(map[int]bool)

	for _, route := range routes {
		if route.ZoneID != UnassignedZoneID {
			assigned++
			if policy.IsOperationalZone(route.ZoneID) { // the Z0 Boundary is not a zone
				zonesUsed[route.ZoneID] = true
			}
		}
	}

	rate := float64(0)
	if total > 0 {
		rate = float64(assigned) / float64(total) * 100
	}

	return AnalysisStats{
		TotalRoutes:      total,
		AssignedRoutes:   assigned,
		UnassignedRoutes: total - assigned,
		AssignmentRate:   rate,
		ZonesUsed:        len(zonesUsed),
	}
}

// IsRouteInZone checks if a route belongs to a specific zone
func (a *Analyzer) IsRouteInZone(route RouteInfo, zoneName string) bool {
	analyzed := a.AnalyzeRoute(route)
	return analyzed.ZoneName == zoneName
}

// GetRouteZone returns the full zone assignment for a route
func (a *Analyzer) GetRouteZone(route RouteInfo) ZoneAssignment {
	analyzed := a.AnalyzeRoute(route)
	return analyzed.ZoneAssignment
}

// SetCustomStrategies allows overriding URI strategies for a zone
func (a *Analyzer) SetCustomStrategies(zoneKey string, keywords []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find and update the zone's keywords
	for i, zk := range a.uriStrategies {
		if zk.ZoneKey == zoneKey {
			a.uriStrategies[i].Keywords = keywords
			return
		}
	}

	// If not found, add new
	a.uriStrategies = append(a.uriStrategies, ZoneKeyword{
		ZoneKey:  zoneKey,
		Keywords: keywords,
	})
}

// GetStrategies returns the current match strategies
func (a *Analyzer) GetStrategies() *MatchStrategies {
	return a.strategies
}

// GetURIStrategies returns URI-based zone strategies
func (a *Analyzer) GetURIStrategies() []ZoneKeyword {
	return a.uriStrategies
}

// ResetWarnings resets the unassigned warning counter
func (a *Analyzer) ResetWarnings() {
	a.mu.Lock()
	a.unassignedWarnings = 0
	a.mu.Unlock()
}

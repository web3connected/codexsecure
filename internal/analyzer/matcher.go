package analyzer

// Matcher - Pattern matching utilities
// URI and string matching for zone detection.
//
// @version 2.0.0

import (
	"path/filepath"
	"sort"
	"strings"
)

// MatchURI checks if a URI matches any keyword (case-insensitive contains)
func MatchURI(uri string, keywords []string) bool {
	return ScoreURI(uri, keywords) > 0
}

// ScoreURI calculates a weighted URI score for a set of keywords.
// Exact first-segment matches outrank later exact-segment matches, which outrank substring matches.
func ScoreURI(uri string, keywords []string) int {
	lowerURI := strings.ToLower(uri)
	trimmed := strings.Trim(lowerURI, "/")
	segments := []string{}
	if trimmed != "" {
		segments = strings.Split(trimmed, "/")
	}

	best := 0
	for _, keyword := range keywords {
		normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
		if normalizedKeyword == "" {
			continue
		}

		keywordScore := 0
		for index, segment := range segments {
			if segment != normalizedKeyword {
				continue
			}

			if index == 0 {
				keywordScore = maxInt(keywordScore, 200+len(normalizedKeyword))
			} else {
				keywordScore = maxInt(keywordScore, 100+len(normalizedKeyword))
			}
		}

		if keywordScore == 0 && strings.Contains(lowerURI, normalizedKeyword) {
			keywordScore = 20 + len(normalizedKeyword)
		}

		best = maxInt(best, keywordScore)
	}

	return best
}

// MatchURIGlob checks if a URI matches a glob pattern
// Supports * (any chars) and ? (single char)
func MatchURIGlob(uri string, pattern string) bool {
	// Normalize both for case-insensitive matching
	matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(uri))
	if err != nil {
		return false
	}
	return matched
}

// MatchController checks if a controller name matches any pattern
func MatchController(controller string, patterns map[string]int) (int, bool) {
	if controller == "" {
		return UnassignedZoneID, false
	}

	// Extract base name (e.g., "UserController" from "App\Http\Controllers\UserController")
	baseName := extractBaseName(controller)

	for pattern, zoneID := range patterns {
		if strings.Contains(strings.ToLower(baseName), strings.ToLower(pattern)) {
			return zoneID, true
		}
	}
	return UnassignedZoneID, false
}

// MatchMiddleware checks if route middleware overlaps with zone middleware
func MatchMiddleware(routeMiddleware []string, zoneMiddleware []string) bool {
	if len(routeMiddleware) == 0 || len(zoneMiddleware) == 0 {
		return false
	}

	// Build set of route middleware
	routeSet := make(map[string]bool)
	for _, m := range routeMiddleware {
		routeSet[strings.ToLower(m)] = true
	}

	// Check for any intersection
	for _, m := range zoneMiddleware {
		if routeSet[strings.ToLower(m)] {
			return true
		}
	}
	return false
}

// MatchMiddlewareAll checks if all required middleware are present
func MatchMiddlewareAll(routeMiddleware []string, requiredMiddleware []string) bool {
	if len(requiredMiddleware) == 0 {
		return true
	}

	routeSet := make(map[string]bool)
	for _, m := range routeMiddleware {
		routeSet[strings.ToLower(m)] = true
	}

	for _, m := range requiredMiddleware {
		if !routeSet[strings.ToLower(m)] {
			return false
		}
	}
	return true
}

// extractBaseName extracts the class name from a fully qualified name
// e.g., "App\Http\Controllers\UserController" -> "UserController"
// e.g., "App\Http\Controllers\UserController@index" -> "UserController"
func extractBaseName(fqn string) string {
	// Remove method suffix if present
	if idx := strings.Index(fqn, "@"); idx != -1 {
		fqn = fqn[:idx]
	}

	// Handle namespace separators (both \ and /)
	if idx := strings.LastIndex(fqn, "\\"); idx != -1 {
		return fqn[idx+1:]
	}
	if idx := strings.LastIndex(fqn, "/"); idx != -1 {
		return fqn[idx+1:]
	}

	return fqn
}

// ParseAction parses a controller@method string into ActionInfo
func ParseAction(actionName string) ActionInfo {
	if actionName == "" {
		return ActionInfo{Type: "closure"}
	}

	// Check for controller@method format
	if idx := strings.Index(actionName, "@"); idx != -1 {
		controller := actionName[:idx]
		method := actionName[idx+1:]
		return ActionInfo{
			Controller: controller,
			Method:     method,
			Type:       "controller",
		}
	}

	// Check if it looks like a closure indicator
	if strings.Contains(actionName, "Closure") || actionName == "Closure" {
		return ActionInfo{
			Type: "closure",
			Raw:  actionName,
		}
	}

	// Assume it's a single-action controller
	return ActionInfo{
		Controller: actionName,
		Type:       "invokable",
		Raw:        actionName,
	}
}

// FindKeywordZone finds the zone ID that matches a keyword in the URI
func FindKeywordZone(uri string, strategies []ZoneKeyword) (int, string, bool) {
	type scoredZone struct {
		zoneID  int
		zoneKey string
		score   int
	}

	matches := make([]scoredZone, 0, len(strategies))
	for _, zk := range strategies {
		score := ScoreURI(uri, zk.Keywords)
		if score > 0 {
			matches = append(matches, scoredZone{zoneID: zk.ZoneID, zoneKey: zk.ZoneKey, score: score})
		}
	}

	if len(matches) == 0 {
		return UnassignedZoneID, "", false
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	return matches[0].zoneID, matches[0].zoneKey, true
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

// ScoreRoute calculates a match score for a route against a zone
type MatchResult struct {
	ZoneID   int
	ZoneKey  string
	Score    int
	URIMatch bool
	CtrlMatch bool
	MWMatch  bool
}

// CalculateMatchScore determines how well a route matches a zone
func CalculateMatchScore(
	uri string,
	controller string,
	middleware []string,
	zoneKeywords []string,
	controllerPatterns map[string]int,
	zoneMiddleware []string,
	zoneID int,
	zoneKey string,
) MatchResult {
	result := MatchResult{
		ZoneID:  zoneID,
		ZoneKey: zoneKey,
	}

	// URI matching
	if MatchURI(uri, zoneKeywords) {
		result.Score++
		result.URIMatch = true
	}

	// Controller matching
	if matchedZone, ok := MatchController(controller, controllerPatterns); ok && matchedZone == zoneID {
		result.Score++
		result.CtrlMatch = true
	}

	// Middleware matching
	if MatchMiddleware(middleware, zoneMiddleware) {
		result.Score++
		result.MWMatch = true
	}

	return result
}

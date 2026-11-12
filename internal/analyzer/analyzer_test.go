package analyzer

import (
	"testing"

	"github.com/web3connected/codexsecure-api/internal/policy"
)

func TestAnalyzeRouteUsesExplicitZoneModulePrefix(t *testing.T) {
	registry := policy.NewRegistry()
	analyzer := NewAnalyzer(registry)

	result := analyzer.AnalyzeRoute(RouteInfo{
		URI:     "/zones/z2/login",
		Methods: []string{"GET"},
	})

	if result.ZoneID != 2 {
		t.Fatalf("expected zone id 2, got %d", result.ZoneID)
	}
	if result.ZoneKey != "identity" {
		t.Fatalf("expected identity zone key, got %q", result.ZoneKey)
	}
	if result.Source != "route_module" {
		t.Fatalf("expected route_module source, got %q", result.Source)
	}
	if result.Confidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %v", result.Confidence)
	}
}
package analyzer

import (
	"testing"

	"github.com/web3connected/codexsecure-api/internal/policy"
)

// Analyzer zone keys must be the policy registry's keys for the same zone IDs,
// so policy lookup, route analysis and Stage 6 alignment resolve identically.
func TestAnalyzerZoneKeysMatchPolicyRegistry(t *testing.T) {
	registry := policy.NewRegistry()

	for _, vocabulary := range DefaultVocabularies() {
		zone, ok := registry.GetZone(vocabulary.ZoneID)
		if !ok || zone.Key != vocabulary.ZoneKey {
			t.Errorf("vocabulary Z%d key %q, policy key %q", vocabulary.ZoneID, vocabulary.ZoneKey, zone.Key)
		}
	}

	for _, strategy := range DefaultURIStrategies() {
		zone, ok := registry.GetZone(strategy.ZoneID)
		if !ok || zone.Key != strategy.ZoneKey {
			t.Errorf("URI strategy Z%d key %q, policy key %q", strategy.ZoneID, strategy.ZoneKey, zone.Key)
		}
	}

	for pattern, key := range DefaultControllerStrategies() {
		if _, ok := registry.GetZoneByKey(key); !ok {
			t.Errorf("controller pattern %q uses unknown zone key %q", pattern, key)
		}
	}

	for key := range DefaultMiddlewareStrategies() {
		if _, ok := registry.GetZoneByKey(key); !ok {
			t.Errorf("middleware strategy uses unknown zone key %q", key)
		}
	}
}

// Routes classify into the approved traffic lanes, and the former classifiers
// (Z0 spam words, Z8 resource sync, Z10 behavior, Z11 agent mesh) no longer apply.
func TestAnalyzeRouteUsesApprovedTrafficLanes(t *testing.T) {
	a := NewAnalyzer(policy.NewRegistry())

	cases := []struct {
		uri    string
		zoneID int
		key    string
	}{
		{"/admin/settings", 12, "administration"},
		{"/payments/checkout", 8, "payments"},
		{"/wallet/connect", 8, "payments"},
		{"/streams/live", 10, "streaming"},
		{"/video/channel", 10, "streaming"},
		{"/replication/peers", 11, "synchronization"},
		{"/jobs/queue", 5, "processing"},
		{"/audit/events", 6, "logging"},
		{"/messages/inbox", 7, "communications"},
	}
	for _, tc := range cases {
		got := a.AnalyzeRoute(RouteInfo{URI: tc.uri, Methods: []string{"GET"}})
		if got.ZoneID != tc.zoneID || got.ZoneKey != tc.key {
			t.Errorf("%s = Z%d %q, want Z%d %q", tc.uri, got.ZoneID, got.ZoneKey, tc.zoneID, tc.key)
		}
	}

	notInto := map[string]int{
		"/spam/reports":               0, // the Z0 Boundary is not a spam destination
		"/fraud/review":               0,
		"/api/webhooks/external/sync": 8, // external sync is not payments traffic
		"/system/integrations/sync":   8,
		"/heatmaps":                   10, // behavior tracking is not streaming
		"/engagement/clickmap":        10,
		"/agents":                     11, // agents are not synchronization traffic
		"/websocket":                  11,
	}
	for uri, zoneID := range notInto {
		if got := a.AnalyzeRoute(RouteInfo{URI: uri, Methods: []string{"GET"}}); got.ZoneID == zoneID {
			t.Errorf("%s was assigned to Z%d by an obsolete classifier (%s)", uri, zoneID, got.Source)
		}
	}
}

func TestDetectRouteZoneUsesApprovedKeys(t *testing.T) {
	a := NewAnalyzer(policy.NewRegistry())
	for uri, key := range map[string]string{
		"/admin":    "administration",
		"/stream":   "streaming",
		"/sync":     "synchronization",
		"/payments": "payments",
	} {
		if got := a.DetectRouteZone(uri, ""); got != key {
			t.Errorf("DetectRouteZone(%q) = %q, want %q", uri, got, key)
		}
	}
	if got := a.DetectRouteZone("/x", "StreamController"); got != "streaming" {
		t.Errorf("StreamController = %q, want streaming", got)
	}
	if got := a.DetectRouteZone("/x", "SpamController"); got == "boundary" {
		t.Error("SpamController must not classify into the Boundary")
	}
}

// zones_used counts operational zones only; routes at the Z0 Boundary are assigned but not a zone.
func TestStatsCountOnlyOperationalZones(t *testing.T) {
	a := NewAnalyzer(policy.NewRegistry())
	result := a.AnalyzeRoutes([]RouteInfo{
		{URI: "/boundary", Methods: []string{"GET"}},
		{URI: "/admin/settings", Methods: []string{"GET"}},
	})
	if result.Routes[0].ZoneID != 0 || result.Routes[0].ZoneKey != "boundary" {
		t.Fatalf("/boundary = Z%d %q", result.Routes[0].ZoneID, result.Routes[0].ZoneKey)
	}
	if result.Stats.AssignedRoutes != 2 || result.Stats.ZonesUsed != 1 {
		t.Errorf("assigned = %d, zones_used = %d, want 2 and 1", result.Stats.AssignedRoutes, result.Stats.ZonesUsed)
	}
}

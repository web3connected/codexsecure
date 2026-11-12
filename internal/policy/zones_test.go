package policy

import "testing"

// The approved identities: the Z0 Boundary plus the twelve operational traffic classes.
func TestDefaultZonesHaveApprovedIdentities(t *testing.T) {
	want := map[int][2]string{
		0:  {"boundary", "Boundary"},
		1:  {"public", "Public"},
		2:  {"identity", "Identity"},
		3:  {"operations", "Operations"},
		4:  {"protected", "Protected Data"},
		5:  {"processing", "Processing"},
		6:  {"logging", "Logging"},
		7:  {"communications", "Communications"},
		8:  {"payments", "Payments"},
		9:  {"analytics", "Analytics"},
		10: {"streaming", "Streaming"},
		11: {"synchronization", "Synchronization"},
		12: {"administration", "Administration"},
	}

	zones := DefaultZones()
	if len(zones) != len(want) {
		t.Fatalf("expected Z0–Z12, got %d zones", len(zones))
	}
	for id, w := range want {
		zone, ok := zones[id]
		if !ok || zone.ID != id || zone.Key != w[0] || zone.Name != w[1] {
			t.Errorf("Z%d = %d/%q/%q, want %q/%q", id, zone.ID, zone.Key, zone.Name, w[0], w[1])
		}
	}
	if zones[0].Role == "" || !zones[0].Rules.ReadOnly {
		t.Error("Z0 must remain the read-only Boundary")
	}
	if err := MustValidate(DefaultPolicySnapshot()); err != nil {
		t.Fatal(err)
	}
}

func TestGetZoneByKeyResolvesAliasesToTheSameZone(t *testing.T) {
	registry := NewRegistry()
	for key, id := range map[string]int{
		"administration": 12, "admin": 12, " Admin ": 12,
		"boundary": 0, "lobby": 0, "negative": 0,
		"operations": 3, "ops": 3,
		"processing": 5, "system": 5,
		"logging": 6, "eventlogs": 6,
		"communications": 7, "comms": 7,
		"payments": 8, "streaming": 10, "synchronization": 11,
	} {
		if zone, ok := registry.GetZoneByKey(key); !ok || zone.ID != id {
			t.Errorf("GetZoneByKey(%q) = Z%d (%v), want Z%d", key, zone.ID, ok, id)
		}
	}

	// Former names of zones whose traffic responsibility changed resolve to nothing.
	for _, key := range []string{"resourcesync", "behavior", "aimesh", ""} {
		if zone, ok := registry.GetZoneByKey(key); ok {
			t.Errorf("GetZoneByKey(%q) = Z%d, want no zone", key, zone.ID)
		}
	}
}

func TestValidationRejectsAnAliasShadowingAnotherZone(t *testing.T) {
	snapshot := DefaultPolicySnapshot()
	zone := snapshot.Zones[3]
	zone.Aliases = append(zone.Aliases, "administration")
	snapshot.Zones[3] = zone

	if err := MustValidate(snapshot); err == nil {
		t.Fatal("an alias equal to another zone's key must be rejected")
	}
}

// Exactly twelve zones; Z0 is a technical identifier for the Boundary, not a thirteenth zone.
func TestTwelveOperationalZonesAndTheBoundary(t *testing.T) {
	operational := 0
	for id := range DefaultZones() {
		if IsOperationalZone(id) {
			operational++
		}
	}
	if operational != 12 || OperationalZoneCount != 12 {
		t.Errorf("operational zones = %d (OperationalZoneCount %d), want 12", operational, OperationalZoneCount)
	}
	if IsOperationalZone(BoundaryZoneID) || IsOperationalZone(13) {
		t.Error("Z0 and Z13 are not operational zones")
	}
	if len(DefaultZones()) != OperationalZoneCount+1 {
		t.Errorf("registry identifiers = %d, want the Boundary plus 12 zones", len(DefaultZones()))
	}
}

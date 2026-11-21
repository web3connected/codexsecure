package communications

import "testing"

// Recursive zones: /admin/login is in the administration area (Z12), and its sign-in form is the
// identity door into it (Z12:Z2). Only the identity door, and only into an area that requires
// authentication, is accepted beside an exact zone match.
func TestEvaluateAcceptsTheIdentityDoorIntoAnAuthenticatedArea(t *testing.T) {
	base := "http://localhost:8000"
	evaluate := func(path, zone string) Evaluation {
		doc := &Document{SchemaVersion: 1, Destination: base + path, DestinationMatch: "exact", Methods: []string{"POST"}, Type: "form", Zone: zone, Enabled: true}
		return Evaluate("approved", "valid", 1, 1, doc, Observation{Destination: base + path, Method: "POST", Type: "form", Zone: zone}, base)
	}
	for _, c := range []struct {
		path, zone, effective string
	}{
		{"/admin/login", "Z2", "approved"},          // identity door into Z12 (auth required): Z12:Z2
		{"/admin/forgot-secret", "Z2", "approved"},  // the recovery door too
		{"/admin/login", "Z12", "approved"},         // an exact area match still aligns
		{"/admin/login", "Z4", "unauthorized"},      // another zone is not a door into Z12
		{"/pages/contact-us", "Z2", "unauthorized"}, // Z1 requires no sign-in: no identity door
		{"/login", "Z2", "approved"},                // plain Z2 is unchanged
		{"/login", "Z1", "unauthorized"},
	} {
		got := evaluate(c.path, c.zone)
		if got.Effective != c.effective {
			t.Errorf("%s approved in %s: got %s %v, want %s", c.path, c.zone, got.Effective, got.Mismatches, c.effective)
		}
	}
	if zoneAligned("Z2", "") || zoneAligned("Z2", "Z0") || zoneAligned("Z2", "Z13") || zoneAligned("Z2", "Z012") {
		t.Fatal("the identity door needs a defined operational area")
	}
}

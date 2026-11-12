package handlers

import (
	"testing"

	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// Stage 4 intent defaults to the zone key, so Z12 traffic carries "administration"
// as well as the "admin" label; both must keep fingerprint enforcement.
func TestShouldEnforceFingerprintForAdministrationIntent(t *testing.T) {
	zone := policy.DefaultZones()[4] // business tier, so only the intent decides
	for _, intent := range []string{"admin", "administration"} {
		if !shouldEnforceFingerprint(zone, Stage4IntentResult{Intent: intent}, fingerprint.FilterResult{}) {
			t.Errorf("intent %q must enforce fingerprints", intent)
		}
	}
	if shouldEnforceFingerprint(zone, Stage4IntentResult{Intent: "public"}, fingerprint.FilterResult{}) {
		t.Error("public intent on a business zone must not enforce fingerprints")
	}
}

func TestInferZoneCandidateFromPathUsesPolicyKeys(t *testing.T) {
	registry := policy.NewRegistry()
	for path, id := range map[string]int{
		"/admin/x":   12,
		"/auth/x":    2,
		"/wallet/x":  8,
		"/system/x":  5,
		"/analytics": 9,
		"/sync/x":    11,
		"/other":     1,
	} {
		zone, ok := registry.GetZoneByKey(inferZoneCandidateFromPath(path))
		if !ok || zone.ID != id {
			t.Errorf("%s -> %q, want Z%d", path, inferZoneCandidateFromPath(path), id)
		}
	}
}

package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// canonicalRegistryPath is the shared expected registry. The same file is used by
// scripts/verify_zone_alignment.sh against the running API.
var canonicalRegistryPath = filepath.Join("..", "..", "testdata", "zones", "canonical_registry.json")

func loadCanonicalRegistry(t *testing.T) map[int]string {
	t.Helper()

	content, err := os.ReadFile(canonicalRegistryPath)
	if err != nil {
		t.Fatalf("read canonical registry: %v", err)
	}
	var fixture struct {
		Zones map[string]string `json:"zones"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("decode canonical registry: %v", err)
	}
	expected, err := ParseExpectedRegistry(fixture.Zones)
	if err != nil {
		t.Fatalf("parse canonical registry: %v", err)
	}
	return expected
}

func logResult(t *testing.T, result AlignmentResult) {
	t.Helper()
	encoded, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("alignment result:\n%s", encoded)
}

// TestDefaultRegistryAlignsWithCanonicalRegistry compares the real policy
// registry (DefaultZones via NewRegistry) against the canonical registry.
func TestDefaultRegistryAlignsWithCanonicalRegistry(t *testing.T) {
	expected := loadCanonicalRegistry(t)
	if len(expected) != 13 {
		t.Fatalf("canonical registry must list Z0–Z12, got %d zones", len(expected))
	}

	result := NewRegistry().Alignment(expected)
	logResult(t, result)

	if !result.Aligned || result.Status != "aligned" {
		t.Errorf("registry is not aligned: %d difference(s)", len(result.Differences))
	}
	for _, difference := range result.Differences {
		expectedKey, actualKey := "<none>", "<none>"
		if difference.Expected != nil {
			expectedKey = *difference.Expected
		}
		if difference.Actual != nil {
			actualKey = *difference.Actual
		}
		t.Errorf("%s %s: expected %q, actual %q", difference.ZoneCode, difference.Type, expectedKey, actualKey)
	}
	if len(result.Matched) != 13 {
		t.Errorf("expected all 13 zones to match, matched %d: %v", len(result.Matched), result.Matched)
	}
	if result.Signature != result.ExpectedSignature {
		t.Errorf("signature %s does not equal expected signature %s", result.Signature, result.ExpectedSignature)
	}
}

// TestDeliberateMismatchFailsAlignment changes one expected key (copy only;
// the canonical file is untouched) and checks the real registry no longer aligns.
func TestDeliberateMismatchFailsAlignment(t *testing.T) {
	expected := loadCanonicalRegistry(t)
	expected[5] = "systems"

	result := NewRegistry().Alignment(expected)
	logResult(t, result)

	if result.Aligned || result.Status != "mismatch" {
		t.Fatal("alignment must fail when an expected key is changed")
	}
	var found bool
	for _, difference := range result.Differences {
		if difference.ZoneCode == "Z5" {
			found = difference.Type == DifferenceMismatched && *difference.Expected == "systems" && *difference.Actual == "processing"
		}
	}
	if !found {
		t.Errorf("Z5 mismatch not reported: %+v", result.Differences)
	}
	for _, code := range result.Matched {
		if code == "Z5" {
			t.Error("Z5 must not be reported as matched")
		}
	}
	if result.Signature == result.ExpectedSignature {
		t.Error("signatures must differ when registries differ")
	}
}

func TestCompareRegistriesReportsMissingExtraAndMismatched(t *testing.T) {
	actual := map[int]string{0: "boundary", 1: "public", 2: "identity", 13: "experimental"}
	expected := map[int]string{0: "boundary", 1: "Public ", 2: "id", 3: "operations"}

	result := CompareRegistries(actual, expected)

	if result.Aligned {
		t.Fatal("expected mismatch")
	}
	if got := result.Matched; len(got) != 2 || got[0] != "Z0" || got[1] != "Z1" {
		t.Errorf("matched = %v, want [Z0 Z1] (keys are trimmed and lowercased)", got)
	}
	want := []struct {
		code, kind string
	}{
		{"Z2", DifferenceMismatched},
		{"Z3", DifferenceMissing},
		{"Z13", DifferenceExtra},
	}
	if len(result.Differences) != len(want) {
		t.Fatalf("differences = %+v", result.Differences)
	}
	for i, w := range want {
		d := result.Differences[i]
		if d.ZoneCode != w.code || d.Type != w.kind {
			t.Errorf("difference %d = %s/%s, want %s/%s", i, d.ZoneCode, d.Type, w.code, w.kind)
		}
	}
	if result.Differences[1].Actual != nil || result.Differences[2].Expected != nil {
		t.Error("missing zones have no actual key and extra zones have no expected key")
	}
	if result.ZonesChecked != 5 {
		t.Errorf("zones_checked = %d, want 5", result.ZonesChecked)
	}
}

func TestRegistrySignatureCanonicalForm(t *testing.T) {
	// sha256("Z0:boundary\nZ2:identity\nZ10:streaming"), computed independently:
	// printf 'Z0:boundary\nZ2:identity\nZ10:streaming' | sha256sum
	const want = "0a3eabb11e6d25d96b61014749090f858cbfbd63f784ff8cf0971e7654115868"

	got := RegistrySignature(map[int]string{10: "Streaming", 0: " boundary", 2: "identity"})
	if got != want {
		t.Errorf("signature = %s, want %s", got, want)
	}
}

func TestParseExpectedRegistryRejectsInvalidInput(t *testing.T) {
	cases := map[string]map[string]string{
		"empty":          {},
		"bad identifier": {"Zx": "public"},
		"negative id":    {"-1": "public"},
		"signed id":      {"+1": "public"},
		"empty key":      {"Z1": "  "},
		"duplicate zone": {"Z3": "operations", "3": "operations"},
	}
	for name, zones := range cases {
		if _, err := ParseExpectedRegistry(zones); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	parsed, err := ParseExpectedRegistry(map[string]string{"z3": "OPERATIONS", "12": "administration"})
	if err != nil || parsed[3] != "operations" || parsed[12] != "administration" {
		t.Errorf("parsed = %v, err = %v", parsed, err)
	}
}

// The canonical registry signature, computed independently:
// printf 'Z0:boundary\nZ1:public\n...\nZ12:administration' | sha256sum
const canonicalRegistrySignature = "5eca793c1a786b864899595b2ae784a433e795efaa9a70c24cdcfba8ecbaaff5"

func TestCanonicalRegistrySignature(t *testing.T) {
	if got := RegistrySignature(NewRegistry().KeyRegistry()); got != canonicalRegistrySignature {
		t.Errorf("signature = %s, want %s", got, canonicalRegistrySignature)
	}
}

// An alias names the same zone: "admin" for Z12 aligns without a structural mismatch.
func TestAliasesAlignAsTheSameZone(t *testing.T) {
	expected := loadCanonicalRegistry(t)
	expected[7] = "comms"
	expected[12] = "admin"

	result := NewRegistry().Alignment(expected)
	logResult(t, result)

	if !result.Aligned || result.Status != "aligned" || len(result.Differences) != 0 || len(result.Matched) != 13 {
		t.Fatalf("aliases must align: %+v", result.Differences)
	}
	want := []ZoneEquivalent{
		{ZoneID: 7, ZoneCode: "Z7", Submitted: "comms", Canonical: "communications"},
		{ZoneID: 12, ZoneCode: "Z12", Submitted: "admin", Canonical: "administration"},
	}
	if len(result.Equivalents) != len(want) {
		t.Fatalf("equivalents = %+v, want %+v", result.Equivalents, want)
	}
	for i := range want {
		if result.Equivalents[i] != want[i] {
			t.Errorf("equivalents[%d] = %+v, want %+v", i, result.Equivalents[i], want[i])
		}
	}
	// The signatures still describe each registry exactly as named.
	if result.Signature != canonicalRegistrySignature || result.ExpectedSignature == result.Signature {
		t.Errorf("signature = %s, expected_signature = %s", result.Signature, result.ExpectedSignature)
	}
}

// An alias only matches its own zone; another zone's key or alias is a zone-number conflict.
func TestAliasAtAnotherZoneIsMismatched(t *testing.T) {
	for id, key := range map[int]string{3: "admin", 5: "administration", 6: "comms"} {
		expected := loadCanonicalRegistry(t)
		expected[id] = key

		result := NewRegistry().Alignment(expected)
		if result.Aligned || len(result.Differences) != 1 || result.Differences[0].ZoneID != id ||
			result.Differences[0].Type != DifferenceMismatched {
			t.Errorf("Z%d=%q: differences = %+v", id, key, result.Differences)
		}
		if len(result.Equivalents) != 0 {
			t.Errorf("Z%d=%q: equivalents = %+v", id, key, result.Equivalents)
		}
	}
}

// Z8, Z10 and Z11 carry different traffic than their former names, so a
// registry still using those names is not aligned; renamed zones are equivalent.
func TestFormerRegistryReportsOnlyChangedResponsibilities(t *testing.T) {
	former := map[int]string{
		0: "negative", 1: "public", 2: "identity", 3: "ops", 4: "protected", 5: "system", 6: "eventlogs",
		7: "comms", 8: "resourcesync", 9: "analytics", 10: "behavior", 11: "aimesh", 12: "admin",
	}

	result := NewRegistry().Alignment(former)
	logResult(t, result)

	if result.Aligned {
		t.Fatal("a registry with former Z8/Z10/Z11 names must not align")
	}
	want := map[string][2]string{
		"Z8":  {"resourcesync", "payments"},
		"Z10": {"behavior", "streaming"},
		"Z11": {"aimesh", "synchronization"},
	}
	if len(result.Differences) != len(want) {
		t.Fatalf("differences = %+v", result.Differences)
	}
	for _, d := range result.Differences {
		w, ok := want[d.ZoneCode]
		if !ok || d.Type != DifferenceMismatched || *d.Expected != w[0] || *d.Actual != w[1] {
			t.Errorf("unexpected difference %s %s %v %v", d.ZoneCode, d.Type, *d.Expected, *d.Actual)
		}
	}
	if len(result.Matched) != 10 || len(result.Equivalents) != 6 {
		t.Errorf("matched = %v, equivalents = %+v", result.Matched, result.Equivalents)
	}
}

func TestMissingAndUnexpectedZonesAreReported(t *testing.T) {
	expected := loadCanonicalRegistry(t)
	delete(expected, 11)
	expected[13] = "experimental"

	result := NewRegistry().Alignment(expected)
	if result.Aligned || len(result.Differences) != 2 {
		t.Fatalf("differences = %+v", result.Differences)
	}
	if d := result.Differences[0]; d.ZoneCode != "Z11" || d.Type != DifferenceExtra {
		t.Errorf("Z11 = %+v, want extra (in the API, not submitted)", d)
	}
	if d := result.Differences[1]; d.ZoneCode != "Z13" || d.Type != DifferenceMissing {
		t.Errorf("Z13 = %+v, want missing (submitted, not in the API)", d)
	}
}

// web3codexRegistryPath is the registry Web3Codex submits today (Z0 still "lobby"),
// generated from its codexsecure.zone_registry config.
var web3codexRegistryPath = filepath.Join("..", "..", "testdata", "zones", "web3codex_registry.json")

// Web3Codex's current payload differs only in the Z0 name, which is an alias of
// "boundary". Its own signature (sha256 of the "Z0:lobby" form) stays its own.
func TestWeb3CodexRegistryAlignsThroughTheBoundaryAlias(t *testing.T) {
	saved := canonicalRegistryPath
	canonicalRegistryPath = web3codexRegistryPath
	expected := loadCanonicalRegistry(t)
	canonicalRegistryPath = saved

	result := NewRegistry().Alignment(expected)
	logResult(t, result)

	if !result.Aligned || len(result.Matched) != 13 || len(result.Differences) != 0 {
		t.Fatalf("Web3Codex registry must align: %+v", result.Differences)
	}
	want := ZoneEquivalent{ZoneID: 0, ZoneCode: "Z0", Submitted: "lobby", Canonical: "boundary"}
	if len(result.Equivalents) != 1 || result.Equivalents[0] != want {
		t.Errorf("equivalents = %+v, want [%+v]", result.Equivalents, want)
	}
	// printf 'Z0:lobby\nZ1:public\n...\nZ12:administration' | sha256sum
	const web3codexSignature = "dd2d6c35d024281381a814f44f3297d788553d310025a99f809b50b523b6960b"
	if result.ExpectedSignature != web3codexSignature || result.Signature != canonicalRegistrySignature {
		t.Errorf("signature = %s, expected_signature = %s", result.Signature, result.ExpectedSignature)
	}
}

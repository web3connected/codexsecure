package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// canonicalDefinitionsPath is the agreed definition set: the Web3Codex SDK's
// zone_registry and zone_definitions with the Z0 Boundary definition (see
// docs/ZONE_ALIGNMENT.md). web3codex_definitions.json is the SDK config as it is today.
var canonicalDefinitionsPath = filepath.Join("..", "..", "testdata", "zones", "canonical_definitions.json")

// Signatures computed independently in Python (json.dumps, compact, ensure_ascii=False).
const (
	canonicalDefinitionsSignature = "1316311da4960d6b633e81b35e47deb304e922cc1a6913916a1b8256b450589c"
	smallDefinitionSignature      = "93f6d3e85a37d93595c8168db89c0ba26afc30d9c8a072e1e7ce3faa9e261aa7"
)

func loadCanonicalDefinitions(t *testing.T) map[int]DefinitionContract {
	t.Helper()

	content, err := os.ReadFile(canonicalDefinitionsPath)
	if err != nil {
		t.Fatalf("read canonical definitions: %v", err)
	}
	var fixture struct {
		Zones map[string]ExpectedDefinition `json:"zones"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("decode canonical definitions: %v", err)
	}
	expected, err := ParseExpectedDefinitions(fixture.Zones)
	if err != nil {
		t.Fatalf("parse canonical definitions: %v", err)
	}
	return expected
}

func logDefinitionResult(t *testing.T, result DefinitionAlignmentResult) {
	t.Helper()
	encoded, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("definition alignment result:\n%s", encoded)
}

func differenceFor(result DefinitionAlignmentResult, code string) *DefinitionDifference {
	for i := range result.Differences {
		if result.Differences[i].ZoneCode == code {
			return &result.Differences[i]
		}
	}
	return nil
}

func hasField(difference *DefinitionDifference, field string) bool {
	if difference == nil {
		return false
	}
	for _, f := range difference.Fields {
		if f.Field == field {
			return true
		}
	}
	return false
}

func TestDefaultDefinitionsAlignWithCanonicalDefinitions(t *testing.T) {
	expected := loadCanonicalDefinitions(t)
	if len(expected) != 13 {
		t.Fatalf("canonical definitions must list Z0–Z12, got %d", len(expected))
	}

	result := NewRegistry().DefinitionAlignment(expected)
	logDefinitionResult(t, result)

	if !result.Aligned || len(result.Differences) != 0 || len(result.Matched) != 13 {
		for _, d := range result.Differences {
			for _, f := range d.Fields {
				t.Errorf("%s %s: expected %v, actual %v", d.ZoneCode, f.Field, f.Expected, f.Actual)
			}
			if len(d.Fields) == 0 {
				t.Errorf("%s %s", d.ZoneCode, d.Type)
			}
		}
		t.Fatalf("definitions not aligned: %d difference(s)", len(result.Differences))
	}
	if result.Signature != result.ExpectedSignature {
		t.Errorf("signature %s != expected %s", result.Signature, result.ExpectedSignature)
	}
}

func TestDefinitionSignatureMatchesIndependentImplementation(t *testing.T) {
	if got := DefinitionSignature(loadCanonicalDefinitions(t)); got != canonicalDefinitionsSignature {
		t.Errorf("fixture signature = %s, want %s", got, canonicalDefinitionsSignature)
	}

	// "&", "/", "<", ">" must not be escaped; methods are sorted; absent role/executive are ""/false.
	small := map[int]DefinitionContract{7: {
		Key: " Comms ", Name: "Comms & Alerts", Description: "A/B <x>",
		MethodsAllowed: []string{"post", "GET", "POST"},
	}}
	if got := DefinitionSignature(small); got != smallDefinitionSignature {
		t.Errorf("small signature = %s, want %s\ncanonical: %s", got, smallDefinitionSignature, CanonicalDefinitionJSON(small[7]))
	}
}

func TestDefinitionSignatureIsDeterministic(t *testing.T) {
	first := DefinitionSignature(NewRegistry().DefinitionContracts())
	for i := 0; i < 50; i++ {
		if got := DefinitionSignature(NewRegistry().DefinitionContracts()); got != first {
			t.Fatalf("signature changed between runs: %s vs %s", first, got)
		}
	}
}

func TestDefinitionAlignmentDetectsMismatches(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[int]DefinitionContract)
		code   string
		field  string
	}{
		{"zone key", func(d map[int]DefinitionContract) { c := d[8]; c.Key = "resourcesync"; d[8] = c }, "Z8", "key"},
		{"functional name", func(d map[int]DefinitionContract) { c := d[11]; c.Name = "Realtime"; d[11] = c }, "Z11", "name"},
		{"functional description", func(d map[int]DefinitionContract) { c := d[3]; c.Description = "Live runtime execution."; d[3] = c }, "Z3", "description"},
		{"HTTP methods", func(d map[int]DefinitionContract) {
			c := d[9]
			c.MethodsAllowed = []string{"GET", "POST", "OPTIONS"}
			d[9] = c
		}, "Z9", "methods_allowed"},
		{"read_only", func(d map[int]DefinitionContract) { c := d[9]; c.ReadOnly = false; d[9] = c }, "Z9", "read_only"},
		{"executive", func(d map[int]DefinitionContract) { c := d[12]; c.Executive = false; d[12] = c }, "Z12", "executive"},
		{"subzones", func(d map[int]DefinitionContract) {
			c := d[2]
			c.Subzones = map[string]any{"Z2:Z01": map[string]any{"name": "Login flows"}}
			d[2] = c
		}, "Z2", "subzones"},
		{"Z0 role", func(d map[int]DefinitionContract) { c := d[0]; c.Role = "Intake orchestration path."; d[0] = c }, "Z0", "role"},
		{"Z0 key", func(d map[int]DefinitionContract) { c := d[0]; c.Key = "intake"; d[0] = c }, "Z0", "key"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expected := loadCanonicalDefinitions(t)
			tc.mutate(expected)

			result := CompareDefinitions(loadCanonicalDefinitions(t), expected)

			if result.Aligned || result.Status != "mismatch" {
				t.Fatal("expected mismatch")
			}
			if len(result.Differences) != 1 {
				t.Fatalf("expected exactly one zone difference, got %+v", result.Differences)
			}
			difference := differenceFor(result, tc.code)
			if difference == nil || difference.Type != DifferenceMismatched || !hasField(difference, tc.field) || len(difference.Fields) != 1 {
				t.Fatalf("expected %s mismatched on %s only, got %+v", tc.code, tc.field, result.Differences)
			}
			if len(result.Matched) != 12 {
				t.Errorf("expected 12 matched zones, got %d", len(result.Matched))
			}
			if result.Signature == result.ExpectedSignature {
				t.Error("signatures must differ when definitions differ")
			}
		})
	}
}

func TestDefinitionAlignmentReportsMissingAndExtraZones(t *testing.T) {
	expected := loadCanonicalDefinitions(t)
	delete(expected, 12)
	expected[13] = DefinitionContract{Key: "experimental", Name: "Experimental", MethodsAllowed: []string{"GET"}}

	result := CompareDefinitions(loadCanonicalDefinitions(t), expected)

	if result.Aligned || result.ZonesChecked != 14 {
		t.Fatalf("aligned=%v zones_checked=%d", result.Aligned, result.ZonesChecked)
	}
	if d := differenceFor(result, "Z12"); d == nil || d.Type != DifferenceExtra {
		t.Errorf("Z12 should be extra (in API, not submitted): %+v", d)
	}
	if d := differenceFor(result, "Z13"); d == nil || d.Type != DifferenceMissing {
		t.Errorf("Z13 should be missing (submitted, not in API): %+v", d)
	}
}

func TestDefinitionAlignmentIgnoresPresentationDifferences(t *testing.T) {
	expected := loadCanonicalDefinitions(t)
	c := expected[7]
	c.MethodsAllowed = []string{"options", "POST", "GET", "GET"} // order, case, duplicates
	c.Name = "  " + c.Name + "  "
	c.Subzones = map[string]any{} // empty object == empty list
	expected[7] = c

	if result := CompareDefinitions(loadCanonicalDefinitions(t), expected); !result.Aligned {
		t.Fatalf("normalization should align: %+v", result.Differences)
	}
}

func TestParseExpectedDefinitionsRejectsInvalidInput(t *testing.T) {
	readOnly := false
	valid := func() ExpectedDefinition {
		d := ExpectedDefinition{Key: "public", Name: "Public"}
		d.Rules.ReadOnly = &readOnly
		d.Rules.MethodsAllowed = []string{"GET"}
		return d
	}

	cases := map[string]map[string]ExpectedDefinition{
		"empty":             {},
		"bad identifier":    {"Zx": valid()},
		"duplicate zone":    {"Z1": valid(), "1": valid()},
		"empty key":         {"Z1": func() ExpectedDefinition { d := valid(); d.Key = " "; return d }()},
		"empty name":        {"Z1": func() ExpectedDefinition { d := valid(); d.Name = ""; return d }()},
		"missing read_only": {"Z1": func() ExpectedDefinition { d := valid(); d.Rules.ReadOnly = nil; return d }()},
		"no methods":        {"Z1": func() ExpectedDefinition { d := valid(); d.Rules.MethodsAllowed = nil; return d }()},
		"invalid method":    {"Z1": func() ExpectedDefinition { d := valid(); d.Rules.MethodsAllowed = []string{"FETCH"}; return d }()},
		"invalid subzones":  {"Z1": func() ExpectedDefinition { d := valid(); d.Subzones = json.RawMessage(`{"a":`); return d }()},
	}
	for name, zones := range cases {
		if _, err := ParseExpectedDefinitions(zones); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	if _, err := ParseExpectedDefinitions(map[string]ExpectedDefinition{"z1": valid()}); err != nil {
		t.Errorf("valid definition rejected: %v", err)
	}
}

// Registry alignment (ID → key) and definition alignment are separate checks.
func TestRegistryAndDefinitionAlignmentAreIndependent(t *testing.T) {
	expected := loadCanonicalDefinitions(t)
	c := expected[5]
	c.Description = "Different description, same key."
	expected[5] = c

	keys := make(map[int]string, len(expected))
	for id, contract := range expected {
		keys[id] = contract.Key
	}
	actual := loadCanonicalDefinitions(t)
	actualKeys := make(map[int]string, len(actual))
	for id, contract := range actual {
		actualKeys[id] = contract.Key
	}

	if registry := CompareRegistries(actualKeys, keys); !registry.Aligned {
		t.Fatal("registry alignment should still pass: keys are identical")
	}
	if definitions := CompareDefinitions(actual, expected); definitions.Aligned {
		t.Fatal("definition alignment must fail on the description change")
	}
}

func TestDefinitionAlignmentAcceptsKeyAliases(t *testing.T) {
	expected := loadCanonicalDefinitions(t)
	c := expected[12]
	c.Key = "admin"
	expected[12] = c

	result := NewRegistry().DefinitionAlignment(expected)
	if !result.Aligned || len(result.Matched) != 13 {
		t.Fatalf("alias key must align: %+v", result.Differences)
	}
	if result.ExpectedSignature != DefinitionSignature(expected) {
		t.Error("expected_signature must describe the definitions as submitted")
	}

	c.Key = "resourcesync"
	expected[12] = c
	if result := NewRegistry().DefinitionAlignment(expected); result.Aligned {
		t.Fatal("another zone's former key must not align")
	}
}

// Web3Codex's current definitions still describe Z0 as "Lobby / Gateway". The key
// is an alias, but name, description and role are real differences to report.
func TestWeb3CodexDefinitionsDifferOnlyInTheZ0Description(t *testing.T) {
	saved := canonicalDefinitionsPath
	canonicalDefinitionsPath = filepath.Join("..", "..", "testdata", "zones", "web3codex_definitions.json")
	expected := loadCanonicalDefinitions(t)
	canonicalDefinitionsPath = saved

	result := NewRegistry().DefinitionAlignment(expected)
	if result.Aligned || len(result.Matched) != 12 || len(result.Differences) != 1 {
		t.Fatalf("matched = %v, differences = %+v", result.Matched, result.Differences)
	}
	difference := differenceFor(result, "Z0")
	if difference == nil || len(difference.Fields) != 3 || hasField(difference, "key") ||
		!hasField(difference, "name") || !hasField(difference, "description") || !hasField(difference, "role") {
		t.Errorf("Z0 difference = %+v, want name, description and role only", difference)
	}
}

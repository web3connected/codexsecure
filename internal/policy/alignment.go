package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SignatureAlgorithm identifies the canonical zone registry signature.
//
// Canonical form: one "Z{id}:{key}" line per zone, ordered by numeric zone ID
// ascending, joined by "\n" with no trailing newline. Keys are trimmed and
// lowercased. The signature is the lowercase hex SHA-256 of that string.
//
// Example: "Z0:boundary\nZ1:public\n...\nZ12:administration"
const SignatureAlgorithm = "sha256:zone-registry-v1"

// Alignment difference types.
const (
	DifferenceMissing    = "missing"    // expected zone ID is absent from the API registry
	DifferenceExtra      = "extra"      // API registry zone ID is absent from the expected registry
	DifferenceMismatched = "mismatched" // zone ID present in both, keys differ
)

// RegistryEntry is a single zone ID → key pair in canonical order.
type RegistryEntry struct {
	ZoneID   int    `json:"zone_id"`
	ZoneCode string `json:"zone_code"`
	Key      string `json:"key"`
}

// ZoneEquivalent records a zone matched through an alias of the API key
// (e.g. "admin" submitted for Z12, whose key is "administration").
type ZoneEquivalent struct {
	ZoneID    int    `json:"zone_id"`
	ZoneCode  string `json:"zone_code"`
	Submitted string `json:"submitted"`
	Canonical string `json:"canonical"`
}

// ZoneDifference describes one zone that does not align.
type ZoneDifference struct {
	ZoneID   int     `json:"zone_id"`
	ZoneCode string  `json:"zone_code"`
	Type     string  `json:"type"`
	Expected *string `json:"expected"`
	Actual   *string `json:"actual"`
}

// AlignmentResult is the outcome of comparing an expected registry with the API registry.
type AlignmentResult struct {
	Aligned            bool             `json:"aligned"`
	Status             string           `json:"status"` // "aligned" | "mismatch"
	PolicyVersion      string           `json:"policy_version"`
	SignatureAlgorithm string           `json:"signature_algorithm"`
	Signature          string           `json:"signature"`          // API registry
	ExpectedSignature  string           `json:"expected_signature"` // submitted registry
	ZonesChecked       int              `json:"zones_checked"`
	Matched            []string         `json:"matched"`
	Equivalents        []ZoneEquivalent `json:"equivalents,omitempty"` // matched zones submitted under an alias; omitted when none
	Differences        []ZoneDifference `json:"differences"`
}

// NormalizeZoneKey returns the canonical form of a zone key.
func NormalizeZoneKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// ParseZoneIdentifier accepts "Z3", "z3" or "3" and returns the numeric zone ID.
// Any non-negative ID is accepted so that unknown zones surface as differences.
func ParseZoneIdentifier(identifier string) (int, error) {
	value := strings.TrimSpace(identifier)
	if len(value) > 0 && (value[0] == 'Z' || value[0] == 'z') {
		value = value[1:]
	}
	id, err := strconv.Atoi(value)
	if err != nil || id < 0 || value == "" || strings.HasPrefix(value, "+") {
		return -1, fmt.Errorf("invalid zone identifier %q", identifier)
	}
	return id, nil
}

// ParseExpectedRegistry converts a submitted {"Z0": "boundary", ...} mapping
// into zone ID → normalized key. It rejects malformed identifiers, empty keys
// and identifiers that resolve to the same zone ID (e.g. "Z3" and "3").
func ParseExpectedRegistry(zones map[string]string) (map[int]string, error) {
	if len(zones) == 0 {
		return nil, fmt.Errorf("zones must contain at least one zone")
	}

	parsed := make(map[int]string, len(zones))
	for identifier, key := range zones {
		id, err := ParseZoneIdentifier(identifier)
		if err != nil {
			return nil, err
		}
		normalized := NormalizeZoneKey(key)
		if normalized == "" {
			return nil, fmt.Errorf("zone %s must map to a non-empty key", identifier)
		}
		if _, exists := parsed[id]; exists {
			return nil, fmt.Errorf("zone Z%d is listed more than once", id)
		}
		parsed[id] = normalized
	}
	return parsed, nil
}

// CanonicalEntries returns the registry as entries ordered by zone ID.
func CanonicalEntries(registry map[int]string) []RegistryEntry {
	ids := make([]int, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	entries := make([]RegistryEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, RegistryEntry{
			ZoneID:   id,
			ZoneCode: fmt.Sprintf("Z%d", id),
			Key:      NormalizeZoneKey(registry[id]),
		})
	}
	return entries
}

// RegistrySignature computes the canonical signature (see SignatureAlgorithm).
func RegistrySignature(registry map[int]string) string {
	entries := CanonicalEntries(registry)
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.ZoneCode+":"+entry.Key)
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// CompareRegistries compares an expected registry against an actual registry.
// Aligned is true only when both contain exactly the same zone IDs with the same keys.
func CompareRegistries(actual map[int]string, expected map[int]string) AlignmentResult {
	return compareRegistries(actual, expected, nil)
}

// compareRegistries is CompareRegistries where a submitted key that is one of
// aliases[id] also matches zone id. An alias only matches at its own zone ID;
// a key belonging to another zone is still a mismatch.
func compareRegistries(actual map[int]string, expected map[int]string, aliases map[int][]string) AlignmentResult {
	ids := make(map[int]struct{}, len(actual)+len(expected))
	for id := range actual {
		ids[id] = struct{}{}
	}
	for id := range expected {
		ids[id] = struct{}{}
	}
	ordered := make([]int, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Ints(ordered)

	result := AlignmentResult{
		SignatureAlgorithm: SignatureAlgorithm,
		Signature:          RegistrySignature(actual),
		ExpectedSignature:  RegistrySignature(expected),
		ZonesChecked:       len(ordered),
		Matched:            []string{},
		Equivalents:        []ZoneEquivalent{},
		Differences:        []ZoneDifference{},
	}

	for _, id := range ordered {
		code := fmt.Sprintf("Z%d", id)
		actualKey, inActual := actual[id]
		expectedKey, inExpected := expected[id]
		actualKey = NormalizeZoneKey(actualKey)
		expectedKey = NormalizeZoneKey(expectedKey)

		switch {
		case inActual && inExpected && actualKey == expectedKey:
			result.Matched = append(result.Matched, code)
		case inActual && inExpected && containsKey(aliases[id], expectedKey):
			result.Matched = append(result.Matched, code)
			result.Equivalents = append(result.Equivalents, ZoneEquivalent{
				ZoneID: id, ZoneCode: code, Submitted: expectedKey, Canonical: actualKey,
			})
		case inActual && inExpected:
			result.Differences = append(result.Differences, ZoneDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceMismatched,
				Expected: stringPointer(expectedKey), Actual: stringPointer(actualKey),
			})
		case inExpected:
			result.Differences = append(result.Differences, ZoneDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceMissing,
				Expected: stringPointer(expectedKey),
			})
		default:
			result.Differences = append(result.Differences, ZoneDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceExtra,
				Actual: stringPointer(actualKey),
			})
		}
	}

	result.Aligned = len(result.Differences) == 0
	result.Status = "mismatch"
	if result.Aligned {
		result.Status = "aligned"
	}
	return result
}

// KeyRegistry returns the registry's zone ID → key mapping.
func (r *Registry) KeyRegistry() map[int]string {
	registry := make(map[int]string, len(r.snapshot.Zones))
	for id, zone := range r.snapshot.Zones {
		registry[id] = zone.Key
	}
	return registry
}

// Aliases returns each zone's alias keys by zone ID.
func (r *Registry) Aliases() map[int][]string {
	aliases := make(map[int][]string, len(r.snapshot.Zones))
	for id, zone := range r.snapshot.Zones {
		aliases[id] = zone.Aliases
	}
	return aliases
}

// Alignment compares an expected registry against this policy registry.
// Aliases of a zone's key count as the same zone.
func (r *Registry) Alignment(expected map[int]string) AlignmentResult {
	result := compareRegistries(r.KeyRegistry(), expected, r.Aliases())
	result.PolicyVersion = r.snapshot.Version
	return result
}

func containsKey(keys []string, key string) bool {
	for _, candidate := range keys {
		if NormalizeZoneKey(candidate) == key {
			return true
		}
	}
	return false
}

func stringPointer(value string) *string {
	return &value
}

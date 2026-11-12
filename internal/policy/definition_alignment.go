package policy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DefinitionSignatureAlgorithm identifies the canonical zone definition signature.
//
// Canonical form: one "Z{id}:{json}" line per zone, ordered by numeric zone ID
// ascending, joined by "\n" with no trailing newline. {json} is the zone's
// DefinitionContract encoded as compact JSON with fields in the order below,
// methods uppercased/deduplicated/sorted, an absent executive flag as false,
// empty subzones as [], object keys sorted, and no escaping of "/", "&", "<",
// ">" or non-ASCII characters (PHP: JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE).
// The signature is the lowercase hex SHA-256 of that string.
//
// A matching signature only shows that two definition sets are identical. It is
// not authentication and does not show that any route belongs in its zone.
const DefinitionSignatureAlgorithm = "sha256:zone-definitions-v1"

// DefinitionFields lists the compared fields in canonical order. Go-only
// enforcement fields (tier, auth_required, tiu_required) are not part of the contract.
var DefinitionFields = []string{"key", "name", "description", "role", "read_only", "methods_allowed", "executive", "subzones"}

// DefinitionContract is the part of a zone definition shared with SDKs.
// Field order is the canonical JSON order.
type DefinitionContract struct {
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Role           string   `json:"role"`
	ReadOnly       bool     `json:"read_only"`
	MethodsAllowed []string `json:"methods_allowed"`
	Executive      bool     `json:"executive"`
	Subzones       any      `json:"subzones"`
}

// ExpectedDefinition is a submitted zone definition, in the SDK configuration shape.
type ExpectedDefinition struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	Rules       struct {
		ReadOnly       *bool    `json:"read_only"`
		MethodsAllowed []string `json:"methods_allowed"`
		Executive      *bool    `json:"executive"`
	} `json:"rules"`
	Subzones json.RawMessage `json:"subzones"`
}

// DefinitionFieldDifference is one field that differs within a zone.
type DefinitionFieldDifference struct {
	Field    string `json:"field"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
}

// DefinitionDifference describes one zone whose definition does not align.
// Type uses the registry difference types: missing, extra, mismatched.
type DefinitionDifference struct {
	ZoneID   int                         `json:"zone_id"`
	ZoneCode string                      `json:"zone_code"`
	Type     string                      `json:"type"`
	Fields   []DefinitionFieldDifference `json:"fields"`
}

// DefinitionAlignmentResult is the outcome of comparing submitted zone definitions
// with the API's own definitions.
type DefinitionAlignmentResult struct {
	Aligned            bool                   `json:"aligned"`
	Status             string                 `json:"status"` // "aligned" | "mismatch"
	PolicyVersion      string                 `json:"policy_version"`
	SignatureAlgorithm string                 `json:"signature_algorithm"`
	Signature          string                 `json:"signature"`          // API definitions
	ExpectedSignature  string                 `json:"expected_signature"` // submitted definitions
	ComparedFields     []string               `json:"compared_fields"`
	ZonesChecked       int                    `json:"zones_checked"`
	Matched            []string               `json:"matched"`
	Differences        []DefinitionDifference `json:"differences"`
}

// ContractFromZone extracts the shared contract from a policy zone definition.
func ContractFromZone(zone ZoneDefinition) DefinitionContract {
	var subzones any
	if len(zone.Subzones) > 0 {
		subzones = zone.Subzones
	}
	return normalizeContract(DefinitionContract{
		Key:            zone.Key,
		Name:           zone.Name,
		Description:    zone.Description,
		Role:           zone.Role,
		ReadOnly:       zone.Rules.ReadOnly,
		MethodsAllowed: zone.Rules.MethodsAllowed,
		Executive:      zone.Rules.Executive,
		Subzones:       subzones,
	})
}

// ParseExpectedDefinitions validates submitted definitions and converts them to contracts.
func ParseExpectedDefinitions(zones map[string]ExpectedDefinition) (map[int]DefinitionContract, error) {
	if len(zones) == 0 {
		return nil, fmt.Errorf("zones must contain at least one zone")
	}

	parsed := make(map[int]DefinitionContract, len(zones))
	for identifier, definition := range zones {
		id, err := ParseZoneIdentifier(identifier)
		if err != nil {
			return nil, err
		}
		if _, exists := parsed[id]; exists {
			return nil, fmt.Errorf("zone Z%d is listed more than once", id)
		}
		if NormalizeZoneKey(definition.Key) == "" {
			return nil, fmt.Errorf("zone %s must have a non-empty key", identifier)
		}
		if strings.TrimSpace(definition.Name) == "" {
			return nil, fmt.Errorf("zone %s must have a non-empty name", identifier)
		}
		if definition.Rules.ReadOnly == nil {
			return nil, fmt.Errorf("zone %s rules.read_only is required", identifier)
		}
		if len(definition.Rules.MethodsAllowed) == 0 {
			return nil, fmt.Errorf("zone %s rules.methods_allowed must list at least one method", identifier)
		}
		for _, method := range definition.Rules.MethodsAllowed {
			if !ValidMethods[strings.ToUpper(strings.TrimSpace(method))] {
				return nil, fmt.Errorf("zone %s has invalid method %q", identifier, method)
			}
		}

		var subzones any
		if trimmed := bytes.TrimSpace(definition.Subzones); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			decoder := json.NewDecoder(bytes.NewReader(trimmed))
			decoder.UseNumber()
			if err := decoder.Decode(&subzones); err != nil {
				return nil, fmt.Errorf("zone %s subzones must be valid JSON: %w", identifier, err)
			}
		}

		contract := DefinitionContract{
			Key:            definition.Key,
			Name:           definition.Name,
			Description:    definition.Description,
			Role:           definition.Role,
			ReadOnly:       *definition.Rules.ReadOnly,
			MethodsAllowed: definition.Rules.MethodsAllowed,
			Subzones:       subzones,
		}
		if definition.Rules.Executive != nil {
			contract.Executive = *definition.Rules.Executive
		}
		parsed[id] = normalizeContract(contract)
	}
	return parsed, nil
}

// CanonicalDefinitionJSON encodes a contract in canonical JSON form.
func CanonicalDefinitionJSON(contract DefinitionContract) string {
	return canonicalJSON(normalizeContract(contract))
}

// DefinitionSignature computes the canonical signature (see DefinitionSignatureAlgorithm).
func DefinitionSignature(definitions map[int]DefinitionContract) string {
	ids := sortedContractIDs(definitions)
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, fmt.Sprintf("Z%d:%s", id, CanonicalDefinitionJSON(definitions[id])))
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// CompareDefinitions compares submitted definitions against actual definitions.
// Aligned is true only when both contain the same zone IDs with identical contracts.
func CompareDefinitions(actual map[int]DefinitionContract, expected map[int]DefinitionContract) DefinitionAlignmentResult {
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

	result := DefinitionAlignmentResult{
		SignatureAlgorithm: DefinitionSignatureAlgorithm,
		Signature:          DefinitionSignature(actual),
		ExpectedSignature:  DefinitionSignature(expected),
		ComparedFields:     DefinitionFields,
		ZonesChecked:       len(ordered),
		Matched:            []string{},
		Differences:        []DefinitionDifference{},
	}

	for _, id := range ordered {
		code := fmt.Sprintf("Z%d", id)
		actualContract, inActual := actual[id]
		expectedContract, inExpected := expected[id]

		switch {
		case inActual && inExpected:
			fields := compareContracts(normalizeContract(expectedContract), normalizeContract(actualContract))
			if len(fields) == 0 {
				result.Matched = append(result.Matched, code)
				continue
			}
			result.Differences = append(result.Differences, DefinitionDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceMismatched, Fields: fields,
			})
		case inExpected:
			result.Differences = append(result.Differences, DefinitionDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceMissing, Fields: []DefinitionFieldDifference{},
			})
		default:
			result.Differences = append(result.Differences, DefinitionDifference{
				ZoneID: id, ZoneCode: code, Type: DifferenceExtra, Fields: []DefinitionFieldDifference{},
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

// DefinitionContracts returns this registry's definitions as shared contracts.
func (r *Registry) DefinitionContracts() map[int]DefinitionContract {
	contracts := make(map[int]DefinitionContract, len(r.snapshot.Zones))
	for id, zone := range r.snapshot.Zones {
		contracts[id] = ContractFromZone(zone)
	}
	return contracts
}

// DefinitionAlignment compares submitted definitions against this policy registry.
// A submitted key that is an alias of the zone's key compares as that key;
// expected_signature still describes the definitions as submitted.
func (r *Registry) DefinitionAlignment(expected map[int]DefinitionContract) DefinitionAlignmentResult {
	resolved := make(map[int]DefinitionContract, len(expected))
	for id, contract := range expected {
		if zone, ok := r.snapshot.Zones[id]; ok && zone.HasKey(contract.Key) {
			contract.Key = zone.Key
		}
		resolved[id] = contract
	}
	result := CompareDefinitions(r.DefinitionContracts(), resolved)
	result.ExpectedSignature = DefinitionSignature(expected)
	result.PolicyVersion = r.snapshot.Version
	return result
}

func compareContracts(expected DefinitionContract, actual DefinitionContract) []DefinitionFieldDifference {
	type field struct {
		name             string
		expected, actual any
	}
	fields := []field{
		{"key", expected.Key, actual.Key},
		{"name", expected.Name, actual.Name},
		{"description", expected.Description, actual.Description},
		{"role", expected.Role, actual.Role},
		{"read_only", expected.ReadOnly, actual.ReadOnly},
		{"methods_allowed", expected.MethodsAllowed, actual.MethodsAllowed},
		{"executive", expected.Executive, actual.Executive},
		{"subzones", expected.Subzones, actual.Subzones},
	}

	differences := []DefinitionFieldDifference{}
	for _, f := range fields {
		if canonicalJSON(f.expected) != canonicalJSON(f.actual) {
			differences = append(differences, DefinitionFieldDifference{Field: f.name, Expected: f.expected, Actual: f.actual})
		}
	}
	return differences
}

func normalizeContract(contract DefinitionContract) DefinitionContract {
	contract.Key = NormalizeZoneKey(contract.Key)
	contract.Name = strings.TrimSpace(contract.Name)
	contract.Description = strings.TrimSpace(contract.Description)
	contract.Role = strings.TrimSpace(contract.Role)

	seen := make(map[string]bool, len(contract.MethodsAllowed))
	methods := make([]string, 0, len(contract.MethodsAllowed))
	for _, method := range contract.MethodsAllowed {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method != "" && !seen[method] {
			seen[method] = true
			methods = append(methods, method)
		}
	}
	sort.Strings(methods)
	contract.MethodsAllowed = methods

	contract.Subzones = normalizeSubzones(contract.Subzones)
	return contract
}

// normalizeSubzones maps absent or empty subzones to an empty list; anything
// else is compared by its canonical JSON form.
func normalizeSubzones(subzones any) any {
	switch typed := subzones.(type) {
	case nil:
		return []any{}
	case []any:
		if len(typed) == 0 {
			return []any{}
		}
	case map[string]any:
		if len(typed) == 0 {
			return []any{}
		}
	}
	return subzones
}

func canonicalJSON(value any) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Sprintf("<unencodable:%v>", err)
	}
	return strings.TrimSuffix(buffer.String(), "\n")
}

func sortedContractIDs(definitions map[int]DefinitionContract) []int {
	ids := make([]int, 0, len(definitions))
	for id := range definitions {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

package communications

import (
	"fmt"
	"net/url"

	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

var (
	destinationPolicy        = policy.NewRegistry()
	destinationRouteAnalyzer = analyzer.NewAnalyzer(destinationPolicy)
)

// zoneAligned reports whether an approved zone fits the area its destination's URI belongs to.
// Zones are recursive (Web3Codex routes/zones/z12/z2, zone path Z12:Z2): the URI names the area,
// and a communication may be a door into it. Here that is the identity door: an approved identity
// (Z2) communication is the way into any area that requires authentication ("sign in to enter").
// Every other difference stays a zone mismatch.
func zoneAligned(approved, area string) bool {
	if approved == area {
		return true
	}
	identity, ok := destinationPolicy.GetZoneByKey("identity")
	if !ok || approved != policy.ZoneIDToCode(identity.ID) {
		return false
	}
	var id int
	if _, e := fmt.Sscanf(area, "Z%d", &id); e != nil || policy.ZoneIDToCode(id) != area || !policy.IsOperationalZone(id) {
		return false
	}
	return destinationPolicy.RequiresAuth(id)
}

type Observation struct {
	ID          int64  `json:"id"`
	Page        string `json:"page"`
	Destination string `json:"destination"`
	Method      string `json:"method"`
	Type        string `json:"type"`
	Zone        string `json:"zone"`
}
type Evaluation struct {
	Effective  string   `json:"effective"`
	Mismatches []string `json:"mismatches"`
}

// Evaluate follows the PHP contract's precedence. Relative approved paths resolve
// against the site's origin; query strings remain part of exact matching.
func Evaluate(status, structure string, revision, approvedStructure int64, doc *Document, observed Observation, baseURL string) Evaluation {
	result := Evaluation{Mismatches: []string{}}
	if doc == nil {
		switch status {
		case "disabled", "revoked", "rejected":
			result.Effective = status
		default:
			result.Effective = "unapproved"
		}
		return result
	}
	if status == "disabled" || status == "revoked" {
		result.Effective = status
		return result
	}
	if !doc.Enabled {
		result.Effective = "disabled"
		return result
	}
	if structure != "valid" {
		result.Effective = "suspended"
		return result
	}
	kind := observed.Type
	if kind == "ajax" {
		kind = "fetch"
	}
	if !sameDestination(doc.Destination, observed.Destination, baseURL) {
		result.Mismatches = append(result.Mismatches, "destination_mismatch")
	}
	found := false
	for _, m := range doc.Methods {
		if m == observed.Method {
			found = true
		}
	}
	if !found {
		result.Mismatches = append(result.Mismatches, "method_mismatch")
	}
	if kind != doc.Type {
		result.Mismatches = append(result.Mismatches, "type_mismatch")
	}
	if !zoneAligned(doc.Zone, destinationZone(observed.Destination, baseURL)) {
		result.Mismatches = append(result.Mismatches, "zone_mismatch")
	}
	if len(result.Mismatches) > 0 {
		result.Effective = "unauthorized"
		return result
	}
	if revision != approvedStructure {
		result.Effective = "reapproval_required"
		return result
	}
	// Unexpected stored states cannot authorize even if a revision is referenced.
	if status != "approved" {
		result.Effective = "unapproved"
		return result
	}
	result.Effective = "approved"
	return result
}
func sameDestination(approved, detected, base string) bool {
	a, okA := resolveDestination(approved, base)
	b, okB := resolveDestination(detected, base)
	return okA && okB && a.String() == b.String()
}

func destinationZone(destination, base string) string {
	resolved, ok := resolveDestination(destination, base)
	if !ok || !sameOrigin(base, resolved.String()) {
		return ""
	}
	path := resolved.Path
	if path == "" {
		path = "/"
	}
	assignment := destinationRouteAnalyzer.GetRouteZone(analyzer.RouteInfo{
		URI:     path,
		Methods: []string{},
	})
	if !policy.IsOperationalZone(assignment.ZoneID) {
		return ""
	}
	return fmt.Sprintf("Z%d", assignment.ZoneID)
}

func resolveDestination(value, base string) (*url.URL, bool) {
	u, err := url.Parse(value)
	if err != nil {
		return nil, false
	}
	if !u.IsAbs() {
		b, err := url.Parse(base)
		if err != nil || b.Host == "" {
			return nil, false
		}
		b.Path = "/"
		b.RawPath = ""
		b.RawQuery = ""
		b.Fragment = ""
		u = b.ResolveReference(u)
	}
	if u.Host == "" {
		return nil, false
	}
	return u, true
}

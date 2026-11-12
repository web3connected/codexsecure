	package schema

import "strings"

// Registry is an in-memory trusted route registry built from a schema artifact.
type Registry struct {
	artifact *Artifact
	records  []Record
	bySchemaID map[string]Record
}

// MatchResult represents a Stage 1 weighted route guess.
type MatchResult struct {
	Matched             bool
	SchemaID            string
	RouteName           string
	RouteFamily         string
	ProposedZoneHint    string
	Confidence          float64
	QuarantineCandidate bool
	QuarantineReason    string
	DetectionMethod     string
}

// NewRegistry builds a registry from a validated artifact.
func NewRegistry(artifact *Artifact) *Registry {
	if artifact == nil {
		return nil
	}

	records := make([]Record, 0, len(artifact.Records))
	bySchemaID := make(map[string]Record)
	for _, record := range artifact.Records {
		if record.Status.Active && record.Status.Trusted {
			records = append(records, record)
			if strings.TrimSpace(record.SchemaID) != "" {
				bySchemaID[record.SchemaID] = record
			}
		}
	}

	return &Registry{artifact: artifact, records: records, bySchemaID: bySchemaID}
}

// LookupSchema returns the trusted schema record for a schema_id.
func (r *Registry) LookupSchema(schemaID string) (Record, bool) {
	if r == nil {
		return Record{}, false
	}
	if strings.TrimSpace(schemaID) == "" {
		return Record{}, false
	}
	rec, ok := r.bySchemaID[schemaID]
	return rec, ok
}

// MatchRequestWithRecord returns the Stage 1 match plus the matched trusted Record (if any).
func (r *Registry) MatchRequestWithRecord(method string, path string) (MatchResult, *Record) {
	match := r.MatchRequest(method, path)
	if !match.Matched {
		return match, nil
	}
	rec, ok := r.LookupSchema(match.SchemaID)
	if !ok {
		return match, nil
	}
	return match, &rec
}

// Artifact returns the source artifact.
func (r *Registry) Artifact() *Artifact {
	if r == nil {
		return nil
	}
	return r.artifact
}

// MatchRequest returns the best weighted guess for a request method/path.
// This does not assign the final zone. It only provides a Stage 1 hypothesis.
func (r *Registry) MatchRequest(method string, path string) MatchResult {
	result := MatchResult{
		Matched:             false,
		ProposedZoneHint:    "Z0",
		Confidence:          0,
		QuarantineCandidate: true,
		QuarantineReason:    "no trusted route pattern matched request",
		DetectionMethod:     "local_schema_preprocess",
	}

	if r == nil || len(r.records) == 0 {
		result.QuarantineReason = "schema registry unavailable"
		return result
	}

	normalizedMethod := normalizeMethod(method)
	normalizedPath := normalizePath(path)
	bestScore := -1.0
	var best Record

	for _, record := range r.records {
		score := 0.0
		recordPath := normalizePath(record.Route.URI)

		if recordPath == normalizedPath {
			score += 0.60
		} else if strings.Trim(recordPath, "/") != "" && strings.Contains(normalizedPath, strings.Trim(recordPath, "/")) {
			score += 0.20
		}

		if methodAllowed(normalizedMethod, record.Route.Methods) {
			score += 0.30
		}

		if strings.TrimSpace(record.Route.Family) != "" && strings.Contains(normalizedPath, strings.ToLower(strings.TrimSpace(record.Route.Family))) {
			score += 0.10
		}

		if score > bestScore {
			bestScore = score
			best = record
		}
	}

	if bestScore < 0.70 {
		result.Confidence = clampScore(bestScore)
		return result
	}

	result.Matched = true
	result.SchemaID = best.SchemaID
	result.RouteName = best.Route.Name
	result.RouteFamily = best.Route.Family
	result.ProposedZoneHint = firstNonEmpty(best.Route.ZoneHint, "Z0")
	result.Confidence = clampScore(bestScore)
	result.QuarantineCandidate = false
	result.QuarantineReason = ""
	return result
}

func normalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.ToLower(trimmed)
}

func methodAllowed(method string, allowed []string) bool {
	for _, item := range allowed {
		if normalizeMethod(item) == method {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
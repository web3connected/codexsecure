package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Artifact is the top-level page-schema.json contract loaded by CodexSecure.
type Artifact struct {
	ArtifactVersion string             `json:"artifact_version"`
	SchemaSetID     string             `json:"schema_set_id"`
	Application     ApplicationMeta    `json:"application"`
	Generation      GenerationMeta     `json:"generation"`
	Policies        ArtifactPolicies   `json:"policies"`
	PhaseLock       PhaseLockConfig    `json:"phase_lock"`
	Records         []Record           `json:"records"`
}

type ApplicationMeta struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	BaseURL     string `json:"base_url"`
	Framework   string `json:"framework"`
}

type GenerationMeta struct {
	Generator     string `json:"generator"`
	GeneratedAt   string `json:"generated_at"`
	TrustMode     string `json:"trust_mode"`
	BaselineState string `json:"baseline_state"`
}

type ArtifactPolicies struct {
	DefaultEnforcementProfile string `json:"default_enforcement_profile"`
	DefaultZoneOnUnknownRoute string `json:"default_zone_on_unknown_route"`
	DefaultOnSchemaMiss       string `json:"default_on_schema_miss"`
	DefaultOnDiff             string `json:"default_on_diff"`
}

type PhaseLockConfig struct {
	Enabled             bool               `json:"enabled"`
	Algorithm           string             `json:"algorithm"`
	StrongLockThreshold float64            `json:"strong_lock_threshold"`
	DefaultScore        float64            `json:"default_score"`
	Weights             map[string]float64 `json:"weights"`
}

type Record struct {
	SchemaID      string       `json:"schema_id"`
	SchemaVersion string       `json:"schema_version"`
	Route         RouteMeta    `json:"route"`
	Intent        IntentMeta   `json:"intent"`
	Status        RecordStatus `json:"status"`
}

type RouteMeta struct {
	Name     string   `json:"name"`
	Family   string   `json:"family"`
	URI      string   `json:"uri"`
	Methods  []string `json:"methods"`
	ZoneHint string   `json:"zone_hint"`
}

type IntentMeta struct {
	Type      string `json:"type"`
	Operation string `json:"operation"`
}

type RecordStatus struct {
	Trusted     bool   `json:"trusted"`
	Active      bool   `json:"active"`
	ReviewState string `json:"review_state"`
}

// Summary describes the loaded schema artifact in logs and diagnostics.
type Summary struct {
	SchemaSetID     string
	ApplicationName string
	Environment     string
	RecordCount     int
	BaselineState   string
	TrustMode       string
	PhaseLock       bool
}

// LoadArtifact reads, parses, and validates a page schema artifact from disk.
func LoadArtifact(path string) (*Artifact, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}

	var artifact Artifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		return nil, fmt.Errorf("parse artifact: %w", err)
	}

	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	return &artifact, nil
}

// Validate checks the minimum contract required for startup loading.
func (a *Artifact) Validate() error {
	if strings.TrimSpace(a.ArtifactVersion) == "" {
		return fmt.Errorf("artifact_version is required")
	}
	if strings.TrimSpace(a.SchemaSetID) == "" {
		return fmt.Errorf("schema_set_id is required")
	}
	if strings.TrimSpace(a.Application.Name) == "" {
		return fmt.Errorf("application.name is required")
	}
	if strings.TrimSpace(a.Generation.GeneratedAt) == "" {
		return fmt.Errorf("generation.generated_at is required")
	}
	if len(a.Records) == 0 {
		return fmt.Errorf("records must contain at least one schema record")
	}

	for index, record := range a.Records {
		if strings.TrimSpace(record.SchemaID) == "" {
			return fmt.Errorf("records[%d].schema_id is required", index)
		}
		if strings.TrimSpace(record.Route.Name) == "" {
			return fmt.Errorf("records[%d].route.name is required", index)
		}
		if strings.TrimSpace(record.Route.URI) == "" {
			return fmt.Errorf("records[%d].route.uri is required", index)
		}
		if len(record.Route.Methods) == 0 {
			return fmt.Errorf("records[%d].route.methods must contain at least one method", index)
		}
	}

	if a.PhaseLock.Enabled {
		if a.PhaseLock.DefaultScore < 0 || a.PhaseLock.DefaultScore > 1 {
			return fmt.Errorf("phase_lock.default_score must be between 0 and 1")
		}
		if a.PhaseLock.StrongLockThreshold < 0 || a.PhaseLock.StrongLockThreshold > 1 {
			return fmt.Errorf("phase_lock.strong_lock_threshold must be between 0 and 1")
		}
	}

	return nil
}

// Summary returns the key runtime facts about the loaded schema artifact.
func (a *Artifact) Summary() Summary {
	return Summary{
		SchemaSetID:     a.SchemaSetID,
		ApplicationName: a.Application.Name,
		Environment:     a.Application.Environment,
		RecordCount:     len(a.Records),
		BaselineState:   a.Generation.BaselineState,
		TrustMode:       a.Generation.TrustMode,
		PhaseLock:       a.PhaseLock.Enabled,
	}
}
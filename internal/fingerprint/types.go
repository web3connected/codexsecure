package fingerprint

// Fingerprint Types
// Structs for request analysis and sync accuracy detection.
//
// Converted from: CodexRequestAnalyzer.php
// @version 2.0.0

// SyncStatus constants
const (
	SyncStatusPerfect = "perfect" // drift == 0.0
	SyncStatusGood    = "good"    // drift <= threshold (default 0.01)
	SyncStatusWarning = "warning" // drift <= 0.05
	SyncStatusError   = "error"   // drift > 0.05
)

// Default thresholds
const (
	DefaultTIUSyncThreshold = 0.01
	WarningThreshold        = 0.05
)

// RouteData represents the route attributes for fingerprinting
type RouteData struct {
	URI        string `json:"uri"`
	Controller string `json:"controller"`
	Method     string `json:"method"`
}

// AnalysisDetails provides context about the analysis
type AnalysisDetails struct {
	Route       string         `json:"route"`
	Controller  string         `json:"controller"`
	InputSample map[string]any `json:"input_sample,omitempty"`
}

// AnalysisResult is the output of request analysis
type AnalysisResult struct {
	RouteFingerprint string          `json:"route_fingerprint"`
	InputFingerprint string          `json:"input_fingerprint"`
	Drift            float64         `json:"drift"`
	SyncStatus       string          `json:"sync_status"`
	Details          AnalysisDetails `json:"details"`
}

// AnalyzeRequest is the input for request analysis
type AnalyzeRequest struct {
	Route RouteData      `json:"route" binding:"required"`
	Input map[string]any `json:"input"`
}

// FingerprintRequest is the input for generating a fingerprint
type FingerprintRequest struct {
	Route RouteData `json:"route" binding:"required"`
}

// DriftRequest is the input for calculating drift between two fingerprints
type DriftRequest struct {
	Fingerprint1 string `json:"fingerprint1" binding:"required"`
	Fingerprint2 string `json:"fingerprint2" binding:"required"`
}

// DriftResult is the output of drift calculation
type DriftResult struct {
	Drift      float64 `json:"drift"`
	SyncStatus string  `json:"sync_status"`
}

// FilterRequest is the input for filtering intent fields
type FilterRequest struct {
	Input map[string]any `json:"input" binding:"required"`
}

// FilterResult is the output of intent field filtering
type FilterResult struct {
	Filtered       map[string]any `json:"filtered"`
	OriginalCount  int            `json:"original_count"`
	FilteredCount  int            `json:"filtered_count"`
	IntentFields   []string       `json:"intent_fields_matched"`
	UsedFallback   bool           `json:"used_fallback"`
}

// Config holds fingerprint analyzer configuration
type Config struct {
	TIUSyncThreshold float64  `json:"tiu_sync_threshold"`
	IntentFields     []string `json:"intent_fields"`
	NoiseFields      []string `json:"noise_fields"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		TIUSyncThreshold: DefaultTIUSyncThreshold,
		IntentFields:     DefaultIntentFields(),
		NoiseFields:      DefaultNoiseFields(),
	}
}

// DefaultIntentFields returns the default intent-driven field patterns
func DefaultIntentFields() []string {
	return []string{
		"email",
		"username",
		"password",
		"query",
		"search",
		"title",
		"content",
		"message",
	}
}

// DefaultNoiseFields returns fields to strip before fingerprinting
func DefaultNoiseFields() []string {
	return []string{
		"_token",
		"_method",
		"csrf_token",
		"csrf",
		"_csrf",
	}
}

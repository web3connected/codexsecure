package fingerprint

// Fingerprint - Request fingerprinting and sync analysis
// Generates fingerprints and calculates drift for TIU sync verification.
//
// Converted from: CodexRequestAnalyzer.php
// @version 2.0.0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// Analyzer analyzes requests for sync accuracy.
// When a CodexHash client is provided, remote hashing is preferred;
// local sha256 is used as a fail-open fallback.
type Analyzer struct {
	config     *Config
	hashClient *codexhash.Client
}

// NewAnalyzer creates a new request analyzer.
// Pass a non-nil *codexhash.Client to enable API-backed hashing.
func NewAnalyzer(config *Config) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Analyzer{config: config, hashClient: codexhash.NewClient()}
}

// NewAnalyzerWithHashClient creates a new request analyzer with an explicit CodexHash client.
func NewAnalyzerWithHashClient(config *Config, hc *codexhash.Client) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Analyzer{config: config, hashClient: hc}
}

// Analyze performs full request analysis comparing route to input
func (a *Analyzer) Analyze(route RouteData, input map[string]any) AnalysisResult {
	// Generate route fingerprint
	routeFingerprint := a.GenerateRouteFingerprint(route)

	// Filter and fingerprint input
	filterResult := FilterIntentFields(input, a.config)
	inputFingerprint := a.GenerateInputFingerprint(filterResult.Filtered)

	// Calculate drift
	drift := a.CalculateDrift(routeFingerprint, inputFingerprint)

	// Determine sync status
	syncStatus := a.GetSyncStatus(drift)

	// Build input sample (first 5 fields)
	inputSample := buildInputSample(filterResult.Filtered, 5)

	return AnalysisResult{
		RouteFingerprint: routeFingerprint,
		InputFingerprint: inputFingerprint,
		Drift:            drift,
		SyncStatus:       syncStatus,
		Details: AnalysisDetails{
			Route:       route.URI,
			Controller:  route.Controller,
			InputSample: inputSample,
		},
	}
}

// GenerateRouteFingerprint creates a fingerprint from route data
func (a *Analyzer) GenerateRouteFingerprint(route RouteData) string {
	// Build canonical data map
	data := map[string]string{
		"controller": extractBaseName(route.Controller),
		"method":     route.Method,
		"uri":        route.URI,
	}

	// Keys are already sorted alphabetically in the map literal
	return a.hashData(data)
}

// GenerateInputFingerprint creates a fingerprint from filtered input
func (a *Analyzer) GenerateInputFingerprint(input map[string]any) string {
	if len(input) == 0 {
		return a.hashData(map[string]any{})
	}
	return a.hashData(input)
}

// CalculateDrift calculates the drift between two fingerprints
// Returns a value between 0.0 (perfect match) and 1.0 (complete mismatch)
func (a *Analyzer) CalculateDrift(fingerprint1, fingerprint2 string) float64 {
	if fingerprint1 == fingerprint2 {
		return 0.0
	}

	// Convert hex to bytes for comparison
	bytes1, err1 := hex.DecodeString(fingerprint1)
	bytes2, err2 := hex.DecodeString(fingerprint2)

	if err1 != nil || err2 != nil {
		return 1.0 // Maximum drift on error
	}

	// Calculate Hamming distance as drift
	return calculateHammingDrift(bytes1, bytes2)
}

// GetSyncStatus determines sync status from drift value
func (a *Analyzer) GetSyncStatus(drift float64) string {
	switch {
	case drift == 0.0:
		return SyncStatusPerfect
	case drift <= a.config.TIUSyncThreshold:
		return SyncStatusGood
	case drift <= WarningThreshold:
		return SyncStatusWarning
	default:
		return SyncStatusError
	}
}

// GetConfig returns the current configuration
func (a *Analyzer) GetConfig() *Config {
	return a.config
}

// SetConfig updates the configuration
func (a *Analyzer) SetConfig(config *Config) {
	a.config = config
}

// hashData generates a hash of the data.
// When a CodexHash client is configured it calls the API; otherwise falls back to local sha256.
func (a *Analyzer) hashData(data any) string {
	// Serialize to canonical JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return strings.Repeat("0", 64)
	}
	serialized := string(jsonBytes)

	// Try CodexHash API first
	if a.hashClient != nil {
		result := a.hashClient.QuickHash(context.Background(), serialized)
		if result != "" && result != strings.Repeat("0", 64) {
			return result
		}
	}

	// Local sha256 fallback
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// calculateHammingDrift calculates normalized Hamming distance
func calculateHammingDrift(a, b []byte) float64 {
	if len(a) != len(b) {
		return 1.0
	}

	totalBits := len(a) * 8
	diffBits := 0

	for i := range a {
		xor := a[i] ^ b[i]
		diffBits += popCount(xor)
	}

	return float64(diffBits) / float64(totalBits)
}

// popCount counts the number of set bits (population count)
func popCount(b byte) int {
	count := 0
	for b != 0 {
		count += int(b & 1)
		b >>= 1
	}
	return count
}

// extractBaseName extracts the base class name from a fully qualified name
func extractBaseName(fqn string) string {
	if fqn == "" {
		return ""
	}

	// Handle PHP-style namespace separators
	if idx := strings.LastIndex(fqn, "\\"); idx != -1 {
		return fqn[idx+1:]
	}

	// Handle Go-style package paths
	if idx := strings.LastIndex(fqn, "/"); idx != -1 {
		return fqn[idx+1:]
	}

	// Handle controller@method format
	if idx := strings.Index(fqn, "@"); idx != -1 {
		return extractBaseName(fqn[:idx])
	}

	return fqn
}

// buildInputSample extracts the first n fields from filtered input
func buildInputSample(input map[string]any, n int) map[string]any {
	if len(input) == 0 {
		return nil
	}

	// Get sorted keys
	keys := GetSortedKeys(input)

	// Take first n
	if len(keys) > n {
		keys = keys[:n]
	}

	sample := make(map[string]any)
	for _, key := range keys {
		sample[key] = input[key]
	}

	return sample
}

// CompareFingerprints compares two fingerprints and returns detailed result
func (a *Analyzer) CompareFingerprints(fp1, fp2 string) DriftResult {
	drift := a.CalculateDrift(fp1, fp2)
	return DriftResult{
		Drift:      math.Round(drift*10000) / 10000, // Round to 4 decimal places
		SyncStatus: a.GetSyncStatus(drift),
	}
}

// QuickFingerprint generates a fingerprint for any data
func (a *Analyzer) QuickFingerprint(data any) string {
	return a.hashData(data)
}

// AnalyzeWithContext performs analysis with additional context
func (a *Analyzer) AnalyzeWithContext(route RouteData, input map[string]any, context map[string]any) AnalysisResult {
	result := a.Analyze(route, input)

	// Add context to details if provided
	if context != nil {
		for key, value := range context {
			// Only add non-overlapping context
			if _, exists := result.Details.InputSample[key]; !exists {
				if result.Details.InputSample == nil {
					result.Details.InputSample = make(map[string]any)
				}
				result.Details.InputSample["_ctx_"+key] = value
			}
		}
	}

	return result
}

// HashInput hashes arbitrary input data
func HashInput(input map[string]any) string {
	analyzer := NewAnalyzer(nil)
	return analyzer.QuickFingerprint(sortMapKeys(input))
}

// HashRoute hashes route data
func HashRoute(route RouteData) string {
	analyzer := NewAnalyzer(nil)
	return analyzer.GenerateRouteFingerprint(route)
}

// sortMapByKeys returns data for JSON marshaling with sorted keys
func sortMapByKeys(m map[string]any) map[string]any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make(map[string]any)
	for _, k := range keys {
		result[k] = m[k]
	}
	return result
}

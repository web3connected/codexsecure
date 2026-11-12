package fingerprint

// Filter - Intent field filtering
// Extracts intent-driven fields from request input for stable fingerprinting.
//
// Converted from: CodexRequestAnalyzer.php filterIntentFields()
// @version 2.0.0

import (
	"sort"
	"strings"
)

// FilterIntentFields extracts only intent-driven fields from input
// Steps:
// 1. Remove noise fields (_token, csrf, etc.)
// 2. Normalize keys and values (lowercase, trim)
// 3. Filter to intent fields (email, username, password, etc.)
// 4. Fallback to normalized if filtering removed everything
// 5. Sort keys for stable hashing
func FilterIntentFields(input map[string]any, config *Config) FilterResult {
	if config == nil {
		config = DefaultConfig()
	}

	result := FilterResult{
		OriginalCount:  len(input),
		IntentFields:   []string{},
		UsedFallback:   false,
	}

	if len(input) == 0 {
		result.Filtered = make(map[string]any)
		return result
	}

	// Step 1: Remove noise fields
	cleaned := removeNoiseFields(input, config.NoiseFields)

	// Step 2: Normalize input
	normalized := normalizeInput(cleaned)

	// Step 3: Filter to intent fields
	filtered, matchedFields := filterToIntentFields(normalized, config.IntentFields)
	result.IntentFields = matchedFields

	// Step 4: Fallback if filtering removed everything
	if len(filtered) == 0 {
		filtered = normalized
		result.UsedFallback = true
	}

	// Step 5: Sort keys (Go maps are unordered, so we return sorted)
	result.Filtered = sortMapKeys(filtered)
	result.FilteredCount = len(result.Filtered)

	return result
}

// removeNoiseFields removes common noise fields from input
func removeNoiseFields(input map[string]any, noiseFields []string) map[string]any {
	result := make(map[string]any)

	noiseSet := make(map[string]bool)
	for _, nf := range noiseFields {
		noiseSet[strings.ToLower(nf)] = true
	}

	for key, value := range input {
		if !noiseSet[strings.ToLower(key)] {
			result[key] = value
		}
	}

	return result
}

// normalizeInput normalizes keys and values (lowercase, trim)
func normalizeInput(input map[string]any) map[string]any {
	normalized := make(map[string]any)

	for key, value := range input {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		normalizedValue := normalizeValue(value)
		normalized[normalizedKey] = normalizedValue
	}

	return normalized
}

// normalizeValue normalizes a value (lowercase string, recursive for arrays)
func normalizeValue(value any) any {
	switch v := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(v))
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeValue(item)
		}
		return result
	case []string:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = strings.ToLower(strings.TrimSpace(item))
		}
		return result
	case map[string]any:
		return normalizeInput(v)
	default:
		return value
	}
}

// filterToIntentFields keeps only fields that match intent patterns
func filterToIntentFields(input map[string]any, intentFields []string) (map[string]any, []string) {
	filtered := make(map[string]any)
	matchedFields := []string{}

	for key, value := range input {
		for _, intentField := range intentFields {
			if strings.Contains(key, intentField) {
				filtered[key] = value
				matchedFields = append(matchedFields, intentField)
				break
			}
		}
	}

	// Deduplicate matched fields
	matchedFields = uniqueStrings(matchedFields)

	return filtered, matchedFields
}

// sortMapKeys returns a new map with keys in sorted order
// Note: Go maps don't preserve order, but this ensures deterministic iteration
func sortMapKeys(input map[string]any) map[string]any {
	// Get sorted keys
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build ordered map (for JSON serialization, order is preserved)
	result := make(map[string]any)
	for _, key := range keys {
		result[key] = input[key]
	}

	return result
}

// uniqueStrings removes duplicates from a string slice
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	sort.Strings(result)
	return result
}

// GetSortedKeys returns the keys of a map in sorted order
func GetSortedKeys(input map[string]any) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

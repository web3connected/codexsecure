package models

import (
	"encoding/json"
	"time"
)

// Zone represents a security zone in the CodexSecure system
type Zone struct {
	ID               int64           `json:"id" db:"id"`
	ZoneCode         string          `json:"zone_code" db:"zone_code"`          // Z1, Z2:Z01, etc.
	ZoneName         string          `json:"zone_name" db:"zone_name"`          // public, identity, etc.
	ParentZone       *string         `json:"parent_zone" db:"parent_zone"`      // For sub-zones
	FrequencyHz      float64         `json:"frequency_hz" db:"frequency_hz"`    // Harmonic frequency
	HarmonicSignature string         `json:"harmonic_signature" db:"harmonic_signature"`
	RiskLevel        int             `json:"risk_level" db:"risk_level"`        // 0-12
	Description      string          `json:"description" db:"description"`
	MiddlewareStack  json.RawMessage `json:"middleware_stack" db:"middleware_stack"` // JSON array
	IsActive         bool            `json:"is_active" db:"is_active"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// Middleware returns the middleware stack as a string slice
func (z *Zone) Middleware() ([]string, error) {
	var middleware []string
	if err := json.Unmarshal(z.MiddlewareStack, &middleware); err != nil {
		return nil, err
	}
	return middleware, nil
}

// IsSubZone checks if this is a sub-zone (fractal)
func (z *Zone) IsSubZone() bool {
	return z.ParentZone != nil && *z.ParentZone != ""
}

// ZoneConfig holds configuration for a zone
type ZoneConfig struct {
	ZoneID          string                 `json:"zone_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Middleware      []string               `json:"middleware"`
	RateLimits      *RateLimitConfig       `json:"rate_limits,omitempty"`
	AllowedPatterns []string               `json:"allowed_patterns,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstSize         int `json:"burst_size"`
}

// ZoneHealth represents health status of a zone
type ZoneHealth struct {
	ZoneCode       string    `json:"zone_code"`
	IsHealthy      bool      `json:"is_healthy"`
	ActiveRequests int       `json:"active_requests"`
	ErrorRate      float64   `json:"error_rate"`
	LastChecked    time.Time `json:"last_checked"`
}

// ZoneMetrics holds metrics for a zone
type ZoneMetrics struct {
	ZoneCode         string  `json:"zone_code"`
	TotalRequests    int64   `json:"total_requests"`
	SuccessfulRoutes int64   `json:"successful_routes"`
	FailedRoutes     int64   `json:"failed_routes"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ValidationCount  int64   `json:"validation_count"`
	TIUEventCount    int64   `json:"tiu_event_count"`
}

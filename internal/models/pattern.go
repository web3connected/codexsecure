package models

import (
	"time"
)

// PatternType represents the type of pattern detected
type PatternType string

const (
	PatternTypeEmail       PatternType = "email"
	PatternTypePhone       PatternType = "phone"
	PatternTypeCard        PatternType = "card"
	PatternTypeIP          PatternType = "ip"
	PatternTypeURL         PatternType = "url"
	PatternTypeUsername    PatternType = "username"
	PatternTypePassword    PatternType = "password"
	PatternTypeJSON        PatternType = "json"
	PatternTypeXML         PatternType = "xml"
	PatternTypeCustom      PatternType = "custom"
)

// Pattern represents a detected pattern and its zone mapping
type Pattern struct {
	ID          int64       `json:"id" db:"id"`
	PatternHash string      `json:"pattern_hash" db:"pattern_hash"` // SHA-256 hash
	PatternType PatternType `json:"pattern_type" db:"pattern_type"`
	Intent      string      `json:"intent" db:"intent"`           // login, payment, etc.
	ZoneCode    string      `json:"zone_code" db:"zone_code"`     // Target zone
	Confidence  float64     `json:"confidence" db:"confidence"`   // 0.0-1.0
	IsApproved  bool        `json:"is_approved" db:"is_approved"` // Manual approval for high-risk
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// PatternDetectionRequest represents a request to detect pattern
type PatternDetectionRequest struct {
	Data    string            `json:"data"`              // Data to analyze (not stored)
	Context map[string]string `json:"context,omitempty"` // Additional context (route, action, etc.)
}

// PatternDetectionResult represents the result of pattern detection
type PatternDetectionResult struct {
	PatternHash     string      `json:"pattern_hash"`
	PatternType     PatternType `json:"pattern_type"`
	DetectedIntent  string      `json:"detected_intent"`
	RecommendedZone string      `json:"recommended_zone"`
	Confidence      float64     `json:"confidence"`
	RequiresApproval bool       `json:"requires_approval"`
}

// PatternRegistration represents a new pattern registration
type PatternRegistration struct {
	PatternHash string      `json:"pattern_hash"`
	PatternType PatternType `json:"pattern_type"`
	Intent      string      `json:"intent"`
	ZoneCode    string      `json:"zone_code"`
	Metadata    string      `json:"metadata,omitempty"` // JSON metadata
}

// IntentType represents common intent types in the system
type IntentType string

const (
	IntentAuthLogin       IntentType = "auth.login"
	IntentAuthRegister    IntentType = "auth.register"
	IntentAuthVerify      IntentType = "auth.verify"
	IntentAuthPasswordReset IntentType = "auth.password_reset"
	IntentPaymentCreate   IntentType = "payment.create"
	IntentPaymentVerify   IntentType = "payment.verify"
	IntentPaymentRefund   IntentType = "payment.refund"
	IntentUserProfile     IntentType = "user.profile"
	IntentUserSettings    IntentType = "user.settings"
	IntentUserDelete      IntentType = "user.delete"
	IntentAdminManage     IntentType = "admin.manage"
	IntentAdminAudit      IntentType = "admin.audit"
	IntentAnalyticsView   IntentType = "analytics.view"
	IntentAnalyticsExport IntentType = "analytics.export"
)

// IntentContext provides context for intent derivation
type IntentContext struct {
	Route       string            `json:"route"`        // Request path
	Method      string            `json:"method"`       // HTTP method
	Action      string            `json:"action"`       // Action name from naming convention
	Headers     map[string]string `json:"headers"`      // Request headers
	QueryParams map[string]string `json:"query_params"` // Query parameters
}

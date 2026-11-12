package middleware

// Zone Violation Errors
// Custom error types for zone violations.
//
// Converted from: ExceptionsCodexZoneViolation.php
// @version 2.0.0

import (
	"fmt"
	"time"
)

// ViolationType defines types of zone violations
type ViolationType string

const (
	ViolationUnauthorized   ViolationType = "UNAUTHORIZED"
	ViolationMethodDenied   ViolationType = "METHOD_DENIED"
	ViolationTIUDrift       ViolationType = "TIU_DRIFT"
	ViolationSessionMissing ViolationType = "SESSION_MISSING"
	ViolationSessionExpired ViolationType = "SESSION_EXPIRED"
	ViolationZoneLocked     ViolationType = "ZONE_LOCKED"
	ViolationZoneNotFound   ViolationType = "ZONE_NOT_FOUND"
	ViolationAuthRequired   ViolationType = "AUTH_REQUIRED"
	ViolationExecRequired   ViolationType = "EXECUTIVE_REQUIRED"
	ViolationTierDenied     ViolationType = "TIER_DENIED"
)

// ZoneViolation represents a zone access violation
type ZoneViolation struct {
	Zone      string            `json:"zone"`
	Type      ViolationType     `json:"type"`
	Message   string            `json:"message"`
	Context   map[string]any    `json:"context,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	Timestamp int64             `json:"timestamp"`
	Code      int               `json:"code"`
}

// Error implements the error interface
func (e *ZoneViolation) Error() string {
	return fmt.Sprintf("[%s] Zone %s: %s", e.Type, e.Zone, e.Message)
}

// NewZoneViolation creates a new zone violation error
func NewZoneViolation(zone string, vType ViolationType, msg string) *ZoneViolation {
	return &ZoneViolation{
		Zone:      zone,
		Type:      vType,
		Message:   msg,
		Context:   make(map[string]any),
		Timestamp: time.Now().Unix(),
		Code:      403,
	}
}

// WithContext adds context to the violation
func (e *ZoneViolation) WithContext(key string, value any) *ZoneViolation {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// WithUserID sets the user ID
func (e *ZoneViolation) WithUserID(userID string) *ZoneViolation {
	e.UserID = userID
	return e
}

// WithCode sets the HTTP status code
func (e *ZoneViolation) WithCode(code int) *ZoneViolation {
	e.Code = code
	return e
}

// ToJSON returns a JSON-compatible map
func (e *ZoneViolation) ToJSON() map[string]any {
	result := map[string]any{
		"error":     e.Message,
		"code":      string(e.Type),
		"zone":      e.Zone,
		"timestamp": e.Timestamp,
	}

	if len(e.Context) > 0 {
		result["context"] = e.Context
	}

	if e.UserID != "" {
		result["user_id"] = e.UserID
	}

	return result
}

// Standard zone violation errors
var (
	ErrZoneUnauthorized = NewZoneViolation("", ViolationUnauthorized, "Unauthorized zone access").WithCode(403)
	ErrMethodDenied     = NewZoneViolation("", ViolationMethodDenied, "Method not allowed for zone").WithCode(405)
	ErrAuthRequired     = NewZoneViolation("", ViolationAuthRequired, "Authentication required").WithCode(401)
	ErrExecRequired     = NewZoneViolation("", ViolationExecRequired, "Executive access required").WithCode(403)
)

// IsZoneViolation checks if an error is a zone violation
func IsZoneViolation(err error) bool {
	_, ok := err.(*ZoneViolation)
	return ok
}

// GetViolationType extracts violation type from error
func GetViolationType(err error) ViolationType {
	if zv, ok := err.(*ZoneViolation); ok {
		return zv.Type
	}
	return ""
}

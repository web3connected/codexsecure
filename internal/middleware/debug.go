package middleware

// Debug Middleware - Zone debug headers
// Adds debug headers for zone information when enabled.
//
// Converted from: CodexDebugZoneMiddleware.php
// @version 2.0.0

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Debug header names
const (
	HeaderRequestID     = "X-Codex-Request-ID"
	HeaderExecutionTime = "X-Codex-Execution-Time"
	HeaderMemoryUsage   = "X-Codex-Memory-Usage"
	HeaderZoneID        = "X-Codex-Zone-ID"
	HeaderZoneName      = "X-Codex-Zone-Name"
	HeaderRouteName     = "X-Codex-Route-Name"
	HeaderController    = "X-Codex-Controller"
	HeaderMethod        = "X-Codex-Method"
	HeaderPath          = "X-Codex-Path"
	HeaderVersion       = "X-Codex-Version"
)

// DebugConfig configures debug middleware behavior
type DebugConfig struct {
	// Enabled turns debug headers on/off
	Enabled bool

	// DebugIPs is a list of IPs that always get debug headers
	DebugIPs []string

	// Version is the app version to include in headers
	Version string

	// AllowDebugHeader allows X-Debug-Mode header to enable debugging
	AllowDebugHeader bool
}

// DefaultDebugConfig returns development-friendly defaults
func DefaultDebugConfig() *DebugConfig {
	return &DebugConfig{
		Enabled:          true,
		DebugIPs:         []string{"127.0.0.1", "::1"},
		Version:          "2.0.0",
		AllowDebugHeader: true,
	}
}

// DebugMiddleware adds debug headers when enabled
func DebugMiddleware(cfg *DebugConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultDebugConfig()
	}

	return func(c *gin.Context) {
		// Check if debug should be enabled for this request
		if !shouldAddDebugHeaders(c, cfg) {
			c.Next()
			return
		}

		// Start timing
		start := time.Now()
		requestID := uuid.New().String()

		// Store request ID in context
		c.Set("request_id", requestID)

		// Process request
		c.Next()

		// Calculate metrics
		executionTime := time.Since(start)
		memUsage := getMemoryUsage()

		// Add debug headers
		addDebugHeaders(c, requestID, executionTime, memUsage, cfg.Version)
	}
}

// shouldAddDebugHeaders determines if debug headers should be added
func shouldAddDebugHeaders(c *gin.Context, cfg *DebugConfig) bool {
	// Check global enable
	if cfg.Enabled {
		return true
	}

	// Check debug header
	if cfg.AllowDebugHeader && c.GetHeader("X-Debug-Mode") != "" {
		return true
	}

	// Check debug IPs
	clientIP := c.ClientIP()
	for _, ip := range cfg.DebugIPs {
		if ip == clientIP {
			return true
		}
	}

	return false
}

// addDebugHeaders adds all debug headers to response
func addDebugHeaders(c *gin.Context, requestID string, execTime time.Duration, memUsage string, version string) {
	// Request metadata
	c.Header(HeaderRequestID, requestID)
	c.Header(HeaderExecutionTime, fmt.Sprintf("%.2fms", float64(execTime.Microseconds())/1000))
	c.Header(HeaderMemoryUsage, memUsage)

	// Zone information
	c.Header(HeaderZoneID, fmt.Sprintf("%d", GetZoneID(c)))
	c.Header(HeaderZoneName, GetZoneName(c))

	// Route information
	c.Header(HeaderRouteName, c.FullPath())
	c.Header(HeaderMethod, c.Request.Method)
	c.Header(HeaderPath, c.Request.URL.Path)
	c.Header(HeaderVersion, version)

	// Controller info (if available)
	if handler := c.HandlerName(); handler != "" {
		c.Header(HeaderController, handler)
	}
}

// getMemoryUsage returns current memory usage as string
func getMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
}

// GetRequestID retrieves request ID from context
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get("request_id"); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// RequestIDMiddleware ensures every request has a request ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing request ID header
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header(HeaderRequestID, requestID)

		c.Next()
	}
}

// TimingMiddleware adds execution time header
func TimingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		c.Header(HeaderExecutionTime, fmt.Sprintf("%.2fms", float64(time.Since(start).Microseconds())/1000))
	}
}

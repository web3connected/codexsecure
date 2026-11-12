package analyzer

// Strategies - Zone detection patterns
// URI and controller matching strategies for zone assignment.
//
// Converted from: CodexZoneStrategies.php
// @version 2.0.0

import "github.com/web3connected/codexsecure-api/internal/policy"

// ZoneKeyword maps zone keys to their detection keywords
type ZoneKeyword struct {
	ZoneKey  string
	ZoneID   int
	Keywords []string
}

// DefaultURIStrategies returns URI-based zone detection patterns
// Matches zone key from policy package
func DefaultURIStrategies() []ZoneKeyword {
	return []ZoneKeyword{
		{
			ZoneKey:  "boundary",
			ZoneID:   0,
			Keywords: getBoundaryKeywords(),
		},
		{
			ZoneKey:  "public",
			ZoneID:   1,
			Keywords: getPublicKeywords(),
		},
		{
			ZoneKey:  "identity",
			ZoneID:   2,
			Keywords: getIdentityKeywords(),
		},
		{
			ZoneKey:  "operations",
			ZoneID:   3,
			Keywords: getOperationsKeywords(),
		},
		{
			ZoneKey:  "protected",
			ZoneID:   4,
			Keywords: getProtectedKeywords(),
		},
		{
			ZoneKey:  "processing",
			ZoneID:   5,
			Keywords: getProcessingKeywords(),
		},
		{
			ZoneKey:  "logging",
			ZoneID:   6,
			Keywords: getLoggingKeywords(),
		},
		{
			ZoneKey:  "communications",
			ZoneID:   7,
			Keywords: getCommunicationsKeywords(),
		},
		{
			ZoneKey:  "payments",
			ZoneID:   8,
			Keywords: getPaymentsKeywords(),
		},
		{
			ZoneKey:  "analytics",
			ZoneID:   9,
			Keywords: getAnalyticsKeywords(),
		},
		{
			ZoneKey:  "streaming",
			ZoneID:   10,
			Keywords: getStreamingKeywords(),
		},
		{
			ZoneKey:  "synchronization",
			ZoneID:   11,
			Keywords: getSynchronizationKeywords(),
		},
		{
			ZoneKey:  "administration",
			ZoneID:   12,
			Keywords: getAdministrationKeywords(),
		},
	}
}

// DefaultControllerStrategies returns controller name patterns
func DefaultControllerStrategies() map[string]string {
	return map[string]string{
		"Home":         "public",
		"Login":        "identity",
		"Register":     "identity",
		"Logout":       "identity",
		"Auth":         "identity",
		"Dashboard":    "protected",
		"Profile":      "protected",
		"Settings":     "protected",
		"File":         "protected",
		"User":         "protected",
		"Export":       "protected",
		"Log":          "logging",
		"Event":        "logging",
		"System":       "processing",
		"Job":          "processing",
		"Task":         "processing",
		"Notification": "communications",
		"Message":      "communications",
		"Chat":         "communications",
		"Payment":      "payments",
		"Invoice":      "payments",
		"Analytics":    "analytics",
		"Report":       "analytics",
		"Stream":       "streaming",
		"Admin":        "administration",
	}
}

// DefaultMiddlewareStrategies maps middleware combinations to zones
func DefaultMiddlewareStrategies() map[string][]string {
	return map[string][]string{
		"administration": {"auth:admin", "admin", "super"},
		"identity":       {"guest", "throttle"},
		"protected":      {"auth", "verified"},
		"processing":     {"api", "throttle:api"},
		"operations":     {"api", "signed"},
		"payments":       {"auth", "verified", "throttle:payments"},
		"analytics":      {"auth", "can:view-analytics"},
		"communications": {"auth", "throttle:messages"},
	}
}

// DefaultMatchStrategies returns the complete matching configuration
func DefaultMatchStrategies(registry *policy.Registry) *MatchStrategies {
	// Build URI patterns map (pattern -> zone_id)
	uriPatterns := make(map[string]int)
	for _, zk := range DefaultURIStrategies() {
		for _, keyword := range zk.Keywords {
			uriPatterns[keyword] = zk.ZoneID
		}
	}

	// Build controller patterns map
	controllerPatterns := make(map[string]int)
	zoneKeyMap := policy.ZoneKeyMap()
	for pattern, zoneKey := range DefaultControllerStrategies() {
		if zoneID, ok := zoneKeyMap[zoneKey]; ok {
			controllerPatterns[pattern] = zoneID
		}
	}

	return &MatchStrategies{
		URI:        URIStrategies{Patterns: uriPatterns},
		Controller: ControllerStrategies{Patterns: controllerPatterns},
		Middleware: MiddlewareStrategies{ZoneMiddleware: DefaultMiddlewareStrategies()},
	}
}

// Zone-specific keyword definitions

func getBoundaryKeywords() []string {
	return []string{
		"boundary", "lobby", "entry", "gateway", "unaligned", "unrecognized",
	}
}

func getPublicKeywords() []string {
	return []string{
		"public", "home", "welcome", "index", "list",
		"show", "view", "search", "browse", "explore", "pages",
	}
}

func getIdentityKeywords() []string {
	return []string{
		"login", "register", "logout", "auth", "signin",
		"signup", "verify", "confirm", "forgot", "reset",
		"password", "2fa", "mfa", "otp", "token",
		"key", "credential", "identity",
	}
}

func getOperationsKeywords() []string {
	return []string{
		"operations", "delegation", "delegated", "api", "uploads", "forms", "webhooks", "data",
		"input", "endpoint", "request", "response", "payload",
	}
}

func getProtectedKeywords() []string {
	return []string{
		"protected", "secure", "private", "restricted", "confidential",
		"sensitive", "manager", "owner", "user", "member",
		"group", "team", "account", "profile", "settings",
		"preferences", "config", "setup",
	}
}

func getProcessingKeywords() []string {
	return []string{
		"event", "jobs", "tasks", "cron", "schedule",
		"queue", "worker", "process", "daemon", "service",
		"system", "os", "platform", "environment", "server",
		"host", "machine",
	}
}

func getLoggingKeywords() []string {
	return []string{
		"event", "logs", "log", "history", "audit",
		"record", "trace", "track", "monitor", "alert",
		"notification", "message", "signal", "trigger",
	}
}

func getCommunicationsKeywords() []string {
	return []string{
		"comms", "communication", "message", "chat", "email",
		"sms", "notification", "alert", "signal", "call",
		"contact", "connect", "link", "network", "social", "media",
	}
}

func getPaymentsKeywords() []string {
	return []string{
		"payments", "payment", "transaction", "transfer", "invoice",
		"bill", "charge", "fee", "cost", "price",
		"amount", "currency", "wallet", "balance", "checkout",
		"cart", "order", "purchase", "sale", "buy",
		"sell", "trade", "exchange", "market", "bid", "offer",
	}
}

func getAnalyticsKeywords() []string {
	return []string{
		"analytics", "analysis", "report", "reporting", "metrics",
		"metric", "data", "insights", "insight", "dashboard",
		"visualization", "visualize", "chart", "graph", "table", "summary",
	}
}

func getStreamingKeywords() []string {
	return []string{
		"streaming", "stream", "streams", "livestream", "live",
		"video", "audio",
	}
}

func getSynchronizationKeywords() []string {
	return []string{
		"sync", "synchronization", "synchronize", "replication",
		"replicate", "replica", "peer", "peers",
	}
}

func getAdministrationKeywords() []string {
	return []string{
		"admin", "administrator", "superuser", "root", "owner",
		"manager", "control", "access", "privilege", "permission",
		"authority", "moderator",
	}
}

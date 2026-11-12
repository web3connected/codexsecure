package analyzer

// vocabulary.go — Canonical zone vocabulary registry
//
// Each zone owns a PRIMARY vocabulary: words that unambiguously speak its language.
// Words belong to exactly ONE zone — no shared words between zones.
//
// GOVERNING words: if a governing word appears as ANY segment (especially first),
// that zone is locked immediately. Child segments cannot override a governing parent.
// Rule: segments can only ESCALATE (Z1 → Z12 is valid); they never de-escalate.
//
// Segment hierarchy:
//   /admin/routes   → admin (Z12) governs → Z12 locked
//   /auth/profile   → auth (Z2) governs   → Z2 locked
//   /shop/checkout  → no governing word   → score both, checkout (Z8) wins
//   /routes         → no governing word   → "routes" in Z1 primary → Z1
//
// This replaces flat keyword matching in strategies.go for URI scoring only.
// Controller + middleware matching remain unchanged.

import "strings"

// ZoneVocabulary defines the canonical language of a zone.
type ZoneVocabulary struct {
	ZoneID   int
	ZoneKey  string
	// Primary — words that speak this zone's language.
	// Each word belongs to exactly one zone (no cross-zone duplicates).
	Primary []string
	// Governing — if ANY segment in the URI matches, the zone is locked immediately.
	// These are words whose presence guarantees the zone regardless of context.
	// Typically namespace words: admin, auth, api.
	Governing []string
}

// VocabularyResult is the output of ScoreURIWithVocabulary.
type VocabularyResult struct {
	ZoneID   int
	ZoneKey  string
	Score    int
	Governed bool   // true = a governing keyword locked this zone
	Segment  string // the segment that triggered the match
}

// DefaultVocabularies returns the canonical zone vocabulary registry.
// This is the authoritative mapping of language → zone.
func DefaultVocabularies() []ZoneVocabulary {
	return []ZoneVocabulary{
		{
			ZoneID:  0,
			ZoneKey: "boundary",
			// Entry boundary — not an operational zone. Quarantine is a separate
			// Stage 1 security decision, not something URI words assign.
			Primary: []string{
				"boundary", "lobby", "entry", "gateway", "unaligned", "unrecognized",
			},
			Governing: []string{},
		},
		{
			ZoneID:  1,
			ZoneKey: "public",
			// Generic top-level concepts — public information with no auth requirement.
			// Single-segment paths whose subject is public data land here.
			Primary: []string{
				"home", "about", "welcome", "index",
				"blog", "news", "posts", "articles",
				"docs", "documentation", "help", "faq", "support",
				"contact", "terms", "privacy", "legal",
				"pages", "content", "public",
				"routes", "zones", "map", "search", "browse", "explore",
				"shop", "store", "products", "catalogue", "catalog",
				"pricing", "plans",
				"getting", "started", "onboarding", "guide", "tutorial",
			},
			Governing: []string{}, // public words never govern — higher zones can override
		},
		{
			ZoneID:  2,
			ZoneKey: "identity",
			// Identity and authentication flows — the language of who you are.
			Primary: []string{
				"login", "logout", "signin", "signout",
				"register", "signup", "enroll",
				"verify", "verification", "confirm", "confirmation",
				"forgot", "reset", "recover", "recovery",
				"mfa", "2fa", "otp", "totp",
				"token", "credential", "credentials",
				"identity", "session",
			},
			// auth and register are governing — /auth/* is always Z2 territory
			Governing: []string{"auth", "login", "register", "signin"},
		},
		{
			ZoneID:  3,
			ZoneKey: "operations",
			// Operational commands and delegated control actions by representatives or services.
			Primary: []string{
				"operations", "delegation", "delegated",
				"api", "webhook", "webhooks", "callback", "callbacks",
				"endpoint", "endpoints", "payload", "request",
				"upload", "uploads", "import", "export",
				"feed",
			},
			// api and webhook are strong governing signals
			Governing: []string{"api", "webhook", "webhooks"},
		},
		{
			ZoneID:  4,
			ZoneKey: "protected",
			// Authenticated user space — personal data behind a login.
			Primary: []string{
				"dashboard", "account", "profile",
				"settings", "preferences", "notifications",
				"inbox", "activity", "history",
				"members", "membership",
			},
			Governing: []string{"dashboard"},
		},
		{
			ZoneID:  5,
			ZoneKey: "processing",
			// Execution and background processing — jobs, queues, scheduled tasks.
			Primary: []string{
				"jobs", "queue", "queues",
				"cron", "schedule", "scheduler",
				"worker", "workers", "daemon", "process",
				"health", "status", "ping", "heartbeat",
				"environment", "platform",
			},
			Governing: []string{},
		},
		{
			ZoneID:  6,
			ZoneKey: "logging",
			// Audit and event history — read-only access to what happened.
			Primary: []string{
				"logs", "log", "audit", "audits",
				"events", "event-log", "eventlog",
				"trace", "traces", "monitor",
				"alerts", "incidents",
			},
			Governing: []string{"audit", "logs"},
		},
		{
			ZoneID:  7,
			ZoneKey: "communications",
			// Communications — messages sent between people or systems.
			Primary: []string{
				"messages", "message", "chat", "chats",
				"email", "emails", "mail",
				"sms", "comms", "communications",
				"conversations", "conversation",
				"announcements", "broadcast",
			},
			Governing: []string{},
		},
		{
			ZoneID:  8,
			ZoneKey: "payments",
			// Financial transaction traffic.
			Primary: []string{
				"payment", "payments", "checkout",
				"invoice", "invoices", "billing",
				"subscription", "subscriptions",
				"transaction", "transactions",
				"wallet", "wallets", "balance",
				"order", "orders", "purchase",
				"cart", "refund", "refunds",
			},
			Governing: []string{"payment", "payments", "checkout", "billing"},
		},
		{
			ZoneID:  9,
			ZoneKey: "analytics",
			// Aggregated insights and reporting — observational data.
			Primary: []string{
				"analytics", "analysis",
				"reports", "report", "reporting",
				"metrics", "metric",
				"insights", "insight",
				"statistics", "stats",
				"charts", "graphs",
			},
			Governing: []string{"analytics"},
		},
		{
			ZoneID:  10,
			ZoneKey: "streaming",
			// Continuous audio, video, live broadcast and media-stream delivery.
			Primary: []string{
				"stream", "streams", "streaming",
				"livestream", "livestreams", "live",
				"video", "videos", "audio",
			},
			Governing: []string{},
		},
		{
			ZoneID:  11,
			ZoneKey: "synchronization",
			// Distributed state exchange, replication and peer coordination.
			Primary: []string{
				"sync", "synchronization", "synchronize",
				"replication", "replicate", "replica", "replicas",
				"peer", "peers",
			},
			Governing: []string{},
		},
		{
			ZoneID:  12,
			ZoneKey: "administration",
			// Administrative control — highest privilege, governs everything beneath it.
			Primary: []string{
				"admin", "administrator", "administrators",
				"manage", "management", "manager",
				"control", "controls", "panel",
				"permissions", "permission", "roles",
				"superuser", "root", "owner",
				"moderator", "moderators",
				"config", "configuration",
				"users", "user-management",
				"system-admin",
			},
			// admin and manage are the strongest governing keywords in the system
			Governing: []string{"admin", "manage", "administrator", "manage", "control-panel"},
		},
	}
}

// ScoreURIWithVocabulary is the primary URI zone scorer.
//
// Design principle: this function is an INFERENCE engine for routes that have no
// explicit zone tag declaration. When a route file declares zone: 'Z1_PUBLIC', that
// declaration is the source of truth — this scorer is not consulted.
// A public route never has the word "public" in its URL (/about, /home, /blog).
// Zone identity lives in the route tag, not the URL.
//
// Algorithm:
//  1. Walk segments left-to-right. If any segment CONTAINS a governing keyword
//     as a word-boundary token (split on - and _), lock that zone immediately.
//     Examples: /admin → "admin" governs Z12
//               /wp-admin → tokens ["wp","admin"] → "admin" governs Z12
//               /codex_admin → tokens ["codex","admin"] → "admin" governs Z12
//  2. If no governing keyword found, score all zones using primary vocabulary
//     with first-segment bonus.
//  3. Return the best-scoring zone.
//
// Escalation rule: governing words found later in the path still lock the zone,
// but first-segment governing words are preferred (score 600 vs 500).
func ScoreURIWithVocabulary(uri string, vocabs []ZoneVocabulary) VocabularyResult {
	trimmed := strings.ToLower(strings.Trim(uri, "/"))
	segments := []string{}
	if trimmed != "" {
		segments = strings.Split(trimmed, "/")
	}

	// Build governing lookup: word → vocabulary
	type govEntry struct {
		vocab *ZoneVocabulary
	}
	govMap := make(map[string]govEntry)
	for i := range vocabs {
		for _, g := range vocabs[i].Governing {
			word := strings.ToLower(g)
			if _, exists := govMap[word]; !exists { // first definition wins
				govMap[word] = govEntry{vocab: &vocabs[i]}
			}
		}
	}

	// segmentTokens splits a URL segment on - and _ so compound forms like
	// "wp-admin", "codex_admin", "super-admin" all yield the governing token "admin".
	segmentTokens := func(seg string) []string {
		return strings.FieldsFunc(seg, func(r rune) bool {
			return r == '-' || r == '_'
		})
	}

	// Walk segments — check each token within a segment for governing keywords.
	// First-segment governing match = higher confidence (600) vs later segment (500).
	var firstGov *VocabularyResult
	for idx, seg := range segments {
		seg = strings.ToLower(seg)
		tokens := segmentTokens(seg)
		for _, tok := range tokens {
			if entry, ok := govMap[tok]; ok {
				score := 500
				if idx == 0 {
					score = 600
				}
				result := VocabularyResult{
					ZoneID:   entry.vocab.ZoneID,
					ZoneKey:  entry.vocab.ZoneKey,
					Score:    score,
					Governed: true,
					Segment:  seg, // preserve original compound segment for diagnostics
				}
				if idx == 0 {
					// First-segment governing = definitive, stop immediately
					return result
				}
				// Later governing keyword — record it but keep walking
				if firstGov == nil {
					firstGov = &result
				}
				break // one governing token per segment is enough
			}
		}
	}
	if firstGov != nil {
		return *firstGov
	}

	// No governing keyword — score all zones by primary vocabulary
	best := VocabularyResult{ZoneID: UnassignedZoneID, ZoneKey: "unassigned"}
	for _, v := range vocabs {
		score := scorePrimary(segments, v.Primary)
		if score > best.Score {
			best = VocabularyResult{
				ZoneID:   v.ZoneID,
				ZoneKey:  v.ZoneKey,
				Score:    score,
				Governed: false,
			}
		}
	}
	return best
}

// scorePrimary calculates score for a URI against a zone's primary vocabulary.
// First-segment exact match = 200 + word length (high confidence).
// Child-segment exact match = 100 + word length (medium confidence).
// Substring fallback = 20 + word length (low confidence, advisory only).
func scorePrimary(segments []string, primary []string) int {
	best := 0
	for _, word := range primary {
		word = strings.ToLower(word)
		wordScore := 0
		for idx, seg := range segments {
			if seg == word {
				s := 100 + len(word)
				if idx == 0 {
					s = 200 + len(word)
				}
				if s > wordScore {
					wordScore = s
				}
			}
		}
		if wordScore == 0 {
			// Substring fallback — low confidence
			for _, seg := range segments {
				if strings.Contains(seg, word) {
					wordScore = 20 + len(word)
					break
				}
			}
		}
		if wordScore > best {
			best = wordScore
		}
	}
	return best
}

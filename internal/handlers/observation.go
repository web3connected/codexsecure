package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/database"
)

// ── Rate limiter ─────────────────────────────────────────────────────────────

// observationRateLimiter is an in-memory per-(path,zone) rate cap.
//
// Design:
//   - One entry per unique (path:zoneID) key.
//   - A request is allowed only if minGap has elapsed since the last allowed request.
//   - Session dedup (in SQL) handles same-session replays; this cap blocks
//     adversarial cycling of fresh session hashes at machine speed.
//   - Cleanup goroutine prunes entries older than 5 minutes to bound memory.
type observationRateLimiter struct {
	mu      sync.Mutex
	windows map[string]time.Time
	minGap  time.Duration
}

func newObservationRateLimiter(minGap time.Duration) *observationRateLimiter {
	rl := &observationRateLimiter{
		windows: make(map[string]time.Time),
		minGap:  minGap,
	}
	go rl.cleanup()
	return rl
}

// Allow returns true if the key is outside the rate window and records the
// current time. Returns false (without updating) when inside the window.
func (rl *observationRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if last, seen := rl.windows[key]; seen && time.Since(last) < rl.minGap {
		return false
	}
	rl.windows[key] = time.Now()
	return true
}

// cleanup removes stale entries every 5 minutes to prevent unbounded growth.
func (rl *observationRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-5 * time.Minute)
		rl.mu.Lock()
		for k, v := range rl.windows {
			if v.Before(cutoff) {
				delete(rl.windows, k)
			}
		}
		rl.mu.Unlock()
	}
}

// ── Weight decay ──────────────────────────────────────────────────────────────

// decayedWeight applies exponential decay to a stored weight based on
// how long ago the route was last observed.
//
// Formula:  effective = stored_weight × e^(-t × ln2 / halfLife)
//
// Half-life of 30 days means:
//
//	0 days  → effective = stored        (no decay)
//	30 days → effective = stored × 0.5
//	90 days → effective = stored × 0.125
//
// Prevents a route seen 100× months ago retaining near-certain weight (0.99)
// if the zone assignment has since changed.
func decayedWeight(stored float64, lastObservedAt time.Time) float64 {
	const halfLifeDays = 30.0
	daysSince := time.Since(lastObservedAt).Hours() / 24.0
	decay := math.Exp(-daysSince * math.Log(2) / halfLifeDays)
	return math.Round(stored*decay*1e6) / 1e6 // 6 dp — matches DB column precision
}

// ── Handler ───────────────────────────────────────────────────────────────────

// ObservationHandler persists route observations to route_observations.
//
// Three-layer protection against weight inflation:
//
//  1. Session dedup (SQL) — count increments only when session_hash changes.
//     Same session calling Record N times → count stays at 1.
//
//  2. Rate cap (in-memory) — minimum 2 s between observations for the same
//     (path, zone_id). Blocks adversarial session-hash cycling at machine speed.
//
//  3. Weight decay (read-time) — effective_weight applies exponential decay
//     (30-day half-life). Stale routes drift back toward 0 automatically.
type ObservationHandler struct {
	db          *database.DB
	rateLimiter *observationRateLimiter
}

// NewObservationHandler creates a new ObservationHandler with a 2-second
// per-(path, zone_id) rate cap.
func NewObservationHandler(db *database.DB) *ObservationHandler {
	return &ObservationHandler{
		db:          db,
		rateLimiter: newObservationRateLimiter(2 * time.Second),
	}
}

// observationRecordRequest is the JSON body for POST /api/v1/observations/record.
type observationRecordRequest struct {
	// Path is the canonical route path, e.g. "/auth/login".
	Path string `json:"path" binding:"required"`

	// ZoneID is the confirmed zone integer (1-12).
	ZoneID int `json:"zone_id" binding:"required,min=1,max=12"`

	// PhaseSource must be "phase-locked" or "route-override".
	// PHASE_CONFLICT and UNRESOLVED are rejected — they carry no valid signal.
	PhaseSource string `json:"phase_source" binding:"required"`

	// SessionHash is a stable, opaque identifier for the logical user session.
	// No PII — typically a hash of (session_token + path).
	// When omitted, every call is treated as a distinct session (no dedup).
	SessionHash string `json:"session_hash"`

	// SourcePortal identifies which portal sent this observation.
	// e.g. "codexauth", "codextime", "codexhash", "codexsecure".
	// Optional — stored as-is; used for per-portal analytics.
	SourcePortal string `json:"source_portal"`
}

// Record upserts a single route observation.
// POST /api/v1/observations/record
func (h *ObservationHandler) Record(c *gin.Context) {
	var req observationRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Normalise path
	path := strings.TrimSpace(req.Path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Only well-understood phase sources carry statistical signal.
	phaseSource := strings.ToLower(strings.TrimSpace(req.PhaseSource))
	if phaseSource != "phase-locked" && phaseSource != "route-override" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    "invalid_phase_source",
			"message":  "phase_source must be 'phase-locked' or 'route-override'; conflicts and unresolved events are not recorded",
			"received": req.PhaseSource,
		})
		return
	}

	// ── Layer 2: Rate cap ────────────────────────────────────────────────────
	// Minimum 2 s between any observation for the same (path, zone_id),
	// regardless of session hash. Blocks machine-speed hash cycling.
	rateKey := fmt.Sprintf("%s:%d", path, req.ZoneID)
	if !h.rateLimiter.Allow(rateKey) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"recorded":       false,
			"reason":         "rate_limited",
			"retry_after_ms": 2000,
			"message":        "Minimum 2 s between observations for the same (path, zone_id)",
		})
		return
	}

	// ── Layer 1: Session dedup (enforced in SQL) ──────────────────────────────
	// UpsertRouteObservation only increments count when session_hash changes.
	sessionHash  := strings.TrimSpace(req.SessionHash)
	sourcePortal := strings.TrimSpace(req.SourcePortal)

	obs, err := h.db.UpsertRouteObservation(c.Request.Context(), path, req.ZoneID, phaseSource, sessionHash, sourcePortal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": "Failed to record observation: " + err.Error(),
		})
		return
	}

	// ── Layer 3: Weight decay (read-time) ────────────────────────────────────
	// stored_weight = raw accumulated weight (what's in the DB).
	// effective_weight = stored × decay(days_since_last_seen).
	// The analyzer should use effective_weight for confidence scoring.
	effective := decayedWeight(obs.Weight, obs.LastObservedAt)

	c.JSON(http.StatusOK, gin.H{
		"recorded": true,
		"observation": gin.H{
			"id":                obs.ID,
			"path":              obs.Path,
			"zone_id":           obs.ZoneID,
			"phase_source":      obs.PhaseSource,
			"source_portal":     obs.SourcePortal,
			"observation_count": obs.ObservationCount,
			"stored_weight":     obs.Weight,
			"effective_weight":  effective,
			"last_observed_at":  obs.LastObservedAt,
		},
	})
}


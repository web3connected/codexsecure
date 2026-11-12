package middleware

// Chain - Middleware chain builder
// Builds and configures middleware chains for different route groups.
//
// @version 2.0.0

import (
	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

// ChainConfig configures the middleware chain
type ChainConfig struct {
	// Debug enables debug headers
	Debug bool

	// DebugConfig for debug middleware
	DebugConfig *DebugConfig

	// TIUConfig for TIU middleware
	TIUConfig *TIUConfig

	// AnalysisConfig for analysis middleware
	AnalysisConfig *AnalysisConfig

	// SkipZoneDetection disables zone detection middleware
	SkipZoneDetection bool

	// SkipMethodValidation disables method validation middleware
	SkipMethodValidation bool

	// SkipTIU disables TIU validation middleware
	SkipTIU bool

	// SkipAnalysis disables zone analysis middleware
	SkipAnalysis bool
}

// DefaultChainConfig returns sensible defaults
func DefaultChainConfig() *ChainConfig {
	return &ChainConfig{
		Debug:         false,
		DebugConfig:   DefaultDebugConfig(),
		TIUConfig:     DefaultTIUConfig(),
		AnalysisConfig: DefaultAnalysisConfig(),
	}
}

// Chain builds a middleware chain with all zone middleware
type Chain struct {
	analyzer *analyzer.Analyzer
	registry *policy.Registry
	fp       *fingerprint.Analyzer
	config   *ChainConfig
}

// Ordered middleware stages for zone-aware request processing.
const (
	StageRequestIdentity = "request_identity"
	StageZoneDetection   = "zone_detection"
	StageMethodValidation = "method_validation"
	StageTIUValidation   = "tiu_validation"
	StageAnalysisEnforcement = "analysis_enforcement"
	StageExecutionSync   = "execution_sync"
)

// NewChain creates a new middleware chain
func NewChain(
	a *analyzer.Analyzer,
	reg *policy.Registry,
	fp *fingerprint.Analyzer,
	cfg *ChainConfig,
) *Chain {
	if cfg == nil {
		cfg = DefaultChainConfig()
	}
	return &Chain{
		analyzer: a,
		registry: reg,
		fp:       fp,
		config:   cfg,
	}
}

// Build returns the full middleware chain
func (c *Chain) Build() []gin.HandlerFunc {
	var chain []gin.HandlerFunc

	// 1. Debug middleware (always first)
	if c.config.Debug {
		chain = append(chain, DebugMiddleware(c.config.DebugConfig))
	} else {
		// Always add request ID
		chain = append(chain, RequestIDMiddleware())
	}

	// 2. Zone detection (sets context for other middleware)
	if !c.config.SkipZoneDetection {
		chain = append(chain, ZoneMiddleware(c.analyzer, c.registry))
	}

	// 3. Method validation
	if !c.config.SkipMethodValidation {
		chain = append(chain, MethodMiddleware(c.registry))
	}

	// 4. TIU validation
	if !c.config.SkipTIU && c.fp != nil {
		chain = append(chain, TIUMiddleware(c.fp, c.config.TIUConfig))
	}

	// 5. Zone analysis enforcement
	if !c.config.SkipAnalysis {
		chain = append(chain, AnalysisMiddleware(c.config.AnalysisConfig))
	}

	return chain
}

// Order returns the semantic middleware stage order for the full zone-aware chain.
// The order matters because later middleware consumes request or zone context established earlier.
func (c *Chain) Order() []string {
	return []string{
		StageRequestIdentity,
		StageZoneDetection,
		StageMethodValidation,
		StageTIUValidation,
		StageAnalysisEnforcement,
		StageExecutionSync,
	}
}

// BuildWithExecutionSync returns the full zone-aware chain with Stage 7 execution sync appended last.
func (c *Chain) BuildWithExecutionSync(appVersion string) []gin.HandlerFunc {
	chain := append([]gin.HandlerFunc{}, c.Build()...)
	chain = append(chain, ExecutionSyncMiddleware(appVersion))
	return chain
}

// Apply applies the middleware chain to a router group
func (c *Chain) Apply(rg *gin.RouterGroup) {
	for _, mw := range c.Build() {
		rg.Use(mw)
	}
}

// PublicChain builds a minimal chain for public routes
func (c *Chain) PublicChain() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestIDMiddleware(),
		ZoneMiddleware(c.analyzer, c.registry),
	}
}

// AuthenticatedChain builds a chain for authenticated routes
func (c *Chain) AuthenticatedChain() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestIDMiddleware(),
		ZoneMiddleware(c.analyzer, c.registry),
		MethodMiddleware(c.registry),
		RequireAuth(),
	}
}

// CriticalChain builds a chain for critical/admin routes
func (c *Chain) CriticalChain() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		DebugMiddleware(c.config.DebugConfig),
		ZoneMiddleware(c.analyzer, c.registry),
		MethodMiddleware(c.registry),
		RequireAuth(),
		RequireExecutive(),
		TIUMiddleware(c.fp, c.config.TIUConfig),
	}
}

// ApplyToGroup applies a specific chain preset to a group
type ChainPreset string

const (
	PresetPublic        ChainPreset = "public"
	PresetAuthenticated ChainPreset = "authenticated"
	PresetCritical      ChainPreset = "critical"
	PresetFull          ChainPreset = "full"
)

// ApplyPreset applies a preset chain to a router group
func (c *Chain) ApplyPreset(rg *gin.RouterGroup, preset ChainPreset) {
	var chain []gin.HandlerFunc

	switch preset {
	case PresetPublic:
		chain = c.PublicChain()
	case PresetAuthenticated:
		chain = c.AuthenticatedChain()
	case PresetCritical:
		chain = c.CriticalChain()
	case PresetFull:
		chain = c.Build()
	default:
		chain = c.Build()
	}

	for _, mw := range chain {
		rg.Use(mw)
	}
}

// ForZone creates middleware chain for a specific zone
func (c *Chain) ForZone(zoneID int) []gin.HandlerFunc {
	zone, ok := c.registry.GetZone(zoneID)
	if !ok {
		return c.PublicChain()
	}

	// Build chain based on zone tier
	switch zone.Tier {
	case policy.TierPublic:
		return c.PublicChain()
	case policy.TierBusiness:
		return c.AuthenticatedChain()
	case policy.TierCritical:
		return c.CriticalChain()
	default:
		return c.Build()
	}
}

// UseAll applies all middleware to a router
func UseAll(
	r *gin.Engine,
	a *analyzer.Analyzer,
	reg *policy.Registry,
	fp *fingerprint.Analyzer,
	cfg *ChainConfig,
) {
	chain := NewChain(a, reg, fp, cfg)
	for _, mw := range chain.Build() {
		r.Use(mw)
	}
}

// UsePublic applies public middleware to a router group
func UsePublic(rg *gin.RouterGroup, a *analyzer.Analyzer, reg *policy.Registry) {
	rg.Use(RequestIDMiddleware())
	rg.Use(ZoneMiddleware(a, reg))
}

// UseAuthenticated applies authenticated middleware to a router group
func UseAuthenticated(rg *gin.RouterGroup, a *analyzer.Analyzer, reg *policy.Registry) {
	rg.Use(RequestIDMiddleware())
	rg.Use(ZoneMiddleware(a, reg))
	rg.Use(MethodMiddleware(reg))
	rg.Use(RequireAuth())
}

// UseCritical applies critical middleware to a router group
func UseCritical(rg *gin.RouterGroup, a *analyzer.Analyzer, reg *policy.Registry, fp *fingerprint.Analyzer) {
	rg.Use(DebugMiddleware(nil))
	rg.Use(ZoneMiddleware(a, reg))
	rg.Use(MethodMiddleware(reg))
	rg.Use(RequireAuth())
	rg.Use(RequireExecutive())
	rg.Use(TIUMiddleware(fp, nil))
}

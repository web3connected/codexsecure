package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/internal/server/routes/zones/z2"
	"github.com/web3connected/codexsecure-api/internal/server/routes/zones/z5"
)

// LoadAPIRoutes is the Go equivalent of Laravel routes/api.php.
// It wires API endpoints plus zone-scoped route modules.
func LoadAPIRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	v1.Use(middleware.RequestIDMiddleware())

	if deps.CommunicationHandler != nil {
		deps.CommunicationHandler.RegisterRoutes(v1)
	}
	if deps.ChannelHandler != nil {
		deps.ChannelHandler.RegisterRoutes(v1)
	}
	// Harmonic Alignment recording (Step 3B): gateway assertion + service key, as the channel routes.
	if deps.AlignmentHandler != nil {
		deps.AlignmentHandler.RegisterRoutes(v1)
	}
	// Phase Lock pattern storage (P3): the same gateway assertion + service key contract.
	if deps.PhaseLockStorageHandler != nil {
		deps.PhaseLockStorageHandler.RegisterRoutes(v1)
	}

	v1.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":        "codexsecure",
			"version":        deps.AppVersion,
			"policy_version": deps.PolicyVersion,
			"status":         "operational",
		})
	})

	// Zone route modules must register before /zones/:code.
	zoneChain := middleware.NewChain(deps.RouteAnalyzer, deps.PolicyRegistry, deps.FingerprintAnalyzer, nil)
	z2Group := v1.Group("/zones/z2")
	z2Group.Use(middleware.PhaseLockMiddleware(deps.FingerprintAnalyzer, deps.SchemaRegistry, middleware.DefaultPhaseLockConfig()))
	applyMiddlewareChain(z2Group, zoneChain.BuildWithExecutionSync(deps.AppVersion))
	z2.Register(z2Group)

	z5Group := v1.Group("/zones/z5")
	applyMiddlewareChain(z5Group, zoneChain.BuildWithExecutionSync(deps.AppVersion))
	z5.Register(z5Group)

	// Zone endpoints — service key required for detect/lookup calls.
	// Health and listing remain open for monitoring.
	v1.GET("/zones/health", deps.ZoneHandler.ZoneHealth)
	authenticatedZones := v1.Group("/zones")
	authenticatedZones.Use(middleware.ServiceKeyMiddleware())
	{
		authenticatedZones.GET("", deps.ZoneHandler.ListZones)
		authenticatedZones.POST("/detect", deps.ZoneHandler.DetectZone)
		authenticatedZones.GET("/:code", deps.ZoneHandler.GetZone)
	}

	policyGroup := v1.Group("/policy")
	{
		policyGroup.GET("/snapshot", deps.PolicyHandler.GetSnapshot)
		policyGroup.GET("/zones", deps.PolicyHandler.ListZonePolicies)
		policyGroup.POST("/validate-method", deps.PolicyHandler.ValidateMethod)
		policyGroup.GET("/zones/:idOrKey", deps.PolicyHandler.GetZonePolicy)
		policyGroup.GET("/zones/:idOrKey/requirements", deps.PolicyHandler.GetZoneRequirements)
		policyGroup.GET("/registry", deps.PolicyHandler.GetRegistry)
		policyGroup.POST("/alignment", deps.PolicyHandler.CheckAlignment)
		policyGroup.GET("/definitions", deps.PolicyHandler.GetDefinitions)
		policyGroup.POST("/alignment/definitions", deps.PolicyHandler.CheckDefinitionAlignment)
	}

	analyzerGroup := v1.Group("/analyzer")
	{
		analyzerGroup.GET("/detect", deps.AnalyzerHandler.DetectZone)
		analyzerGroup.GET("/strategies", deps.AnalyzerHandler.GetStrategies)
		analyzerGroup.POST("/route", deps.AnalyzerHandler.AnalyzeSingleRoute)
		analyzerGroup.POST("/routes", deps.AnalyzerHandler.AnalyzeRoutes)
		analyzerGroup.POST("/stats", deps.AnalyzerHandler.GetAnalysisStats)
		analyzerGroup.POST("/check-zone", deps.AnalyzerHandler.CheckRouteInZone)
	}

	fingerprintGroup := v1.Group("/fingerprint")
	{
		fingerprintGroup.GET("/config", deps.FingerprintHandler.GetConfig)
		fingerprintGroup.POST("/analyze", deps.FingerprintHandler.Analyze)
		fingerprintGroup.POST("/route", deps.FingerprintHandler.GenerateRouteFingerprint)
		fingerprintGroup.POST("/input", deps.FingerprintHandler.GenerateInputFingerprint)
		fingerprintGroup.POST("/drift", deps.FingerprintHandler.CalculateDrift)
		fingerprintGroup.POST("/filter", deps.FingerprintHandler.FilterInput)
		fingerprintGroup.POST("/hash", deps.FingerprintHandler.QuickHash)
		fingerprintGroup.POST("/batch", deps.FingerprintHandler.BatchAnalyze)
	}

	phaseLockGroup := v1.Group("/phase-lock")
	{
		if deps.PhaseLockHandler != nil {
			phaseLockGroup.POST("/record", deps.PhaseLockHandler.Record)
		}
	}

	observationGroup := v1.Group("/observations")
	{
		if deps.ObservationHandler != nil {
			observationGroup.POST("/record", deps.ObservationHandler.Record)
		}
	}

	// Stage 4 (Protect) services for the Laravel SDK. Authenticated at the gateway (X-API-Key).
	protectGroup := v1.Group("/protect")
	{
		if deps.ProtectHandler != nil {
			protectGroup.POST("/codexhash", deps.ProtectHandler.GenerateCodexHash)
			// I01: the Seven-Fold Send Hash and validatedHash, through the proprietary CodexHash v2 engine.
			protectGroup.POST("/codexhash/v2", deps.ProtectHandler.GenerateCodexHashV2)
			protectGroup.POST("/tiu", deps.ProtectHandler.IssueTIU)
		}
	}

	// Stage 5 (Record) for GET PageMeta. Authenticated at the gateway (X-API-Key).
	pageMetaGroup := v1.Group("/pagemeta")
	{
		if deps.PageMetaHandler != nil {
			pageMetaGroup.POST("/record", deps.PageMetaHandler.Record)
		}
	}

	pipelineGroup := v1.Group("/pipeline")
	{
		pipelineGroup.POST("/analyze", deps.PipelineHandler.AnalyzeExecution)
		pipelineGroup.POST("/executions", deps.PipelineHandler.CreateExecution)
	}
}

func applyMiddlewareChain(group *gin.RouterGroup, chain []gin.HandlerFunc) {
	for _, handler := range chain {
		group.Use(handler)
	}
}

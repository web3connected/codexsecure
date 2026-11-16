package routes

import (
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/handlers"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/internal/schema"
)

// Dependencies contains the handler set used by route modules.
type Dependencies struct {
	AppVersion    string
	PolicyVersion string

	RouteAnalyzer       *analyzer.Analyzer
	FingerprintAnalyzer *fingerprint.Analyzer
	PolicyRegistry      *policy.Registry
	SchemaRegistry      *schema.Registry

	HealthHandler  *handlers.HealthHandler
	MetricsHandler *handlers.MetricsHandler

	ZoneHandler             *handlers.ZoneHandler
	PolicyHandler           *handlers.PolicyHandler
	AnalyzerHandler         *handlers.AnalyzerHandler
	FingerprintHandler      *handlers.FingerprintHandler
	PhaseLockHandler        *handlers.PhaseLockHandler
	PipelineHandler         *handlers.PipelineHandler
	ObservationHandler      *handlers.ObservationHandler
	ProtectHandler          *handlers.ProtectHandler
	PageMetaHandler         *handlers.PageMetaHandler
	CommunicationHandler    *handlers.CommunicationHandler
	ChannelHandler          *handlers.ChannelHandler
	AlignmentHandler        *handlers.AlignmentHandler
	PhaseLockStorageHandler *handlers.PhaseLockStorageHandler
}

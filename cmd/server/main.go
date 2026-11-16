package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/channels"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/internal/config"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/handlers"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/internal/schema"
	"github.com/web3connected/codexsecure-api/internal/server/routes"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
	"github.com/web3connected/codexsecure-api/pkg/mcp"
)

// Version information
const (
	AppName    = "CodexSecure API"
	AppVersion = "2.0.0"
)

func main() {
	// Setup logging
	setupLogging()

	log.Info().
		Str("app", AppName).
		Str("version", AppVersion).
		Msg("Starting CodexSecure API")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	log.Info().Msg("Configuration loaded successfully")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Test database connection
	if err := db.TestConnection(ctx); err != nil {
		log.Fatal().Err(err).Msg("Database connection test failed")
	}

	// Get database info
	version, err := db.GetDatabaseVersion(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get database version")
	} else {
		log.Info().Str("version", version).Msg("Connected to PostgreSQL")
	}

	// Initialize MCP client
	mcpClient := mcp.NewClient(cfg.MCP.NodeServerURL, cfg.MCP.Enabled)
	if cfg.MCP.Enabled {
		log.Info().Str("url", cfg.MCP.NodeServerURL).Msg("MCP integration enabled")
		// Test MCP connection (non-fatal if it fails)
		if err := mcpClient.Health(ctx); err != nil {
			log.Warn().Err(err).Msg("MCP server health check failed - continuing without MCP")
		} else {
			log.Info().Msg("MCP server connection healthy")
		}
	} else {
		log.Info().Msg("MCP integration disabled")
	}

	// Initialize policy registry with defaults
	policyRegistry := policy.NewRegistry()

	// Validate policy on startup (fail fast if invalid)
	if err := policy.MustValidate(policyRegistry.GetSnapshot()); err != nil {
		log.Fatal().Err(err).Msg("Policy validation failed")
	}
	log.Info().
		Str("version", policy.Version).
		Int("zones", len(policyRegistry.AllZones())).
		Msg("Policy registry initialized")

	// Initialize route analyzer
	routeAnalyzer := analyzer.NewAnalyzer(policyRegistry)
	log.Info().Msg("Route analyzer initialized")

	var schemaRegistry *schema.Registry

	if cfg.Schema.Enabled {
		artifact, err := schema.LoadArtifact(cfg.Schema.ArtifactPath)
		if err != nil {
			if cfg.Schema.RequireValid {
				log.Fatal().Err(err).Str("path", cfg.Schema.ArtifactPath).Msg("Failed to load page schema artifact")
			}
			log.Warn().Err(err).Str("path", cfg.Schema.ArtifactPath).Msg("Page schema artifact unavailable or invalid")
		} else {
			schemaRegistry = schema.NewRegistry(artifact)
			summary := artifact.Summary()
			log.Info().
				Str("schema_set_id", summary.SchemaSetID).
				Str("application", summary.ApplicationName).
				Str("environment", summary.Environment).
				Int("records", summary.RecordCount).
				Str("baseline_state", summary.BaselineState).
				Str("trust_mode", summary.TrustMode).
				Bool("phase_lock_enabled", summary.PhaseLock).
				Msg("Page schema artifact loaded")
		}
	} else {
		log.Info().Msg("Page schema artifact loading disabled")
	}

	// Initialize fingerprint analyzer
	fingerprintAnalyzer := fingerprint.NewAnalyzer(nil)
	log.Info().Msg("Fingerprint analyzer initialized")

	// Start HTTP server
	if err := startHTTPServer(ctx, cfg, db, mcpClient, policyRegistry, routeAnalyzer, fingerprintAnalyzer, schemaRegistry); err != nil {
		log.Fatal().Err(err).Msg("HTTP server failed")
	}

	log.Info().
		Str("host", cfg.Server.Host).
		Str("port", cfg.Server.Port).
		Msg("CodexSecure API initialized")
	log.Info().Msg("Press Ctrl+C to shutdown")

	// Close database on exit
	db.Close()
	log.Info().Msg("CodexSecure API stopped")
}

// setupLogging configures zerolog
func setupLogging() {
	// Pretty logging for development
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	// Set log level
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// startHTTPServer initializes and starts the Gin HTTP server
func startHTTPServer(ctx context.Context, cfg *config.Config, db *database.DB, mcpClient *mcp.Client, policyRegistry *policy.Registry, routeAnalyzer *analyzer.Analyzer, fingerprintAnalyzer *fingerprint.Analyzer, schemaRegistry *schema.Registry) error {
	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS — allow web3connected dev ports and production domains
	allowedOrigins := map[string]bool{
		"http://localhost:3007":               true,
		"http://localhost:3010":               true,
		"http://localhost:3920":               true,
		"https://web3connected.com":           true,
		"https://codextime.web3connected.com": true,
		"https://codexhash.web3connected.com": true,
	}
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Api-Key, X-Codex-Zone")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Setup service tracker
	tracker := middleware.NewServiceTracker(db)
	r.Use(tracker.TrackRequest())

	// Setup handlers
	healthHandler := handlers.NewHealthHandler(db, policyRegistry)
	metricsHandler := handlers.NewMetricsHandler(tracker)
	policyHandler := handlers.NewPolicyHandler(policyRegistry)
	analyzerHandler := handlers.NewAnalyzerHandler(routeAnalyzer)
	fingerprintHandler := handlers.NewFingerprintHandler(fingerprintAnalyzer)
	phaseLockHandler := handlers.NewPhaseLockHandler(routeAnalyzer, fingerprintAnalyzer, schemaRegistry)
	observationHandler := handlers.NewObservationHandler(db)
	codexHashClient := codexhash.NewClient()
	protectHandler := handlers.NewProtectHandler(codexHashClient, codextime.NewClient())
	pageMetaHandler := handlers.NewPageMetaHandler(codexHashClient, db)
	pipelineHandler := handlers.NewPipelineHandler(
		db,
		cfg.Pipeline.APIToken,
		mcpClient,
		routeAnalyzer,
		fingerprintAnalyzer,
		policyRegistry,
		schemaRegistry,
	)

	communicationZones := map[string]bool{}
	for id := range policyRegistry.GetSnapshot().Zones {
		communicationZones[fmt.Sprintf("Z%d", id)] = true
	}
	communicationHandler := handlers.NewCommunicationHandler(communications.NewPostgresStore(db.Pool, communicationZones).WithSnapshots(codexHashClient, communications.SnapshotTTLFromEnv()))
	channelHandler := handlers.NewChannelHandler(channels.NewManager(db.Pool, codexHashClient, codextime.NewClient()))
	// Harmonic Alignment recording (Step 3B): harmonic_zones through the runtime role.
	alignmentHandler := handlers.NewAlignmentHandler(db)
	// Phase Lock pattern storage (P3): CodexHash v2 verification, P2 tables through the runtime role.
	phaseLockStorageHandler := handlers.NewPhaseLockStorageHandler(codexHashClient, db)

	deps := routes.Dependencies{
		CommunicationHandler:    communicationHandler,
		ChannelHandler:          channelHandler,
		AlignmentHandler:        alignmentHandler,
		PhaseLockStorageHandler: phaseLockStorageHandler,
		AppVersion:              AppVersion,
		PolicyVersion:           policy.Version,
		RouteAnalyzer:           routeAnalyzer,
		FingerprintAnalyzer:     fingerprintAnalyzer,
		PolicyRegistry:          policyRegistry,
		SchemaRegistry:          schemaRegistry,
		HealthHandler:           healthHandler,
		MetricsHandler:          metricsHandler,
		PolicyHandler:           policyHandler,
		AnalyzerHandler:         analyzerHandler,
		FingerprintHandler:      fingerprintHandler,
		PhaseLockHandler:        phaseLockHandler,
		PipelineHandler:         pipelineHandler,
		ObservationHandler:      observationHandler,
		ProtectHandler:          protectHandler,
		PageMetaHandler:         pageMetaHandler,
	}

	// Web routes are opt-in for this API service.
	if os.Getenv("ENABLE_WEB_ROUTES") == "true" {
		routes.LoadWebRoutes(r, deps)
	}

	// API routes (Laravel-style routes/api.php equivalent)
	v1 := r.Group("/api/v1")
	routes.LoadAPIRoutes(v1, deps)

	// Create server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
		// Prevent the server from appearing to "lock up" on stalled/slow clients.
		// These defaults are intentionally conservative for a JSON API.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down HTTP server")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	return srv.Shutdown(shutdownCtx)
}

package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/config"
)

// DB holds the CodexZone database connection pool (codex_zones).
// The schema is owned by Web3Codex; this service never creates or migrates it.
type DB struct {
	Pool *pgxpool.Pool
	// Database and Role are the server-reported current_database() and current_user.
	Database string
	Role     string
}

// connectTimeout bounds each new physical connection, not only the startup ping.
const connectTimeout = 5 * time.Second

// PoolConfig builds the pool configuration. Values are quoted for the libpq
// keyword/value format; the password is set on the parsed configuration rather
// than placed in a connection string.
func PoolConfig(cfg *config.DatabaseConfig) (*pgxpool.Config, error) {
	if _, err := strconv.ParseUint(cfg.Port, 10, 16); err != nil {
		return nil, fmt.Errorf("invalid database port %q", cfg.Port)
	}

	poolConfig, err := pgxpool.ParseConfig(fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d",
		quoteConnValue(cfg.Host), quoteConnValue(cfg.Port), quoteConnValue(cfg.Name),
		quoteConnValue(cfg.User), quoteConnValue(cfg.SSLMode), cfg.MaxConns, cfg.MinConns,
	))
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.ConnConfig.Password = cfg.Password
	poolConfig.ConnConfig.ConnectTimeout = connectTimeout
	return poolConfig, nil
}

// quoteConnValue quotes a value for the libpq keyword/value connection format.
func quoteConnValue(value string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value) + "'"
}

// Connect establishes the connection pool and verifies that the server placed
// it in the configured database.
func Connect(cfg *config.DatabaseConfig) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := PoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	db := &DB{Pool: pool}
	if err := pool.QueryRow(ctx, "SELECT current_database(), current_user").Scan(&db.Database, &db.Role); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to read database identity: %w", err)
	}
	if db.Database != cfg.Name {
		pool.Close()
		return nil, fmt.Errorf("connected to database %q, expected %q", db.Database, cfg.Name)
	}

	log.Info().
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("database", db.Database).
		Str("role", db.Role).
		Int("max_conns", cfg.MaxConns).
		Msg("Connected to CodexZone database")

	return db, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Info().Msg("Database connection closed")
	}
}

// Health checks database health
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns database pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// GetDatabaseVersion returns PostgreSQL version
func (db *DB) GetDatabaseVersion(ctx context.Context) (string, error) {
	var version string
	err := db.Pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get database version: %w", err)
	}
	return version, nil
}

// TestConnection performs comprehensive connection test
func (db *DB) TestConnection(ctx context.Context) error {
	// Test basic connectivity
	if err := db.Health(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Test query execution
	var result int
	if err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("query test failed: %w", err)
	}

	return nil
}

// RouteObservation is a single upserted row from route_observations.
type RouteObservation struct {
	ID               int64     `json:"id"`
	Path             string    `json:"path"`
	ZoneID           int       `json:"zone_id"`
	PhaseSource      string    `json:"phase_source"`
	SourcePortal     string    `json:"source_portal"`
	LastSessionHash  string    `json:"last_session_hash"`
	ObservationCount int64     `json:"observation_count"`
	Weight           float64   `json:"weight"`
	LastObservedAt   time.Time `json:"last_observed_at"`
}

// UpsertRouteObservation inserts a new route observation or conditionally
// increments the counter when the session hash changes.
//
// Session deduplication — the count only increments when the incoming
// sessionHash differs from the stored last_session_hash. Repeated calls
// from the same session are no-ops for count and weight, preventing
// raw-call-count inflation (the "forced high weight" attack vector).
//
// Weight formula (only applied on genuine new sessions):
//
//	weight = 1 - (1 / new_observation_count)
//
// Both weight and count are computed purely in SQL — no read-then-write.
func (db *DB) UpsertRouteObservation(ctx context.Context, path string, zoneID int, phaseSource, sessionHash, sourcePortal string) (*RouteObservation, error) {
	var obs RouteObservation
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO route_observations
		       (path, zone_id, phase_source, last_session_hash, observation_count, weight, last_observed_at, created_at, updated_at, source_portal)
		VALUES ($1,   $2,      $3,           $4,                1,                 0.0,    NOW(),            NOW(),      NOW(),      $5)
		ON CONFLICT (path, zone_id) DO UPDATE
		SET observation_count = CASE
		      WHEN route_observations.last_session_hash IS DISTINCT FROM EXCLUDED.last_session_hash
		        THEN route_observations.observation_count + 1
		      ELSE route_observations.observation_count
		    END,
		    weight = CASE
		      WHEN route_observations.last_session_hash IS DISTINCT FROM EXCLUDED.last_session_hash
		        THEN 1.0 - (1.0 / (route_observations.observation_count + 1)::float)
		      ELSE route_observations.weight
		    END,
		    phase_source      = EXCLUDED.phase_source,
		    last_session_hash = EXCLUDED.last_session_hash,
		    source_portal     = EXCLUDED.source_portal,
		    last_observed_at  = NOW(),
		    updated_at        = NOW()
		RETURNING id, path, zone_id, phase_source, source_portal, COALESCE(last_session_hash, ''), observation_count, weight, last_observed_at
	`, path, zoneID, phaseSource, sessionHash, sourcePortal).Scan(
		&obs.ID, &obs.Path, &obs.ZoneID, &obs.PhaseSource,
		&obs.SourcePortal, &obs.LastSessionHash, &obs.ObservationCount, &obs.Weight, &obs.LastObservedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert route observation: %w", err)
	}
	return &obs, nil
}

package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Zone     ZoneConfig
	Pattern  PatternConfig
	MCP      MCPConfig
	Schema   SchemaConfig
	Pipeline PipelineConfig
	Log      LogConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port string
	Host string
	Env  string
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
	MaxConns int
	MinConns int
}

// ZoneConfig holds zone system configuration
type ZoneConfig struct {
	RouterID     string
	Count        int
	FractalDepth int
}

// PatternConfig holds pattern engine configuration
type PatternConfig struct {
	CacheSize     int
	HashAlgorithm string
}

// MCPConfig holds MCP integration configuration
type MCPConfig struct {
	Enabled       bool
	NodeServerURL string
}

// SchemaConfig holds page schema artifact loading configuration.
type SchemaConfig struct {
	Enabled      bool
	ArtifactPath string
	RequireValid bool
}

// PipelineConfig holds Stage 7 recording endpoint configuration.
type PipelineConfig struct {
	APIToken string
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8083"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "172.104.24.82"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnvAny([]string{"DB_NAME", "DB_DATABASE"}, "codex_zones"),
			User:     getEnvAny([]string{"DB_USER", "DB_USERNAME"}, ""),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "require"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Zone: ZoneConfig{
			RouterID:     getEnv("ZONE_ROUTER_ID", "Z0"),
			Count:        getEnvInt("ZONE_COUNT", 12),
			FractalDepth: getEnvInt("ZONE_FRACTAL_DEPTH", 12),
		},
		Pattern: PatternConfig{
			CacheSize:     getEnvInt("PATTERN_CACHE_SIZE", 10000),
			HashAlgorithm: getEnv("PATTERN_HASH_ALGORITHM", "sha256"),
		},
		MCP: MCPConfig{
			Enabled:       getEnvBool("MCP_ENABLED", true),
			NodeServerURL: getEnv("MCP_NODE_SERVER_URL", "http://localhost:3000"),
		},
		Schema: SchemaConfig{
			Enabled:      getEnvBool("PAGE_SCHEMA_ENABLED", false),
			ArtifactPath: getEnv("PAGE_SCHEMA_ARTIFACT_PATH", ""),
			RequireValid: getEnvBool("PAGE_SCHEMA_REQUIRE_VALID", true),
		},
		Pipeline: PipelineConfig{
			APIToken: getEnv("PIPELINE_API_TOKEN", getEnv("CODEX_STAGE7_API_TOKEN", "")),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate required fields
	if cfg.Database.User == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Schema.Enabled && cfg.Schema.ArtifactPath == "" {
		return nil, fmt.Errorf("PAGE_SCHEMA_ARTIFACT_PATH is required when PAGE_SCHEMA_ENABLED=true")
	}

	return cfg, nil
}

// DSN returns PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Name, c.User, c.Password, c.SSLMode,
	)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAny gets the first non-empty environment variable from a list.
func getEnvAny(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return fallback
}

// getEnvInt gets environment variable as int with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

// getEnvBool gets environment variable as bool with fallback
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

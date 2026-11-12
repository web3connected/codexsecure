package database

import (
	"strings"
	"testing"
	"time"

	"github.com/web3connected/codexsecure-api/internal/config"
)

const awkwardPassword = `p a'ss\word=x 'y'`

func testConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Host: "host.docker.internal", Port: "5432", Name: "codex_zones",
		User: "codex_zones_runtime", Password: awkwardPassword, SSLMode: "prefer",
		MaxConns: 25, MinConns: 5,
	}
}

func TestPoolConfigSetsCodexZoneConnection(t *testing.T) {
	poolConfig, err := PoolConfig(testConfig())
	if err != nil {
		t.Fatalf("PoolConfig: %v", err)
	}

	c := poolConfig.ConnConfig
	if c.Host != "host.docker.internal" || c.Port != 5432 || c.Database != "codex_zones" || c.User != "codex_zones_runtime" {
		t.Errorf("connection = %s:%d/%s as %s", c.Host, c.Port, c.Database, c.User)
	}
	if c.Password != awkwardPassword {
		t.Error("password with spaces, quotes and backslashes must be preserved exactly")
	}
	if c.ConnectTimeout != 5*time.Second {
		t.Errorf("connect timeout = %s, want 5s", c.ConnectTimeout)
	}
	if poolConfig.MaxConns != 25 || poolConfig.MinConns != 5 {
		t.Errorf("pool = %d/%d, want 25/5", poolConfig.MaxConns, poolConfig.MinConns)
	}
	for _, fallback := range c.Fallbacks {
		if fallback.Host != "host.docker.internal" || fallback.Port != 5432 {
			t.Errorf("fallback targets %s:%d", fallback.Host, fallback.Port)
		}
	}
}

func TestPoolConfigRejectsInvalidPortWithoutLeakingPassword(t *testing.T) {
	cfg := testConfig()
	cfg.Port = "not-a-port"
	_, err := PoolConfig(cfg)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), awkwardPassword) {
		t.Errorf("error exposes the password: %v", err)
	}
}

func TestConnectFailureDoesNotExposePassword(t *testing.T) {
	cfg := testConfig()
	cfg.Host, cfg.Port = "127.0.0.1", "1" // nothing listens here

	started := time.Now()
	db, err := Connect(cfg)
	if err == nil {
		db.Close()
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), awkwardPassword) || strings.Contains(err.Error(), "word=x") {
		t.Errorf("error exposes the password: %v", err)
	}
	if time.Since(started) > 12*time.Second {
		t.Errorf("connection failure took %s", time.Since(started))
	}
}

func TestQuoteConnValue(t *testing.T) {
	if got := quoteConnValue(`a'b\c`); got != `'a\'b\\c'` {
		t.Errorf("quoteConnValue = %s", got)
	}
}

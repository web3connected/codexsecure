//go:build integration

// Live, read-only verification of the CodexZone runtime connection.
// Uses only DB_* runtime settings; never writes data or changes the schema.
//
//	go test -tags integration ./internal/database/ -run CodexZone
package database

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/web3connected/codexsecure-api/internal/config"
)

func runtimeConfig(t *testing.T) *config.DatabaseConfig {
	t.Helper()
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD"} {
		if os.Getenv(key) == "" {
			t.Skipf("%s not set", key)
		}
	}
	maxConns, _ := strconv.Atoi(getenvDefault("DB_MAX_CONNS", "5"))
	return &config.DatabaseConfig{
		Host: os.Getenv("DB_HOST"), Port: os.Getenv("DB_PORT"), Name: os.Getenv("DB_NAME"),
		User: os.Getenv("DB_USER"), Password: os.Getenv("DB_PASSWORD"),
		SSLMode: getenvDefault("DB_SSL_MODE", "prefer"), MaxConns: maxConns, MinConns: 1,
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestCodexZoneRuntimeConnection(t *testing.T) {
	cfg := runtimeConfig(t)
	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := func(sql string, args ...any) (rows [][]any) {
		t.Helper()
		result, err := db.Pool.Query(ctx, sql, args...)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		defer result.Close()
		for result.Next() {
			values, err := result.Values()
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, values)
		}
		return rows
	}

	t.Run("A database identity", func(t *testing.T) {
		if db.Database != "codex_zones" {
			t.Errorf("current_database() = %q", db.Database)
		}
		t.Logf("current_database=%s", db.Database)
	})

	t.Run("B runtime role", func(t *testing.T) {
		if db.Role != "codex_zones_runtime" {
			t.Errorf("current_user = %q", db.Role)
		}
		role := query(`SELECT rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls FROM pg_roles WHERE rolname = current_user`)[0]
		for i, name := range []string{"superuser", "createdb", "createrole", "replication", "bypassrls"} {
			if role[i].(bool) {
				t.Errorf("runtime role has %s", name)
			}
		}
		ssl := query(`SELECT COALESCE((SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid()), false)`)[0][0]
		t.Logf("current_user=%s ssl=%v", db.Role, ssl)
	})

	t.Run("C PageMeta schema", func(t *testing.T) {
		columns := query(`SELECT column_name, udt_name, is_nullable, COALESCE(column_default, ''),
		                         COALESCE(character_maximum_length, 0), COALESCE(numeric_precision, 0),
		                         COALESCE(numeric_scale, 0), COALESCE(datetime_precision, 0)
		                  FROM information_schema.columns
		                  WHERE table_schema = 'public' AND table_name = 'pagemeta_packages'
		                  ORDER BY ordinal_position`)
		var got []string
		for _, c := range columns {
			got = append(got, strings.Join([]string{
				c[0].(string), c[1].(string), c[2].(string), c[3].(string),
				strconv.Itoa(int(c[4].(int32))), strconv.Itoa(int(c[5].(int32))), strconv.Itoa(int(c[6].(int32))), strconv.Itoa(int(c[7].(int32))),
			}, "|"))
		}
		want := []string{
			"uuid|uuid|NO|gen_random_uuid()|0|0|0|0",
			"payload|jsonb|NO||0|0|0|0",
			"hash|varchar|NO||128|0|0|0",
			"tiu|numeric|NO||0|20|6|0",
			"created_at|timestamptz|NO|CURRENT_TIMESTAMP|0|0|0|6",
			"updated_at|timestamptz|NO|CURRENT_TIMESTAMP|0|0|0|6",
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("columns differ from the approved migration\n got: %v\nwant: %v", got, want)
		}

		constraints := map[string]string{}
		for _, c := range query(`SELECT conname, pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = 'public.pagemeta_packages'::regclass`) {
			constraints[c[0].(string)] = c[1].(string)
		}
		expect := map[string]string{
			"pagemeta_packages_pkey":                 "PRIMARY KEY (uuid)",
			"pagemeta_packages_hash_sha512_check":    "^[0-9a-f]{128}$",
			"pagemeta_packages_tiu_positive_check":   "tiu > (0)::numeric",
			"pagemeta_packages_payload_object_check": "jsonb_typeof(payload) = 'object'",
		}
		for name, fragment := range expect {
			if !strings.Contains(constraints[name], fragment) {
				t.Errorf("constraint %s = %q, want it to contain %q", name, constraints[name], fragment)
			}
		}
		if len(constraints) != len(expect) {
			t.Errorf("unexpected constraints: %v", constraints)
		}

		indexes := query(`SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'pagemeta_packages' ORDER BY 1`)
		if len(indexes) != 2 || indexes[0][0] != "pagemeta_packages_hash_index" || indexes[1][0] != "pagemeta_packages_pkey" {
			t.Errorf("indexes = %v", indexes)
		}
	})

	t.Run("D runtime permissions", func(t *testing.T) {
		checks := []struct {
			sql  string
			want bool
		}{
			{`SELECT has_table_privilege('public.pagemeta_packages', 'SELECT')`, true},
			{`SELECT has_table_privilege('public.pagemeta_packages', 'INSERT')`, true},
			{`SELECT has_table_privilege('public.pagemeta_packages', 'UPDATE')`, true},
			{`SELECT has_table_privilege('public.pagemeta_packages', 'DELETE')`, false},
			{`SELECT has_table_privilege('public.pagemeta_packages', 'TRUNCATE')`, false},
			{`SELECT has_schema_privilege('public', 'CREATE')`, false},
			{`SELECT has_database_privilege(current_database(), 'CREATE')`, false},
			{`SELECT has_database_privilege('web3codex', 'CONNECT')`, false},
			{`SELECT COALESCE((SELECT has_table_privilege('public.migrations', 'SELECT') FROM pg_tables WHERE schemaname = 'public' AND tablename = 'migrations'), false)`, false},
			{`SELECT COALESCE((SELECT has_table_privilege('public.migrations', 'INSERT') FROM pg_tables WHERE schemaname = 'public' AND tablename = 'migrations'), false)`, false},
			{`SELECT (SELECT tableowner FROM pg_tables WHERE schemaname = 'public' AND tablename = 'pagemeta_packages') <> current_user`, true},
		}
		for _, check := range checks {
			if got := query(check.sql)[0][0].(bool); got != check.want {
				t.Errorf("%s = %v, want %v", check.sql, got, check.want)
			}
		}

		// Provisioning deliberately preserved the legacy codex_main PUBLIC ACL.
		// Reported, not enforced here: database permissions belong to Web3Codex.
		if query(`SELECT has_database_privilege('codex_main', 'CONNECT')`)[0][0].(bool) {
			t.Log("NOTE: runtime role can CONNECT to legacy codex_main via PUBLIC (no table access verified separately)")
		}
	})

	t.Run("E context-aware queries and lifecycle", func(t *testing.T) {
		short, cancelShort := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancelShort()
		if _, err := db.Pool.Exec(short, "SELECT pg_sleep(1)"); err == nil {
			t.Error("query must honour context cancellation")
		}
		if err := db.Health(ctx); err != nil {
			t.Errorf("health after cancellation: %v", err)
		}
		if stat := db.Stats(); stat.MaxConns() != int32(cfg.MaxConns) {
			t.Errorf("pool max conns = %d", stat.MaxConns())
		}
	})
}

func TestCodexZoneWrongPasswordIsReportedWithoutSecret(t *testing.T) {
	cfg := runtimeConfig(t)
	cfg.Password = "definitely-wrong-runtime-password"
	db, err := Connect(cfg)
	if err == nil {
		db.Close()
		t.Fatal("expected authentication failure")
	}
	if strings.Contains(err.Error(), cfg.Password) {
		t.Errorf("error exposes the password: %v", err)
	}
	t.Logf("error: %v", err)
}

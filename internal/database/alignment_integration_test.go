//go:build integration

// Harmonic Alignment recording against PostgreSQL (CodexSecure Step 3B).
//
// Concurrency needs committed writes from several connections, so this test never uses
// public.harmonic_zones: it creates a throwaway schema holding copies of pagemeta_packages
// and harmonic_zones (same columns, constraints and unique index, its own id sequence),
// points search_path at it, exercises RecordAlignment's SQL, and drops the schema.
// It needs a role that may CREATE in the database (ALIGNMENT_IT_DB_USER / _PASSWORD,
// with DB_HOST, DB_PORT, DB_NAME); without them it is skipped.
package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	alignmentITUUID  = "0b3c5d7e-1111-4222-8333-944455556666"
	alignmentITOther = "0b3c5d7e-1111-4222-8333-944455557777"
	alignmentITHash  = "ch2:inv-dual/1:512:" + "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" +
		"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
)

func alignmentScratch(t *testing.T) (*DB, *pgxpool.Pool, string) {
	t.Helper()
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_NAME", "ALIGNMENT_IT_DB_USER", "ALIGNMENT_IT_DB_PASSWORD"} {
		if os.Getenv(key) == "" {
			t.Skipf("%s not set", key)
		}
	}
	ctx := context.Background()
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	schema := "codexsecure_it_" + hex.EncodeToString(suffix)
	dsn := (&url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("ALIGNMENT_IT_DB_USER"), os.Getenv("ALIGNMENT_IT_DB_PASSWORD")),
		Host: os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"), Path: "/" + os.Getenv("DB_NAME"),
		RawQuery: "sslmode=" + getenvDefault("DB_SSL_MODE", "prefer")}).String()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	for _, statement := range []string{
		fmt.Sprintf(`CREATE SCHEMA %s`, schema),
		fmt.Sprintf(`CREATE TABLE %s.pagemeta_packages (LIKE public.pagemeta_packages INCLUDING ALL EXCLUDING DEFAULTS)`, schema),
		fmt.Sprintf(`CREATE TABLE %s.harmonic_zones (LIKE public.harmonic_zones INCLUDING ALL EXCLUDING DEFAULTS)`, schema),
		fmt.Sprintf(`CREATE SEQUENCE %s.harmonic_zones_id_seq OWNED BY %s.harmonic_zones.id`, schema, schema),
		fmt.Sprintf(`ALTER TABLE %s.harmonic_zones ALTER COLUMN id SET DEFAULT nextval('%s.harmonic_zones_id_seq')`, schema, schema),
	} {
		if _, err := admin.Exec(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema)); err != nil {
			t.Errorf("drop %s: %v", schema, err)
		}
	})
	for uuid, route := range map[string]string{alignmentITUUID: "it.home", alignmentITOther: "it.other"} {
		payload := fmt.Sprintf(`{"data":{"method":"GET","declared_zone":"Z1","intent":"%s","route":{"name":"%s","uri":"it","controller":"It\\Controller","action":"show"}}}`, route, route)
		if _, err := admin.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.pagemeta_packages (uuid, payload, hash, tiu, created_at, updated_at)
			VALUES ($1, $2::jsonb, $3, 216832585.431537, now(), now())`, schema), uuid, payload, alignmentITHash); err != nil {
			t.Fatal(err)
		}
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 16
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return &DB{Pool: pool}, admin, schema
}

func alignmentITObservation() AlignmentObservation {
	route, intent, controller, action := "it.home", "it.home", "It\\Controller", "show"
	return AlignmentObservation{PageMetaUUID: alignmentITUUID, Zone: "Z1", RouteName: &route, HTTPMethod: "GET",
		CodexHash: alignmentITHash, TIU: 216832585.431537, Intent: &intent, Controller: &controller, Action: &action}
}

func alignmentITRows(t *testing.T, admin *pgxpool.Pool, schema string) int {
	t.Helper()
	var rows int
	if err := admin.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s.harmonic_zones`, schema)).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestAlignmentRecordingAgainstPostgres(t *testing.T) {
	db, admin, schema := alignmentScratch(t)
	ctx := context.Background()

	// Refusals write nothing.
	unknown := alignmentITObservation()
	unknown.PageMetaUUID = "0b3c5d7e-1111-4222-8333-000000000000"
	if _, err := db.RecordAlignment(ctx, unknown); !errors.Is(err, ErrAlignmentPageMetaNotFound) {
		t.Fatalf("unknown uuid: %v", err)
	}
	for field, change := range map[string]func(*AlignmentObservation){
		"relationship.zone":           func(o *AlignmentObservation) { o.Zone = "Z2" },
		"relationship.route_name":     func(o *AlignmentObservation) { other := "it.other"; o.RouteName = &other },
		"relationship.http_method":    func(o *AlignmentObservation) { o.HTTPMethod = "POST" },
		"developer_intent.controller": func(o *AlignmentObservation) { other := "Other\\Controller"; o.Controller = &other },
	} {
		obs := alignmentITObservation()
		change(&obs)
		var mismatch *AlignmentMismatch
		if _, err := db.RecordAlignment(ctx, obs); !errors.As(err, &mismatch) || mismatch.Field != field {
			t.Fatalf("%s: %v", field, err)
		}
	}
	tampered := alignmentITObservation()
	tampered.CodexHash = alignmentITHash[:len(alignmentITHash)-1] + "0"
	if _, err := db.RecordAlignment(ctx, tampered); !errors.Is(err, ErrAlignmentEvidenceMismatch) {
		t.Fatalf("evidence mismatch: %v", err)
	}
	if rows := alignmentITRows(t, admin, schema); rows != 0 {
		t.Fatalf("refused observations wrote %d rows", rows)
	}

	// First observation creates the relationship; the next increments it.
	first, err := db.RecordAlignment(ctx, alignmentITObservation())
	if err != nil || !first.Created || first.AlignmentCount != 1 || first.MisalignmentCount != 0 || first.Result != AlignmentAligned {
		t.Fatalf("first: %+v, %v", first, err)
	}
	later := alignmentITObservation()
	later.TIU = 216832590.5
	second, err := db.RecordAlignment(ctx, later)
	if err != nil || second.Created || second.ID != first.ID || second.AlignmentCount != 2 || second.LastTIU != "216832590.500000" {
		t.Fatalf("second: %+v, %v", second, err)
	}
	// An older TIU never moves last_tiu backwards.
	if third, err := db.RecordAlignment(ctx, alignmentITObservation()); err != nil || third.AlignmentCount != 3 || third.LastTIU != "216832590.500000" {
		t.Fatalf("third: %+v, %v", third, err)
	}

	// Concurrent observations: one row, no lost increment.
	const workers = 40
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.RecordAlignment(ctx, alignmentITObservation()); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent observation: %v", err)
	}
	var rows, count, misalignment int64
	var alignmentWeight, misalignmentWeight *string
	if err := admin.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) OVER (), alignment_count, misalignment_count,
		alignment_weight::text, misalignment_weight::text FROM %s.harmonic_zones WHERE pagemeta_uuid = $1`, schema), alignmentITUUID).
		Scan(&rows, &count, &misalignment, &alignmentWeight, &misalignmentWeight); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || count != 3+workers || misalignment != 0 || alignmentWeight != nil || misalignmentWeight != nil {
		t.Fatalf("after concurrency: rows %d, alignment %d, misalignment %d, weights %v %v", rows, count, misalignment, alignmentWeight, misalignmentWeight)
	}

	// A second relationship in the same zone is its own row.
	other := alignmentITObservation()
	otherRoute, otherIntent := "it.other", "it.other"
	other.PageMetaUUID, other.RouteName, other.Intent = alignmentITOther, &otherRoute, &otherIntent
	if record, err := db.RecordAlignment(ctx, other); err != nil || !record.Created || record.AlignmentCount != 1 {
		t.Fatalf("second relationship: %+v, %v", record, err)
	}
	if rows := alignmentITRows(t, admin, schema); rows != 2 {
		t.Fatalf("two relationships in Z1 expected, got %d rows", rows)
	}

	// A stored row that disagrees with PageMeta is never incremented.
	if _, err := admin.Exec(ctx, fmt.Sprintf(`UPDATE %s.harmonic_zones SET route_name = 'it.changed' WHERE pagemeta_uuid = $1`, schema), alignmentITUUID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordAlignment(ctx, alignmentITObservation()); !errors.Is(err, ErrAlignmentRelationshipConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if err := admin.QueryRow(ctx, fmt.Sprintf(`SELECT alignment_count FROM %s.harmonic_zones WHERE pagemeta_uuid = $1`, schema), alignmentITUUID).Scan(&count); err != nil || count != 3+workers {
		t.Fatalf("conflicting row changed: %d, %v", count, err)
	}
}

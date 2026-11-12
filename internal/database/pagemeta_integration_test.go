//go:build integration

// Live PageMeta recording tests against codex_zones as the runtime role.
// Every write happens inside a transaction that is rolled back; the runtime
// role has no DELETE and no rows are left behind.
package database

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func testPage(t *testing.T) (string, func(view string) PageMetaRecord) {
	t.Helper()
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	uri := "/__codexzone_it_" + hex.EncodeToString(suffix)

	return uri, func(view string) PageMetaRecord {
		data := map[string]any{
			"method": "GET", "route": map[string]any{"name": "it.page", "uri": uri, "methods": []string{"GET", "HEAD"}},
			"declared_zone": "Z1", "view": view,
		}
		raw, _ := json.Marshal(data)
		sum := sha512.Sum512(raw)
		payload, _ := json.Marshal(map[string]any{"package_uuid": nil, "data": data, "security": map[string]any{"codexhash": hex.EncodeToString(sum[:]), "tiu": 1}})
		return PageMetaRecord{Method: "GET", URI: uri, Hash: hex.EncodeToString(sum[:]), TIU: "214000000.123456", Payload: payload}
	}
}

func TestPageMetaRecognitionLifecycle(t *testing.T) {
	db, err := Connect(runtimeConfig(t))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	uri, record := testPage(t)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	first, err := RecordPageMetaTx(ctx, tx, record("v1"))
	if err != nil || first.Result != PageMetaRecorded || !first.Recorded || first.Package == nil {
		t.Fatalf("first: %+v, %v", first, err)
	}
	if first.Package.TIU != "214000000.123456" {
		t.Errorf("tiu stored as %s", first.Package.TIU)
	}

	second, err := RecordPageMetaTx(ctx, tx, record("v1"))
	if err != nil || second.Result != PageMetaKnown || second.Recorded {
		t.Fatalf("repeat: %+v, %v", second, err)
	}
	if second.Package.UUID != first.Package.UUID || second.Package.Hash != first.Package.Hash {
		t.Errorf("repeat identity %s/%s, want %s/%s", second.Package.UUID, second.Package.Hash, first.Package.UUID, first.Package.Hash)
	}

	changed, err := RecordPageMetaTx(ctx, tx, record("v2"))
	if err != nil || changed.Result != PageMetaConfigurationChanged || changed.Recorded || changed.Package != nil {
		t.Fatalf("changed: %+v, %v", changed, err)
	}
	if len(changed.KnownPackages) != 1 || changed.KnownPackages[0].UUID != first.Package.UUID {
		t.Errorf("changed known packages = %+v", changed.KnownPackages)
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pagemeta_packages WHERE payload->'data'->'route'->>'uri' = $1`, uri).Scan(&count); err != nil || count != 1 {
		t.Errorf("rows for test page inside transaction = %d (%v), want 1", count, err)
	}

	// Only pagemeta_packages was written in this transaction.
	rows, err := tx.Query(ctx, `SELECT relname, n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_xact_user_tables WHERE n_tup_ins + n_tup_upd + n_tup_del > 0`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		var ins, upd, del int64
		_ = rows.Scan(&name, &ins, &upd, &del)
		if name != "pagemeta_packages" || ins != 1 || upd != 0 || del != 0 {
			t.Errorf("unexpected write: %s ins=%d upd=%d del=%d", name, ins, upd, del)
		}
	}
	rows.Close()

	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pagemeta_packages WHERE payload->'data'->'route'->>'uri' = $1`, uri).Scan(&count); err != nil || count != 0 {
		t.Errorf("rows left after rollback = %d (%v)", count, err)
	}
}

// Two submissions for the same page are serialized by the page advisory lock:
// the second waits until the first transaction ends, so it cannot insert a duplicate.
func TestPageMetaConcurrentSubmissionsAreSerialized(t *testing.T) {
	db, err := Connect(runtimeConfig(t))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	_, record := testPage(t)

	tx1, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := RecordPageMetaTx(ctx, tx1, record("v1")); err != nil || result.Result != PageMetaRecorded {
		t.Fatalf("tx1: %+v, %v", result, err)
	}

	done := make(chan PageMetaResult, 1)
	go func() {
		tx2, err := db.Pool.Begin(ctx)
		if err != nil {
			done <- PageMetaResult{Result: "error: " + err.Error()}
			return
		}
		defer tx2.Rollback(ctx)
		result, err := RecordPageMetaTx(ctx, tx2, record("v1"))
		if err != nil {
			result.Result = "error: " + err.Error()
		}
		done <- result
	}()

	select {
	case result := <-done:
		t.Fatalf("second submission did not wait for the first: %+v", result)
	case <-time.After(400 * time.Millisecond):
	}

	if err := tx1.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		// tx1 rolled back, so tx2 sees no row and records; it is then rolled back too.
		if result.Result != PageMetaRecorded {
			t.Errorf("second submission after release = %+v", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second submission never completed")
	}
}

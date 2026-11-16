//go:build integration

// Phase Lock pattern storage against PostgreSQL (CodexSecure P3).
//
// Runs only against a disposable cluster that carries the real P2 schema and grants,
// applied by Laravel's migrations (database/migrations/codex_zones/zones/2026_11_15_*):
// the cluster must report the disposable startup marker (codex.phase2b_nonce), so the
// test can never run against a persistent codex_zones database. The store connects as
// codex_zones_runtime, exactly as the Go service does; the admin connection only reads.
//
//	PHASELOCK_IT_RUNTIME_DSN  runtime role DSN (required; else skipped)
//	PHASELOCK_IT_ADMIN_DSN    a role that may read every table (required)
//	PHASELOCK_IT_ENGINE_FIXTURE  optional JSON {"data": {...}, "codexhash": "ch2:…"} produced by
//	    Laravel's PayloadProtection::protectStructure() through the real engine; with
//	    CODEXHASH_API_URL, Go's recomputation is compared with it
package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

const phaseLockITNonce = "phase2b-isolated-20261020"

func phaseLockITHash(seed byte) string {
	return "ch2:inv-dual/1:512:" + strings.Repeat(fmt.Sprintf("%02x", seed), 64)
}

func phaseLockIT(t *testing.T) (*DB, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	runtimeDSN, adminDSN := os.Getenv("PHASELOCK_IT_RUNTIME_DSN"), os.Getenv("PHASELOCK_IT_ADMIN_DSN")
	if runtimeDSN == "" || adminDSN == "" {
		t.Skip("PHASELOCK_IT_RUNTIME_DSN and PHASELOCK_IT_ADMIN_DSN not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var nonce *string
	if err := admin.QueryRow(ctx, `SELECT current_setting('codex.phase2b_nonce', true)`).Scan(&nonce); err != nil || nonce == nil || *nonce != phaseLockITNonce {
		t.Fatalf("refusing to run: not the disposable cluster (%v)", err)
	}
	config, err := pgxpool.ParseConfig(runtimeDSN)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 24
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var role string
	if err := pool.QueryRow(ctx, `SELECT current_user`).Scan(&role); err != nil || role != "codex_zones_runtime" {
		t.Fatalf("store must connect as codex_zones_runtime, got %q (%v)", role, err)
	}
	return &DB{Pool: pool}, pool, admin
}

// phaseLockITDocument is a FormStructure::document() for route; extra adds a field.
func phaseLockITDocument(route, extra string) string {
	fields := `[{"controls":["email"],"name":"email","part":"form","presence":"always","required":true}` +
		`,{"controls":["textarea"],"name":"message","part":"form","presence":"always","required":false}`
	if extra != "" {
		fields += `,{"controls":["text"],"name":"` + extra + `","part":"form","presence":"always","required":false}`
	}
	return `{"method":"POST","destination":{"zone":"Z1","route":"` + route + `","uri":"it/contact","controller":"It\\ContactController","action":"store"},"fields":` + fields + `]}`
}

func phaseLockITRecord(t *testing.T, rel PhaseLockRelationship, document, hash, tiu string) FormStructureRecord {
	t.Helper()
	canonical, err := protect.CanonicalPayload(json.RawMessage(document))
	if err != nil {
		t.Fatal(err)
	}
	return FormStructureRecord{Relationship: rel, Canonical: canonical, CodexHash: hash, TIU: tiu}
}

func phaseLockITRow(t *testing.T, admin *pgxpool.Pool, table string, id int64) string {
	t.Helper()
	var row string
	if err := admin.QueryRow(context.Background(), fmt.Sprintf(`SELECT row_to_json(t)::text FROM public.%s t WHERE id = $1`, table), id).Scan(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

func phaseLockITCount(t *testing.T, admin *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := admin.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPhaseLockStorageAgainstPostgres(t *testing.T) {
	db, runtime, admin := phaseLockIT(t)
	ctx := context.Background()
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	route := "it.p3." + hex.EncodeToString(suffix)
	rel := PhaseLockRelationship{Zone: "Z1", Route: route, Method: "POST"}
	hashA, hashB, hashC := phaseLockITHash(0xa1), phaseLockITHash(0xb2), phaseLockITHash(0xc3)
	eventsBefore := phaseLockITCount(t, admin, `SELECT count(*) FROM phase_lock_pattern_events`)

	// 1. A new structure is recorded; the stored jsonb reproduces the canonical hash input.
	recordA := phaseLockITRecord(t, rel, phaseLockITDocument(route, ""), hashA, "216832585.431537")
	result, pkgA, err := db.RecordFormStructure(ctx, recordA)
	if err != nil || result != PhaseLockRecorded {
		t.Fatalf("record A: %s %v", result, err)
	}
	if string(pkgA.Data) != recordA.Canonical || pkgA.TIU != "216832585.431537" {
		t.Fatalf("stored package A %s %s", pkgA.Data, pkgA.TIU)
	}
	rowA := phaseLockITRow(t, admin, "form_structure_packages", pkgA.ID)
	for _, forbidden := range []string{"visitor", "password", "session", "user_agent", "value"} {
		if strings.Contains(rowA, `"`+forbidden) {
			t.Fatalf("package row carries %s: %s", forbidden, rowA)
		}
	}

	// 2. The identical structure is known: same package, creation TIU kept, nothing written.
	result, again, err := db.RecordFormStructure(ctx, phaseLockITRecord(t, rel, phaseLockITDocument(route, ""), hashA, "216832999.000001"))
	if err != nil || result != PhaseLockKnown || again.ID != pkgA.ID || again.TIU != pkgA.TIU {
		t.Fatalf("repeat A: %s %+v %v", result, again, err)
	}

	// 3. A changed structure is its own package; A is unchanged.
	result, pkgB, err := db.RecordFormStructure(ctx, phaseLockITRecord(t, rel, phaseLockITDocument(route, "phone"), hashB, "216832600.5"))
	if err != nil || result != PhaseLockRecorded || pkgB.ID == pkgA.ID {
		t.Fatalf("record B: %s %v", result, err)
	}
	if phaseLockITRow(t, admin, "form_structure_packages", pkgA.ID) != rowA {
		t.Fatal("recording B changed package A")
	}
	if n := phaseLockITCount(t, admin, `SELECT count(*) FROM form_structure_packages WHERE route_name = $1`, route); n != 2 {
		t.Fatalf("packages %d", n)
	}

	// 4. A stored package that does not reproduce the verified structure is a conflict: the same
	// hash reported for a different document never passes as known.
	if _, _, err := db.RecordFormStructure(ctx, phaseLockITRecord(t, rel, phaseLockITDocument(route, "other"), hashA, "216832585.5")); !errors.Is(err, ErrPhaseLockStructureConflict) {
		t.Fatalf("conflicting structure under A's hash: %v", err)
	}

	// 5. The schema refuses what the handler refuses (second line of defence).
	for name, document := range map[string]string{
		"value":    strings.Replace(phaseLockITDocument(route, ""), `"name":"email",`, `"name":"email","value":"v@example.com",`, 1),
		"number":   strings.Replace(phaseLockITDocument(route, ""), `"required":true`, `"required":1`, 1),
		"relation": strings.Replace(phaseLockITDocument(route, ""), `"zone":"Z1"`, `"zone":"Z2"`, 1),
	} {
		record := phaseLockITRecord(t, rel, document, phaseLockITHash(0xee), "1")
		if _, _, err := db.RecordFormStructure(ctx, record); !errors.Is(err, ErrPhaseLockStructureRejected) {
			t.Fatalf("%s: %v", name, err)
		}
	}

	// 6. The relationship lists both packages, no pattern yet.
	entries, err := db.RelationshipPatterns(ctx, rel)
	if err != nil || len(entries) != 2 || entries[0].Package.CodexHash != hashA || entries[1].Package.CodexHash != hashB ||
		entries[0].Pattern != nil || entries[1].Pattern != nil || string(entries[0].Package.Data) != recordA.Canonical {
		t.Fatalf("patterns before discovery: %+v %v", entries, err)
	}

	// 7. Discovery: Developer → pending; Communications → no stored state; repeats unchanged.
	result, patternA, err := db.DiscoverPattern(ctx, rel, hashA, PhaseLockDeveloperAdmin)
	if err != nil || result != PhaseLockDiscovered || patternA.State == nil || *patternA.State != "pending" || patternA.Authority != PhaseLockDeveloperAdmin {
		t.Fatalf("discover A: %s %+v %v", result, patternA, err)
	}
	result, patternB, err := db.DiscoverPattern(ctx, rel, hashB, PhaseLockCommunicationObject)
	if err != nil || result != PhaseLockDiscovered || patternB.State != nil || patternB.Authority != PhaseLockCommunicationObject {
		t.Fatalf("discover B: %s %+v %v", result, patternB, err)
	}
	rowPatternA := phaseLockITRow(t, admin, "phase_lock_patterns", patternA.ID)
	result, existing, err := db.DiscoverPattern(ctx, rel, hashA, PhaseLockCommunicationObject)
	if err != nil || result != PhaseLockExisting || existing.Authority != PhaseLockDeveloperAdmin || *existing.State != "pending" {
		t.Fatalf("rediscover A under Communications: %s %+v %v", result, existing, err)
	}
	if phaseLockITRow(t, admin, "phase_lock_patterns", patternA.ID) != rowPatternA {
		t.Fatal("rediscovery changed pattern A")
	}
	if result, pattern, err := db.DiscoverPattern(ctx, rel, phaseLockITHash(0xdd), PhaseLockDeveloperAdmin); err != nil || result != PhaseLockNotFound || pattern != nil {
		t.Fatalf("unrecorded structure: %s %v", result, err)
	}
	other := PhaseLockRelationship{Zone: "Z1", Route: route + ".other", Method: "POST"}
	if result, _, err := db.DiscoverPattern(ctx, other, hashA, PhaseLockDeveloperAdmin); err != nil || result != PhaseLockNotFound {
		t.Fatalf("A under another relationship: %s %v", result, err)
	}
	if _, _, err := db.DiscoverPattern(ctx, rel, hashA, "enterprise"); !errors.Is(err, ErrPhaseLockAuthorityInvalid) {
		t.Fatalf("invalid authority: %v", err)
	}

	// 8. Lookup and listing reflect both patterns.
	if found, err := db.Pattern(ctx, rel, hashB); err != nil || found == nil || found.State != nil || found.ID != patternB.ID {
		t.Fatalf("lookup B: %+v %v", found, err)
	}
	if found, err := db.Pattern(ctx, rel, phaseLockITHash(0xdd)); err != nil || found != nil {
		t.Fatalf("lookup unknown: %+v %v", found, err)
	}
	entries, err = db.RelationshipPatterns(ctx, rel)
	if err != nil || len(entries) != 2 || entries[0].Pattern == nil || entries[1].Pattern == nil ||
		*entries[0].Pattern.State != "pending" || entries[1].Pattern.State != nil {
		t.Fatalf("patterns after discovery: %+v %v", entries, err)
	}

	// 9. Concurrent duplicates: one package and one pattern, whatever the interleaving.
	recordC := phaseLockITRecord(t, rel, phaseLockITDocument(route, "company"), hashC, "216832700")
	const workers = 24
	var wg sync.WaitGroup
	results := make(chan string, workers*2)
	failures := make(chan error, workers*2)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, _, err := db.RecordFormStructure(ctx, recordC)
			if err != nil {
				failures <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		authority := PhaseLockDeveloperAdmin
		if i%2 == 1 {
			authority = PhaseLockCommunicationObject
		}
		go func() {
			defer wg.Done()
			result, _, err := db.DiscoverPattern(ctx, rel, hashC, authority)
			if err != nil {
				failures <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent operation failed: %v", err)
	}
	counts := map[string]int{}
	for result := range results {
		counts[result]++
	}
	if counts[PhaseLockRecorded] != 1 || counts[PhaseLockKnown] != workers-1 || counts[PhaseLockDiscovered] != 1 || counts[PhaseLockExisting] != workers-1 {
		t.Fatalf("concurrent outcomes %v", counts)
	}
	if n := phaseLockITCount(t, admin, `SELECT count(*) FROM form_structure_packages WHERE route_name = $1 AND structure_codexhash = $2`, route, hashC); n != 1 {
		t.Fatalf("packages for C: %d", n)
	}
	if n := phaseLockITCount(t, admin, `SELECT count(*) FROM phase_lock_patterns WHERE route_name = $1 AND structure_codexhash = $2`, route, hashC); n != 1 {
		t.Fatalf("patterns for C: %d", n)
	}

	// 10. The runtime role cannot decide, audit, rewrite or delete.
	for name, sql := range map[string]string{
		"approve":          `UPDATE phase_lock_patterns SET state = 'approved', decided_by = 'x', decided_at = now() WHERE id = ` + fmt.Sprint(patternA.ID),
		"rewrite pattern":  `UPDATE phase_lock_patterns SET authority = 'communication_object' WHERE id = ` + fmt.Sprint(patternA.ID),
		"delete pattern":   `DELETE FROM phase_lock_patterns WHERE id = ` + fmt.Sprint(patternA.ID),
		"rewrite package":  `UPDATE form_structure_packages SET tiu = 1 WHERE id = ` + fmt.Sprint(pkgA.ID),
		"delete package":   `DELETE FROM form_structure_packages WHERE id = ` + fmt.Sprint(pkgA.ID),
		"truncate package": `TRUNCATE form_structure_packages CASCADE`,
		"insert event": `INSERT INTO phase_lock_pattern_events (pattern_id, structure_codexhash, decision, previous_state, resulting_state, actor, verification, idempotency_key)
			VALUES (` + fmt.Sprint(patternA.ID) + `, '` + hashA + `', 'approve', 'pending', 'approved', 'x', 'codexhash', gen_random_uuid())`,
		"insert approved": `INSERT INTO phase_lock_patterns (package_id, zone, route_name, http_method, structure_codexhash, authority, state)
			VALUES (` + fmt.Sprint(pkgB.ID) + `, 'Z1', '` + route + `', 'POST', '` + hashB + `', 'developer_admin', 'approved')`,
	} {
		_, err := runtime.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Fatalf("%s: want 42501, got %v", name, err)
		}
	}
	if phaseLockITRow(t, admin, "phase_lock_patterns", patternA.ID) != rowPatternA || phaseLockITRow(t, admin, "form_structure_packages", pkgA.ID) != rowA {
		t.Fatal("a refused statement changed stored data")
	}

	// 11. Nothing was approved or audited.
	if n := phaseLockITCount(t, admin, `SELECT count(*) FROM phase_lock_patterns WHERE route_name = $1 AND state IN ('approved', 'rejected')`, route); n != 0 {
		t.Fatalf("decided patterns: %d", n)
	}
	if n := phaseLockITCount(t, admin, `SELECT count(*) FROM phase_lock_pattern_events`); n != eventsBefore {
		t.Fatalf("events written: %d", n-eventsBefore)
	}
}

// TestPhaseLockEngineAgreement checks that Go's recomputation over the canonical document
// equals the structure CodexHash Laravel obtained from the real engine
// (PayloadProtection::protectStructure()), then records that structure.
func TestPhaseLockEngineAgreement(t *testing.T) {
	path := os.Getenv("PHASELOCK_IT_ENGINE_FIXTURE")
	if path == "" || os.Getenv("CODEXHASH_API_URL") == "" {
		t.Skip("PHASELOCK_IT_ENGINE_FIXTURE and CODEXHASH_API_URL not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Data      json.RawMessage `json:"data"`
		CodexHash string          `json:"codexhash"`
		TIU       json.Number     `json:"tiu"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil || !codexhash.IsV2Evidence(fixture.CodexHash) {
		t.Fatalf("fixture: %v", err)
	}
	ctx := context.Background()
	canonical, err := protect.CanonicalPayload(fixture.Data)
	if err != nil {
		t.Fatal(err)
	}
	client := codexhash.NewClient()
	computed, err := client.V2Hash(ctx, canonical)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if computed.Digest != fixture.CodexHash {
		t.Fatalf("Go recomputation %s… differs from Laravel's %s…", computed.Digest[:40], fixture.CodexHash[:40])
	}
	// One changed character changes the evidence: the comparison is meaningful.
	tampered, err := client.V2Hash(ctx, strings.Replace(canonical, `"required":true`, `"required":false`, 1))
	if err != nil || tampered.Digest == fixture.CodexHash {
		t.Fatalf("tampered structure: %v", err)
	}

	db, _, admin := phaseLockIT(t)
	var doc struct {
		Method      string `json:"method"`
		Destination struct {
			Zone  string  `json:"zone"`
			Route *string `json:"route"`
			URI   string  `json:"uri"`
		} `json:"destination"`
	}
	if err := json.Unmarshal(fixture.Data, &doc); err != nil {
		t.Fatal(err)
	}
	route := doc.Destination.URI
	if doc.Destination.Route != nil {
		route = *doc.Destination.Route
	}
	rel := PhaseLockRelationship{Zone: doc.Destination.Zone, Route: route, Method: doc.Method}
	result, pkg, err := db.RecordFormStructure(ctx, FormStructureRecord{Relationship: rel, Canonical: canonical, CodexHash: computed.Digest, TIU: fixture.TIU.String()})
	if err != nil || (result != PhaseLockRecorded && result != PhaseLockKnown) {
		t.Fatalf("record engine structure: %s %v", result, err)
	}
	// Recomputing from the stored row reproduces the evidence.
	var stored string
	if err := admin.QueryRow(ctx, `SELECT data::text FROM form_structure_packages WHERE id = $1`, pkg.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	again, err := protect.CanonicalPayload(json.RawMessage(stored))
	if err != nil {
		t.Fatal(err)
	}
	fromStored, err := client.V2Hash(ctx, again)
	if err != nil || fromStored.Digest != fixture.CodexHash {
		t.Fatalf("hash recomputed from the stored row differs: %v", err)
	}
}

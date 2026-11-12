//go:build integration

package communications

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// This suite audits the unchanged Laravel migration contract. It does not
// install/alter the schema, run on codex_zones, or use application DB_* settings.
func TestDisposableCommunicationContract(t *testing.T) {
	dsn := os.Getenv("COMMUNICATION_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNICATION_TEST_DSN not set")
	}
	cfg, e := pgx.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid disposable test DSN")
	}
	if cfg.Database != "codexsecure_configuration_test" || cfg.Host != "127.0.0.1" {
		t.Fatal("requires the dedicated loopback disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, e := pgx.ConnectConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close(ctx)
	var database string
	if e = c.QueryRow(ctx, `SELECT current_database()`).Scan(&database); e != nil || database != "codexsecure_configuration_test" {
		t.Fatal("wrong test database")
	}
	tx, e := c.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := tx.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	scalar := func(q string, args ...any) string {
		t.Helper()
		var s string
		if e := tx.QueryRow(ctx, q, args...).Scan(&s); e != nil {
			t.Fatal(e)
		}
		return s
	}
	expectError := func(want, q string, args ...any) {
		t.Helper()
		exec("SAVEPOINT expected_error")
		_, e := tx.Exec(ctx, q, args...)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != want {
			t.Fatalf("expected SQLSTATE %s, got %v", want, e)
		}
		exec("ROLLBACK TO SAVEPOINT expected_error")
		exec("RELEASE SAVEPOINT expected_error")
	}
	exec("SET LOCAL ROLE codex_zones_admin")
	site := "test-" + uuid.NewString()
	key := uuid.NewString()
	siteID := scalar(`INSERT INTO sites(site_key,name,base_url) VALUES($1,'Test','https://example.test') RETURNING id::text`, site)
	exec("SET LOCAL ROLE codex_zones_runtime")
	pageID := scalar(`INSERT INTO pages(site_id,page_key,url,zone) VALUES($1,'contact','https://example.test/contact','Z1') RETURNING id::text`, siteID)
	pageKey := uuid.NewString()
	pageObject := scalar(`INSERT INTO communication_objects(site_id,page_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope) VALUES($1,$2,$3,'form','https://example.test/contact','POST','Z1','https://example.test/contact','page') RETURNING id::text`, siteID, pageID, pageKey)
	if pageObject == "" {
		t.Fatal("page registration failed")
	}
	objectID := scalar(`INSERT INTO communication_objects(site_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope,component_identifier) VALUES($1,$2,'fetch','https://example.test/a','GET','Z1','https://example.test/a','universal','main-navigation') RETURNING id::text`, siteID, key)
	exec(`UPDATE communication_objects SET structure_state='valid',structure_revision=1 WHERE id=$1`, objectID)
	doc := `{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET","POST"],"type":"fetch","zone":"Z1","enabled":true}`
	save := `SELECT codex_communications_save_draft($1,$2,$3::jsonb,$4,'admin-1',$5)::text`
	approve := `SELECT codex_communications_approve($1,$2,$3,$4,'admin-1',$5,NULL)::text`
	call := func(q string, args ...any) map[string]any {
		t.Helper()
		var m map[string]any
		if e := json.Unmarshal([]byte(scalar(q, args...)), &m); e != nil {
			t.Fatal(e)
		}
		return m
	}
	if got := call(save, site, key, doc, 0, uuid.NewString())["draft_version"]; got != float64(1) {
		t.Fatal(got)
	}
	if got := call(save, site, key, doc, 1, uuid.NewString())["draft_version"]; got != float64(2) {
		t.Fatal(got)
	}
	expectError("CX409", save, site, key, doc, 1, uuid.NewString())
	expectError("CX404", save, "other-site", key, doc, 2, uuid.NewString())
	expectError("CX422", save, site, key, `{}`, 2, uuid.NewString())
	expectError("CX422", save, site, key, doc, 2, nil)
	expectError("CX409", approve, site, key, 2, 0, uuid.NewString())
	exec(`UPDATE communication_objects SET structure_state='invalid',structure_issues='["missing_item"]',structure_revision=2 WHERE id=$1`, objectID)
	expectError("CX423", approve, site, key, 2, 2, uuid.NewString())
	expectError("23514", `UPDATE communication_objects SET structure_state='valid' WHERE id=$1`, objectID)
	exec(`UPDATE communication_objects SET structure_state='valid',structure_issues='[]',structure_revision=3 WHERE id=$1`, objectID)
	replay := uuid.NewString()
	result := call(approve, site, key, 2, 3, replay)
	if result["revision"] != float64(1) {
		t.Fatal(result)
	}
	result = call(approve, site, key, 2, 3, replay)
	if result["replayed"] != true {
		t.Fatal(result)
	}
	expectError("CX410", `SELECT codex_communications_disable($1,$2,'admin-1',$3,NULL)`, site, key, replay)
	// Runtime cannot modify any approval, draft, document or event directly.
	expectError("42501", `UPDATE communication_objects SET approval_status='revoked' WHERE id=$1`, objectID)
	expectError("42501", `UPDATE communication_configuration_revisions SET configuration='{}' WHERE communication_object_id=$1`, objectID)
	expectError("42501", `DELETE FROM communication_approval_events WHERE communication_object_id=$1`, objectID)
	// Registry supports exact repeated universal observations across pages.
	instance := uuid.NewString()
	observe := `INSERT INTO communication_observations(site_id,page_id,communication_object_id,observed_instance_key,browser_object_identifier,observed_method,transport_method,observed_zone,canonical_destination) VALUES($1,$2,$3,$4,'browser-1',$5,$5,'Z1',$6)`
	exec(observe, siteID, pageID, objectID, instance, "GET", "https://example.test/a")
	page2 := scalar(`INSERT INTO pages(site_id,page_key,url,zone) VALUES($1,'home','https://example.test/','Z1') RETURNING id::text`, siteID)
	exec(observe, siteID, page2, objectID, uuid.NewString(), "GET", "https://example.test/a")
	// Independent behavior persists without weakening identity constraints.
	exec(observe, siteID, pageID, objectID, uuid.NewString(), "POST", "https://example.test/a")
	exec(observe, siteID, pageID, objectID, uuid.NewString(), "GET", "https://example.test/changed")
	expectError("42501", `UPDATE communication_objects SET detected_url='https://example.test/changed',canonical_destination='https://example.test/changed' WHERE id=$1`, objectID)
	t.Log("independent method and destination evidence accepted")

	// A source page in the new zone does not make it a new universal registration.
	zonePage := scalar(`INSERT INTO pages(site_id,page_key,url,zone) VALUES($1,'zone-two','https://example.test/zone-two','Z2') RETURNING id::text`, siteID)
	exec(`INSERT INTO communication_observations(site_id,page_id,communication_object_id,browser_object_identifier,observed_method,transport_method,observed_zone,canonical_destination) VALUES($1,$2,$3,'zone-change','GET','GET','Z2','https://example.test/a')`, siteID, zonePage, objectID)
	// Communication zone is independent of the source-page zone.
	exec(observe, siteID, zonePage, objectID, uuid.NewString(), "GET", "https://example.test/a")
	// No supported direct update can change registration type to stand in for an observation.
	exec("SET LOCAL ROLE codex_zones_admin")
	expectError("23514", `UPDATE communication_objects SET object_type='form' WHERE id=$1`, objectID)
	exec("SET LOCAL ROLE codex_zones_runtime")
	// Draft lifetime versions prevent ABA after approval deletes a draft.
	result = call(save, site, key, doc, 0, uuid.NewString())
	if result["draft_version"] != float64(3) {
		t.Fatal(result)
	}
	expectError("CX409", approve, site, key, 2, 3, uuid.NewString())
	call(approve, site, key, 3, 3, uuid.NewString())
	history := scalar(`SELECT string_agg(revision::text || ':' || status,',' ORDER BY revision) FROM communication_configuration_revisions WHERE communication_object_id=$1`, objectID)
	if history != "1:superseded,2:approved" {
		t.Fatal(history)
	}
	call(save, site, key, doc, 0, uuid.NewString())
	call(`SELECT codex_communications_reject($1,$2,4,'admin-1',$3,'review rejected')::text`, site, key, uuid.NewString())
	if got := scalar(`SELECT r.revision::text FROM communication_objects o JOIN communication_configuration_revisions r ON r.id=o.approved_revision_id WHERE o.id=$1`, objectID); got != "2" {
		t.Fatal(got)
	}
	call(`SELECT codex_communications_disable($1,$2,'admin-1',$3,NULL)::text`, site, key, uuid.NewString())
	if got := scalar(`SELECT approval_status FROM communication_objects WHERE id=$1`, objectID); got != "disabled" {
		t.Fatal(got)
	}
	call(`SELECT codex_communications_revoke($1,$2,'admin-1',$3,NULL)::text`, site, key, uuid.NewString())
	if got := scalar(`SELECT approval_status || ':' || (approved_revision_id IS NULL)::text FROM communication_objects WHERE id=$1`, objectID); got != "revoked:true" {
		t.Fatal(got)
	}
	exec("SET LOCAL ROLE codex_zones_admin")
	expectError("23514", `UPDATE communication_configuration_revisions SET configuration=$2::jsonb WHERE communication_object_id=$1 AND revision=1`, objectID, strings.Replace(doc, "/a", "/changed", 1))
	expectError("23514", `DELETE FROM communication_approval_events WHERE communication_object_id=$1`, objectID)
}

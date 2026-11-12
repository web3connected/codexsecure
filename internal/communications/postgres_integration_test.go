//go:build integration

package communications

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func disposableStore(t *testing.T) (context.Context, *PostgresStore, *pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("COMMUNICATION_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNICATION_TEST_DSN not set")
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid test DSN")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" || cfg.ConnConfig.Database != "codexsecure_configuration_test" {
		t.Fatal("requires dedicated local disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(admin.Close)
	runtimeCfg := cfg.Copy()
	runtimeCfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, e := c.Exec(ctx, "SET ROLE codex_zones_runtime")
		return e
	}
	runtime, e := pgxpool.NewWithConfig(ctx, runtimeCfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(runtime.Close)
	site := "go-" + uuid.NewString()
	if _, e = admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'Go integration','https://example.test')`, site); e != nil {
		t.Fatal(e)
	}
	zones := map[string]bool{}
	for _, z := range []string{"Z0", "Z1", "Z2", "Z3", "Z4", "Z5", "Z6", "Z7", "Z8", "Z9", "Z10", "Z11", "Z12"} {
		zones[z] = true
	}
	return ctx, NewPostgresStore(runtime, zones), admin, site
}
func ptr[T any](v T) *T { return &v }
func discoveryFixture() Discovery {
	return Discovery{RegistrationKey: uuid.NewString(), Scope: "universal", Component: ptr("main-navigation"), Page: SourcePage{"home", "https://example.test/", "Z1"}, Detected: Detected{"https://example.test/products", "GET", "GET", "fetch", "Z1"}, Declaration: Declaration{ptr("universal"), ptr("main-navigation"), ptr("fetch"), ptr("products-link")}, BrowserIdentifier: "navigation-products", ExpectedStructureRevision: ptr(int64(0))}
}
func TestPostgresStoreLifecycle(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	ingestKey := uuid.NewString()
	first, e := s.Discover(ctx, site, ingestKey, d)
	if e != nil {
		t.Fatal(e)
	}
	if first.Structure.State != "valid" || first.Structure.Revision != 1 {
		t.Fatal(first)
	}
	repeat, e := s.Discover(ctx, site, ingestKey, d)
	if e != nil || !repeat.Replayed {
		t.Fatal(repeat, e)
	}
	changed := d
	changed.BrowserIdentifier = "other"
	if _, e = s.Discover(ctx, site, ingestKey, changed); MapError(e).Code != "ALREADY_PROCESSED" {
		t.Fatal(e)
	}

	duplicate := d
	duplicate.RegistrationKey = uuid.NewString()
	if _, e = s.Discover(ctx, site, uuid.NewString(), duplicate); MapError(e).Code != "REGISTRATION_IDENTITY_CONFLICT" {
		t.Fatal(e)
	}

	movedItem := d
	movedItem.ExpectedStructureRevision = ptr(int64(1))
	movedItem.Declaration.ItemIdentifier = ptr("different-item")
	if _, e = s.Discover(ctx, site, uuid.NewString(), movedItem); MapError(e).Code != "REGISTRATION_IDENTITY_CONFLICT" {
		t.Fatal(e)
	}
	d.ExpectedStructureRevision = ptr(int64(1))
	d.Page = SourcePage{"contact", "https://example.test/contact", "Z1"}
	if repeat, e = s.Discover(ctx, site, uuid.NewString(), d); e != nil || repeat.Structure.Revision != 1 {
		t.Fatal(repeat, e)
	}
	var count int
	if e = admin.QueryRow(ctx, `SELECT count(*) FROM communication_objects o JOIN sites s ON s.id=o.site_id WHERE s.site_key=$1`, site).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	found, e := s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Approved != nil || found.Effective != "unapproved" || len(found.Observations) != 2 {
		t.Fatal(found, e)
	}
	document := json.RawMessage(`{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["post","GET"],"type":"ajax","zone":"Z1","enabled":true}`)
	decide := func(decision Decision) json.RawMessage {
		t.Helper()
		decision.Site = site
		decision.Key = d.RegistrationKey
		decision.Actor = "admin-1"
		if decision.IdempotencyKey == "" {
			decision.IdempotencyKey = uuid.NewString()
		}
		result, e := s.Decide(ctx, decision)
		if e != nil {
			t.Fatal(e)
		}
		return result
	}
	decide(Decision{Action: "draft", Document: document, ExpectedVersion: ptr(int64(0))})
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Draft == nil || found.Draft.Version != 1 || found.Draft.Document.Type != "fetch" || found.Draft.Document.Methods[0] != "GET" {
		t.Fatal(found, e)
	}
	stale := Decision{Action: "draft", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), Document: document, ExpectedVersion: ptr(int64(0))}
	if _, e = s.Decide(ctx, stale); MapError(e).Code != "STALE_REVISION" {
		t.Fatal(e)
	}

	staleApproval := Decision{Action: "approve", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), DraftVersion: ptr(int64(2)), StructureRevision: ptr(int64(1))}
	if _, e = s.Decide(ctx, staleApproval); MapError(e).Code != "STALE_REVISION" {
		t.Fatal(e)
	}
	approveKey := uuid.NewString()
	approval := Decision{Action: "approve", DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1)), IdempotencyKey: approveKey}
	decide(approval)
	replayed := decide(approval)
	var replay map[string]any
	json.Unmarshal(replayed, &replay)
	if replay["replayed"] != true {
		t.Fatal(replay)
	}
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Effective != "approved" || found.Approved == nil || found.Draft != nil || found.Approved.Revision != 1 {
		t.Fatal(found, e)
	}
	// Exact duplicate discovery retains approval; structural changes suspend it.
	d.Declaration.Item = nil
	invalid, e := s.Discover(ctx, site, uuid.NewString(), d)
	if e != nil || invalid.Structure.Revision != 2 || invalid.Structure.State != "invalid" {
		t.Fatal(invalid, e)
	}
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Effective != "suspended" || found.Approved.Revision != 1 {
		t.Fatal(found, e)
	}
	decide(Decision{Action: "draft", Document: document, ExpectedVersion: ptr(int64(0))})
	ineligible := Decision{Action: "approve", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), DraftVersion: ptr(int64(2)), StructureRevision: ptr(int64(2))}
	if _, e = s.Decide(ctx, ineligible); MapError(e).Code != "STRUCTURE_INELIGIBLE" {
		t.Fatal(e)
	}
	d.ExpectedStructureRevision = ptr(int64(2))
	d.Declaration.Item = ptr("fetch")
	fixed, e := s.Discover(ctx, site, uuid.NewString(), d)
	if e != nil || fixed.Structure.Revision != 3 {
		t.Fatal(fixed, e)
	}
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Effective != "reapproval_required" {
		t.Fatal(found, e)
	}
	decide(Decision{Action: "approve", DraftVersion: ptr(int64(2)), StructureRevision: ptr(int64(3))})
	history, e := s.History(ctx, site, d.RegistrationKey)
	if e != nil || len(history.Revisions) != 2 || history.Revisions[0].Status != "superseded" || history.Revisions[0].Document.Methods[1] != "POST" {
		t.Fatal(history, e)
	}
	d.ExpectedStructureRevision = ptr(int64(3))
	d.Detected.Method = "POST"
	if _, e = s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Approved.Revision != 2 || found.Effective != "approved" {
		t.Fatal(found, e)
	}
	decide(Decision{Action: "draft", Document: document, ExpectedVersion: ptr(int64(0))})
	decide(Decision{Action: "reject", DraftVersion: ptr(int64(3)), Reason: ptr("keep current")})
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Approved.Revision != 2 || found.Draft != nil {
		t.Fatal(found, e)
	}
	if _, e = s.Find(ctx, "other-site", d.RegistrationKey); MapError(e).Code != "COMMUNICATION_NOT_FOUND" {
		t.Fatal(e)
	}
	if _, e = s.Find(ctx, site, uuid.NewString()); MapError(e).Code != "COMMUNICATION_NOT_FOUND" {
		t.Fatal(e)
	}
	decide(Decision{Action: "disable"})
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Effective != "disabled" || found.Approved == nil {
		t.Fatal(found, e)
	}
	decide(Decision{Action: "revoke"})
	found, e = s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.Effective != "revoked" || found.Approved != nil {
		t.Fatal(found, e)
	}
}
func TestPostgresAssessmentsAndOwnership(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	universal := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), universal); e != nil {
		t.Fatal(e)
	}
	page := discoveryFixture()
	page.Scope = "page"
	page.Component = nil
	page.Page = SourcePage{"contact", "https://example.test/contact", "Z1"}
	page.Declaration.Scope = ptr("page")
	page.Declaration.Component = nil
	if _, e := s.Discover(ctx, site, uuid.NewString(), page); e != nil {
		t.Fatal(e)
	}
	siteAssessment, e := s.Assess(ctx, site, Coverage{Type: "site"})
	if e != nil || siteAssessment.Complete || len(siteAssessment.Communications) != 2 || siteAssessment.Scope.Type != "site" {
		t.Fatal(siteAssessment, e)
	}
	pageAssessment, e := s.Assess(ctx, site, Coverage{Type: "page", Page: "https://example.test/contact"})
	if e != nil || pageAssessment.Complete || len(pageAssessment.Communications) != 1 || pageAssessment.Communications[0].Registration.Key != page.RegistrationKey {
		t.Fatal(pageAssessment, e)
	}
	universal.Page = page.Page
	universal.ExpectedStructureRevision = ptr(int64(1))
	if _, e = s.Discover(ctx, site, uuid.NewString(), universal); e != nil {
		t.Fatal(e)
	}
	pageAssessment, e = s.Assess(ctx, site, Coverage{Type: "page", Page: page.Page.URL})
	if e != nil || len(pageAssessment.Communications) != 2 {
		t.Fatal(pageAssessment, e)
	}
	for _, c := range pageAssessment.Communications {
		for _, o := range c.Observations {
			if o.Page != page.Page.URL {
				t.Fatal("page assessment leaked another page", o)
			}
		}
	}
	page.Page = SourcePage{"home", "https://example.test/", "Z1"}
	page.ExpectedStructureRevision = ptr(int64(1))
	if _, e = s.Discover(ctx, site, uuid.NewString(), page); MapError(e).Code != "REGISTRATION_IDENTITY_CONFLICT" {
		t.Fatal(e)
	}
	universal.Page.URL = "https://other.test/"
	if _, e = s.Discover(ctx, site, uuid.NewString(), universal); MapError(e).Status != 422 {
		t.Fatal(e)
	}
	if _, e = s.Assess(ctx, site, Coverage{Type: "page", Page: "https://example.test/unknown"}); MapError(e).Code != "PAGE_NOT_FOUND" {
		t.Fatal(e)
	}
}

func TestPostgresConcurrentDraftAndApprovalRetries(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	d := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	raw := json.RawMessage(`{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["GET","POST"],"type":"fetch","zone":"Z1","enabled":true}`)
	type outcome struct {
		result json.RawMessage
		err    error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			result, e := s.Decide(ctx, Decision{Action: "draft", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), Document: raw, ExpectedVersion: ptr(int64(0))})
			results <- outcome{result, e}
		}()
	}
	close(start)
	successes, stale := 0, 0
	for range 2 {
		r := <-results
		if r.err == nil {
			successes++
		} else if MapError(r.err).Code == "STALE_REVISION" {
			stale++
		} else {
			t.Fatal(r.err)
		}
	}
	if successes != 1 || stale != 1 {
		t.Fatal(successes, stale)
	}
	replay := uuid.NewString()
	start = make(chan struct{})
	for range 2 {
		go func() {
			<-start
			result, e := s.Decide(ctx, Decision{Action: "approve", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: replay, DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1))})
			results <- outcome{result, e}
		}()
	}
	close(start)
	replayed := 0
	for range 2 {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		var result map[string]any
		if e := json.Unmarshal(r.result, &result); e != nil {
			t.Fatal(e)
		}
		if result["replayed"] == true {
			replayed++
		}
	}
	history, e := s.History(ctx, site, d.RegistrationKey)
	if e != nil || len(history.Revisions) != 1 || len(history.History) != 2 || replayed != 1 {
		t.Fatal(history, replayed, e)
	}
}

//go:build integration

package communications

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func startPage(t *testing.T, ctx context.Context, s *PostgresStore, site string, page SourcePage, parent *string) string {
	t.Helper()
	key := uuid.NewString()
	if _, e := s.StartDiscovery(ctx, site, key, "scanner", DiscoveryStart{Scope: "page", Page: &page, ParentBatchKey: parent}); e != nil {
		t.Fatal(e)
	}
	return key
}
func finalize(t *testing.T, ctx context.Context, s *PostgresStore, site, key string, count *int, status string) {
	t.Helper()
	raw, e := s.FinalizeDiscovery(ctx, site, key, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: count})
	var result struct {
		Status   string
		Complete bool
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &result); e != nil || result.Status != status || result.Complete != (status == "complete") {
		t.Fatal(string(raw), e)
	}
}
func assertCoverage(t *testing.T, ctx context.Context, s *PostgresStore, site string, scope Coverage, complete bool, gap string) Assessment {
	t.Helper()
	a, e := s.Assess(ctx, site, scope)
	if e != nil || a.Complete != complete || a.CoverageBasis != "finalized_generation" {
		t.Fatal(a, e)
	}
	if gap != "" && !hasAssessmentGap(a, gap) {
		t.Fatal(a)
	}
	return a
}
func TestPostgresCoverageLifecycle(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	scope := Coverage{Type: "page", Page: d.Page.URL}
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	assertCoverage(t, ctx, s, site, scope, false, "discovery_generation_missing")
	d.ExpectedStructureRevision = ptr(int64(1))
	batch := startPage(t, ctx, s, site, d.Page, nil)
	// Canonically identical start replay and changed request conflict.
	raw, e := s.StartDiscovery(ctx, site, batch, "scanner", DiscoveryStart{Scope: "page", Page: &d.Page})
	if e != nil || !jsonContains(raw, "replayed", true) {
		t.Fatal(string(raw), e)
	}
	if _, e = s.StartDiscovery(ctx, site, batch, "other", DiscoveryStart{Scope: "page", Page: &d.Page}); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
	assertCoverage(t, ctx, s, site, scope, false, "discovery_generation_open")
	d.Batch = &DiscoveryBatch{Key: batch, Sequence: 1}
	replay := uuid.NewString()
	if _, e = s.Discover(ctx, site, replay, d); e != nil {
		t.Fatal(e)
	}
	if r, e := s.Discover(ctx, site, replay, d); e != nil || !r.Replayed {
		t.Fatal(r, e)
	}
	// Duplicate processed sequence cannot be replaced, even by a failure.
	if _, e = s.Discover(ctx, site, uuid.NewString(), d); MapError(e).Code != "DISCOVERY_BATCH_CLOSED" {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(1), "complete")
	a := assertCoverage(t, ctx, s, site, scope, true, "")
	if a.Coverage == nil || a.Coverage.CoverageAt == nil || len(a.Communications[0].Observations) != 1 {
		t.Fatal(a)
	}
	finalize(t, ctx, s, site, batch, ptr(1), "complete")
	if _, e = s.FinalizeDiscovery(ctx, site, batch, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: ptr(2)}); MapError(e).Code != "ALREADY_PROCESSED" {
		t.Fatal(e)
	}
	// Out-of-batch evidence permanently invalidates coverage.
	d.Batch = nil
	if _, e = s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	a = assertCoverage(t, ctx, s, site, scope, false, "coverage_invalidated")
	if len(a.Communications[0].Observations) != 0 {
		t.Fatal("display fallback entered assessment", a)
	}
	// An empty complete generation reports an absence without losing completeness.
	batch = startPage(t, ctx, s, site, d.Page, nil)
	finalize(t, ctx, s, site, batch, ptr(0), "complete")
	a = assertCoverage(t, ctx, s, site, scope, true, "universal_not_observed")
	if len(a.CompletenessIssues) != 0 {
		t.Fatal(a)
	}
	// Page-owned missing registrations also produce complete absence findings.
	pageOwned := d
	pageOwned.Scope = "page"
	pageOwned.Component = nil
	pageOwned.RegistrationKey = uuid.NewString()
	pageOwned.ExpectedStructureRevision = ptr(int64(0))
	pageOwned.Declaration.Scope = ptr("page")
	pageOwned.Declaration.Component = nil
	pageOwned.Declaration.ItemIdentifier = ptr("page-owned")
	if _, e = s.Discover(ctx, site, uuid.NewString(), pageOwned); e != nil {
		t.Fatal(e)
	}
	batch = startPage(t, ctx, s, site, d.Page, nil)
	d.Batch = &DiscoveryBatch{batch, 1}
	if _, e = s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(1), "complete")
	assertCoverage(t, ctx, s, site, scope, true, "page_object_missing")
	var baseline string
	if e = admin.QueryRow(ctx, `SELECT object_type FROM communication_objects WHERE registration_key=$1`, d.RegistrationKey).Scan(&baseline); e != nil || baseline != "fetch" {
		t.Fatal(baseline, e)
	}
}
func jsonContains(raw []byte, key string, want any) bool {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m[key] == want
}

func TestPostgresCoverageFailuresAndStaleness(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	batch := startPage(t, ctx, s, site, d.Page, nil)
	d.Batch = &DiscoveryBatch{batch, 1}
	invalid := d
	invalid.Detected.Method = "TRACE"
	if _, e := s.Discover(ctx, site, uuid.NewString(), invalid); MapError(e).Code != "CONFIGURATION_INVALID" {
		t.Fatal(e)
	}
	var outcome, code string
	if e := admin.QueryRow(ctx, `SELECT outcome,failure_code FROM communication_discovery_batch_items i JOIN communication_discovery_batches b ON b.id=i.batch_id WHERE b.batch_key=$1 AND sequence=1`, batch).Scan(&outcome, &code); e != nil || outcome != "rejected" || code != "CONFIGURATION_INVALID" {
		t.Fatal(outcome, code, e)
	}
	// Corrected sequence can replace a rejected row.
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(2), "incomplete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, false, "observations_missing")
	batch = startPage(t, ctx, s, site, d.Page, nil)
	if _, e := s.RecordDiscoveryFailure(ctx, site, batch, DiscoveryFailure{Sequence: 1, Outcome: "failed", Code: "PROCESSING_FAILED"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.RecordDiscoveryFailure(ctx, site, batch, DiscoveryFailure{Sequence: 3, Outcome: "rejected", Code: "CONFIGURATION_INVALID"}); e != nil {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(2), "incomplete")
	a := assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, false, "observations_failed")
	if !hasAssessmentGap(a, "unexpected_sequences") || !hasAssessmentGap(a, "observations_rejected") {
		t.Fatal(a)
	}
	stale := startPage(t, ctx, s, site, d.Page, nil)
	batch = startPage(t, ctx, s, site, d.Page, nil)
	if _, e := s.FinalizeDiscovery(ctx, site, stale, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: ptr(0)}); MapError(e).Code != "DISCOVERY_BATCH_CLOSED" {
		t.Fatal(e)
	}
	if _, e := s.FinalizeDiscovery(ctx, site, batch, "scanner", DiscoveryFinalize{Outcome: "failed"}); e != nil {
		t.Fatal(e)
	}
	assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, false, "discovery_generation_incomplete")
	// New empty page has valid coverage with zero registrations.
	empty := SourcePage{Key: "empty", URL: "https://example.test/empty", Zone: "Z1"}
	batch = startPage(t, ctx, s, site, empty, nil)
	finalize(t, ctx, s, site, batch, ptr(0), "complete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: empty.URL}, true, "")
}
func TestPostgresSiteCoverageManifest(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	pages := []SourcePage{{"home", "https://example.test/", "Z1"}, {"contact", "https://example.test/contact", "Z1"}}
	parent := uuid.NewString()
	if _, e := s.StartDiscovery(ctx, site, parent, "scanner", DiscoveryStart{Scope: "site", ExpectedPages: []string{"home", "contact"}}); e != nil {
		t.Fatal(e)
	}
	child := startPage(t, ctx, s, site, pages[0], &parent)
	finalize(t, ctx, s, site, child, ptr(0), "complete")
	finalize(t, ctx, s, site, parent, nil, "incomplete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, false, "page_not_covered")
	parent = uuid.NewString()
	if _, e := s.StartDiscovery(ctx, site, parent, "scanner", DiscoveryStart{Scope: "site", ExpectedPages: []string{"home", "contact"}}); e != nil {
		t.Fatal(e)
	}
	for _, p := range pages {
		child = startPage(t, ctx, s, site, p, &parent)
		finalize(t, ctx, s, site, child, ptr(0), "complete")
	}
	finalize(t, ctx, s, site, parent, nil, "complete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, true, "")
	// A newer child outside the parent invalidates site currency even if complete.
	child = startPage(t, ctx, s, site, pages[0], nil)
	finalize(t, ctx, s, site, child, ptr(0), "complete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, false, "page_not_covered")
	// Registered pages absent from the manifest prevent complete site scans.
	parent = uuid.NewString()
	if _, e := s.StartDiscovery(ctx, site, parent, "scanner", DiscoveryStart{Scope: "site", ExpectedPages: []string{"home"}}); e != nil {
		t.Fatal(e)
	}
	child = startPage(t, ctx, s, site, pages[0], &parent)
	finalize(t, ctx, s, site, child, ptr(0), "complete")
	finalize(t, ctx, s, site, parent, nil, "incomplete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, false, "registered_page_not_declared")
}
func TestPostgresObservationFinalizeRace(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	d.ExpectedStructureRevision = ptr(int64(1))
	for i := 0; i < 12; i++ {
		batch := startPage(t, ctx, s, site, d.Page, nil)
		var wg sync.WaitGroup
		wg.Add(3)
		var observedErrors [2]error
		var finalizedErr error
		for j := 0; j < 2; j++ {
			go func(j int) {
				defer wg.Done()
				observed := d
				observed.Batch = &DiscoveryBatch{batch, j + 1}
				_, observedErrors[j] = s.Discover(ctx, site, uuid.NewString(), observed)
			}(j)
		}
		go func() {
			defer wg.Done()
			_, finalizedErr = s.FinalizeDiscovery(ctx, site, batch, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: ptr(2)})
		}()
		wg.Wait()
		if finalizedErr != nil {
			t.Fatal(finalizedErr)
		}
		successful := 0
		for _, e := range observedErrors {
			if e == nil {
				successful++
			} else if MapError(e).Code != "DISCOVERY_BATCH_CLOSED" {
				t.Fatal(e)
			}
		}
		var status string
		var count int
		if e := admin.QueryRow(ctx, `SELECT b.status,count(v.id)::int FROM communication_discovery_batches b LEFT JOIN communication_observations v ON v.discovery_batch_id=b.id WHERE b.batch_key=$1 GROUP BY b.status`, batch).Scan(&status, &count); e != nil {
			t.Fatal(e)
		}
		if count != successful || (status == "complete") != (count == 2) || (status != "complete" && status != "incomplete") {
			t.Fatal(status, count, observedErrors)
		}
	}
}

func TestPostgresCurrentEvidenceIndividuallyEvaluated(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	document := json.RawMessage(`{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["GET","POST"],"type":"fetch","zone":"Z1","enabled":true}`)
	decide := func(request Decision) json.RawMessage {
		t.Helper()
		request.Site = site
		request.Key = d.RegistrationKey
		request.Actor = "admin-1"
		request.IdempotencyKey = uuid.NewString()
		raw, e := s.Decide(ctx, request)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	decide(Decision{Action: "draft", Document: document, ExpectedVersion: ptr(int64(0))})
	decide(Decision{Action: "approve", DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1))})
	d.ExpectedStructureRevision = ptr(int64(1))
	batch := startPage(t, ctx, s, site, d.Page, nil)
	for i := 1; i <= 6; i++ {
		observed := d
		observed.Batch = &DiscoveryBatch{batch, i}
		switch i {
		case 2:
			observed.Detected.Method = "POST"
		case 3:
			observed.Detected.Method = "DELETE"
		case 4:
			observed.Detected.Destination = "https://example.test/changed"
		case 5:
			observed.Detected.Type = "form"
		case 6:
			observed.Detected.Zone = "Z2"
		}
		if _, e := s.Discover(ctx, site, uuid.NewString(), observed); e != nil {
			t.Fatal(e)
		}
	}
	finalize(t, ctx, s, site, batch, ptr(6), "complete")
	a := assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, true, "")
	if len(a.Communications) != 1 || len(a.Communications[0].Observations) != 6 || a.Communications[0].Effective != "unauthorized" {
		t.Fatal(a)
	}
	for i, observed := range a.Communications[0].Observations {
		want := "unauthorized"
		if i < 2 || i == 5 {
			want = "approved"
		}
		if observed.Effective != want {
			t.Fatal(observed)
		}
	}
	displayed, e := s.Find(ctx, site, d.RegistrationKey)
	if e != nil || len(displayed.Observations) != 6 || displayed.Effective != "unauthorized" {
		t.Fatal(displayed, e)
	}

	// A fresh matching generation restores authority; approval and baseline are unchanged.
	batch = startPage(t, ctx, s, site, d.Page, nil)
	d.Batch = &DiscoveryBatch{batch, 1}
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(1), "complete")
	a = assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, true, "")
	if a.Communications[0].Effective != "approved" || a.Communications[0].Approved.Revision != 1 {
		t.Fatal(a)
	}
	var count int
	if e := admin.QueryRow(ctx, `SELECT count(*) FROM communication_configuration_revisions r JOIN communication_objects o ON o.id=r.communication_object_id WHERE o.registration_key=$1`, d.RegistrationKey).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
}
func TestPostgresDecisionReplayBindingAndABA(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	d := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	document := json.RawMessage(`{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true}`)
	request := Decision{Action: "draft", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), Document: document, ExpectedVersion: ptr(int64(0))}
	if _, e := s.Decide(ctx, request); e != nil {
		t.Fatal(e)
	}
	if raw, e := s.Decide(ctx, request); e != nil || !jsonContains(raw, "replayed", true) {
		t.Fatal(string(raw), e)
	}
	altered := request
	altered.Actor = "admin-2"
	if _, e := s.Decide(ctx, altered); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
	altered = request
	altered.ExpectedVersion = ptr(int64(1))
	if _, e := s.Decide(ctx, altered); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
	altered = request
	altered.Document = json.RawMessage(`{"schema_version":1,"destination":"/changed","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true}`)
	if _, e := s.Decide(ctx, altered); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
	approve := Decision{Action: "approve", Site: site, Key: d.RegistrationKey, Actor: "admin-1", IdempotencyKey: uuid.NewString(), DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1))}
	if _, e := s.Decide(ctx, approve); e != nil {
		t.Fatal(e)
	}
	altered = approve
	altered.Reason = ptr("altered")
	if _, e := s.Decide(ctx, altered); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
	request.IdempotencyKey = uuid.NewString()
	raw, e := s.Decide(ctx, request)
	if e != nil || !jsonContains(raw, "draft_version", float64(2)) {
		t.Fatal(string(raw), e)
	}
	approve.IdempotencyKey = uuid.NewString()
	if _, e = s.Decide(ctx, approve); MapError(e).Code != "STALE_REVISION" {
		t.Fatal(e)
	}
	// Legacy fixture was created by the disposable bootstrap before migration four.
	legacy := request
	legacy.Site = "go-legacy-replay-fixture"
	legacy.Key = "00000000-0000-4000-8000-000000000071"
	legacy.IdempotencyKey = "00000000-0000-4000-8000-000000000072"
	if _, e = s.Decide(ctx, legacy); MapError(e).Code != "REPLAY_CONFLICT" {
		t.Fatal(e)
	}
}
func TestPostgresFailureRecordingPreservesOriginalError(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	batch := startPage(t, ctx, s, site, d.Page, nil)
	d.Batch = &DiscoveryBatch{batch, 1}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := s.Discover(canceled, site, uuid.NewString(), d); MapError(e).Code != "INTERNAL_ERROR" {
		t.Fatal(e)
	}
	var outcome string
	if e := admin.QueryRow(ctx, `SELECT outcome FROM communication_discovery_batch_items i JOIN communication_discovery_batches b ON b.id=i.batch_id WHERE b.batch_key=$1 AND sequence=1`, batch).Scan(&outcome); e != nil || outcome != "failed" {
		t.Fatal(outcome, e)
	}
	// A closed ledger cannot accept failures: keep the validation code and say so.
	finalize(t, ctx, s, site, batch, ptr(1), "incomplete")
	d.Detected.Method = "TRACE"
	_, e := s.Discover(ctx, site, uuid.NewString(), d)
	api := MapError(e)
	if api.Code != "CONFIGURATION_INVALID" || len(api.Errors["batch.failure_recording"]) == 0 {
		t.Fatal(api, e)
	}
}

func TestPostgresBatchOwnershipAndFingerprint(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	d := discoveryFixture()
	batch := startPage(t, ctx, s, site, d.Page, nil)
	d.Batch = &DiscoveryBatch{batch, 1}
	replay := uuid.NewString()
	if _, e := s.Discover(ctx, site, replay, d); e != nil {
		t.Fatal(e)
	}
	changed := d
	changed.Batch = &DiscoveryBatch{batch, 2}
	if _, e := s.Discover(ctx, site, replay, changed); MapError(e).Code != "ALREADY_PROCESSED" {
		t.Fatal(e)
	}
	// The changed batch sequence is fingerprinted and its rejection is durable.
	finalize(t, ctx, s, site, batch, ptr(2), "incomplete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, false, "observations_rejected")
	batch = startPage(t, ctx, s, site, d.Page, nil)
	changed = d
	changed.Batch = &DiscoveryBatch{batch, 1}
	changed.ExpectedStructureRevision = ptr(int64(1))
	changed.Page = SourcePage{"other", "https://example.test/other", "Z1"}
	if _, e := s.Discover(ctx, site, uuid.NewString(), changed); MapError(e).Code != "CONFIGURATION_INVALID" {
		t.Fatal(e)
	}
	finalize(t, ctx, s, site, batch, ptr(1), "incomplete")
	if _, e := s.FinalizeDiscovery(ctx, "other-site", batch, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: ptr(0)}); MapError(e).Code != "DISCOVERY_BATCH_NOT_FOUND" {
		t.Fatal(e)
	}
}
func TestPostgresSiteInvalidationAndStaleChildren(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	d := discoveryFixture()
	startSite := func() string {
		t.Helper()
		key := uuid.NewString()
		if _, e := s.StartDiscovery(ctx, site, key, "scanner", DiscoveryStart{Scope: "site", ExpectedPages: []string{d.Page.Key}}); e != nil {
			t.Fatal(e)
		}
		return key
	}
	parent := startSite()
	child := startPage(t, ctx, s, site, d.Page, &parent)
	finalize(t, ctx, s, site, child, ptr(0), "complete")
	finalize(t, ctx, s, site, parent, nil, "complete")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, true, "")
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, false, "coverage_invalidated")
	parent = startSite()
	child = startPage(t, ctx, s, site, d.Page, &parent)
	_ = startSite()
	assertCoverage(t, ctx, s, site, Coverage{Type: "page", Page: d.Page.URL}, false, "discovery_generation_stale")
	assertCoverage(t, ctx, s, site, Coverage{Type: "site"}, false, "discovery_generation_open")
	if _, e := s.FinalizeDiscovery(ctx, site, child, "scanner", DiscoveryFinalize{Outcome: "complete", ExpectedObservations: ptr(0)}); MapError(e).Code != "DISCOVERY_BATCH_CLOSED" {
		t.Fatal(e)
	}
}

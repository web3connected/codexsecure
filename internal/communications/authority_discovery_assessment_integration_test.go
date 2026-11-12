//go:build integration

package communications

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDiscoveryChangedIntentGuardAndRetry(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	original := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), original); e != nil {
		t.Fatal(e)
	}
	document := json.RawMessage(`{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["GET","POST"],"type":"fetch","zone":"Z1","enabled":true}`)
	for _, d := range []Decision{{Action: "draft", ExpectedVersion: ptr(int64(0)), Document: document}, {Action: "approve", DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1))}} {
		d.Site = site
		d.Key = original.RegistrationKey
		d.Actor = "admin-1"
		d.IdempotencyKey = uuid.NewString()
		if _, e := s.Decide(ctx, d); e != nil {
			t.Fatal(e)
		}
	}
	original.ExpectedStructureRevision = ptr(int64(1))
	for _, tc := range []struct {
		name, field string
		change      func(*Discovery)
	}{
		{"unapproved method", "detected.method", func(d *Discovery) { d.Detected.Method = "DELETE" }},
		{"another approved method", "detected.method", func(d *Discovery) { d.Detected.Method = "POST" }},
		{"destination mismatch", "detected.destination", func(d *Discovery) { d.Detected.Destination = "https://example.test/changed" }},
		{"changed observed type", "detected.type", func(d *Discovery) { d.Detected.Type = "form" }},
		{"independent communication zone", "detected.zone", func(d *Discovery) { d.Detected.Zone = "Z2" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := original
			tc.change(&changed)
			replay := uuid.NewString()
			_, e := s.Discover(ctx, site, replay, changed)
			if e != nil {
				t.Fatal(e)
			}
			current, e := s.Find(ctx, site, original.RegistrationKey)
			want := "unauthorized"
			if changed.Detected.Method == "POST" || changed.Detected.Zone != "Z1" {
				want = "approved"
			}
			if e != nil || current.Approved == nil || current.Effective != want || current.Structure.Revision != 1 || strings.Join(current.Approved.Document.Methods, ",") != "GET,POST" {
				t.Fatal(current, e)
			}
			var recorded int
			if e = admin.QueryRow(ctx, `SELECT count(*) FROM communication_observations WHERE observed_instance_key=$1::uuid AND observed_type=$2`, replay, changed.Detected.Type).Scan(&recorded); e != nil || recorded != 1 {
				t.Fatal(recorded, e)
			}
			if _, e = s.Discover(ctx, site, uuid.NewString(), original); e != nil {
				t.Fatal(e)
			}
			current, e = s.Find(ctx, site, original.RegistrationKey)
			if e != nil || current.Effective != "approved" {
				t.Fatal(current, e)
			}

		})
	}
}
func TestPageOwnedSimilarCommunicationsDoNotMerge(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	for _, page := range []SourcePage{{"contact", "https://example.test/contact", "Z1"}, {"preferences", "https://example.test/preferences", "Z1"}} {
		d := discoveryFixture()
		d.Scope = "page"
		d.Page = page
		d.Declaration.Scope = ptr("page")
		if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
			t.Fatal(e)
		}
		assessed, e := s.Assess(ctx, site, Coverage{Type: "page", Page: page.URL})
		if e != nil || len(assessed.Communications) != 1 || assessed.Communications[0].Registration.Key != d.RegistrationKey {
			t.Fatal(assessed, e)
		}
	}
	result, e := s.Assess(ctx, site, Coverage{Type: "site"})
	if e != nil || len(result.Communications) != 2 || result.Communications[0].Registration.Key == result.Communications[1].Registration.Key {
		t.Fatal(result, e)
	}
}
func hasAssessmentGap(a Assessment, code string) bool {
	for _, g := range a.CoverageGaps {
		if g.Code == code && g.Action != "" {
			return true
		}
	}
	return false
}
func TestAssessmentReportsMissingRecordsAndProcessingFailure(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	// A persisted registration is not proof that its required observations exist.
	_, e := admin.Exec(ctx, `INSERT INTO communication_objects(site_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope) SELECT id,$2::uuid,'fetch','https://example.test/a','GET','Z1','https://example.test/a','universal' FROM sites WHERE site_key=$1`, site, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	result, e := s.Assess(ctx, site, Coverage{Type: "site"})
	if e != nil || result.Complete || result.Progress.RegisteredCommunications != 1 || result.Progress.EvaluatedCommunications != 1 || result.Progress.EvaluatedObservations != 0 || !hasAssessmentGap(result, "communication_observations_missing") || !hasAssessmentGap(result, "structure_unassessed") {
		t.Fatal(result, e)
	}
	s.pool.Close()
	failed, e := s.Assess(ctx, site, Coverage{Type: "site"})
	if e == nil || failed.Complete || failed.Site != site || !hasAssessmentGap(failed, "assessment_processing_failed") {
		t.Fatal(failed, e)
	}
}

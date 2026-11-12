package communications

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestSharedPHPFixtures(t *testing.T) {
	raw, e := os.ReadFile("testdata/approved-configuration-v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Zones []string
		Valid []struct {
			Name       string
			Input      json.RawMessage
			Normalized Document
		}
		Invalid []struct {
			Name  string
			Input json.RawMessage
		}
	}
	if e = json.Unmarshal(raw, &fixture); e != nil {
		t.Fatal(e)
	}
	zones := map[string]bool{}
	for _, z := range fixture.Zones {
		zones[z] = true
	}
	for _, c := range fixture.Valid {
		t.Run(c.Name, func(t *testing.T) {
			got, e := Validate(c.Input, zones)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(got, c.Normalized) {
				t.Fatalf("got %#v want %#v", got, c.Normalized)
			}
		})
	}
	for _, c := range fixture.Invalid {
		t.Run(c.Name, func(t *testing.T) {
			if _, e := Validate(c.Input, zones); e == nil {
				t.Fatal("invalid fixture accepted")
			}
		})
	}
}
func TestStrictDocuments(t *testing.T) {
	base := `{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true}`
	for _, raw := range []string{`null`, `[]`, `{`, base + ` {}`, base[:len(base)-1] + `,"enabled":false}`, `{"schema_version":1}`, `{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET"," get "],"type":"fetch","zone":"Z1","enabled":true}`} {
		if _, e := Validate([]byte(raw), map[string]bool{"Z1": true}); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
func TestEvaluation(t *testing.T) {
	doc := &Document{Destination: "/products", Methods: []string{"GET", "POST", "PUT"}, Type: "fetch", Zone: "Z1", Enabled: true}
	obs := Observation{Destination: "https://example.test/products", Method: "POST", Type: "fetch", Zone: "Z1"}
	cases := []struct {
		name, status, structure string
		revision                int64
		doc                     *Document
		obs                     Observation
		want                    string
	}{
		{"matching second method", "approved", "valid", 2, doc, obs, "approved"},
		{"no approval", "unapproved", "valid", 2, nil, obs, "unapproved"},
		{"disabled", "disabled", "invalid", 3, doc, obs, "disabled"},
		{"revoked", "revoked", "valid", 2, nil, obs, "revoked"},
		{"structural suspension", "approved", "invalid", 3, doc, obs, "suspended"},
		{"fixed structure still needs approval", "approved", "valid", 3, doc, obs, "reapproval_required"},
		{"unexpected status", "unapproved", "valid", 2, doc, obs, "unapproved"},
	}
	mismatch := obs
	mismatch.Method = "DELETE"
	cases = append(cases, struct {
		name, status, structure string
		revision                int64
		doc                     *Document
		obs                     Observation
		want                    string
	}{"method mismatch before stale structure", "approved", "valid", 3, doc, mismatch, "unauthorized"})
	mismatch = obs
	mismatch.Destination = "https://other.test/a"
	cases = append(cases, struct {
		name, status, structure string
		revision                int64
		doc                     *Document
		obs                     Observation
		want                    string
	}{"destination mismatch", "approved", "valid", 2, doc, mismatch, "unauthorized"})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Evaluate(c.status, c.structure, c.revision, 2, c.doc, c.obs, "https://example.test/path"); got.Effective != c.want {
				t.Fatalf("got %#v want %s", got, c.want)
			}
		})
	}
	off := *doc
	off.Enabled = false
	if got := Evaluate("approved", "invalid", 3, 2, &off, obs, "https://example.test"); got.Effective != "disabled" {
		t.Fatal(got)
	}
	typeMismatch := obs
	typeMismatch.Type = "link"
	if got := Evaluate("approved", "valid", 2, 2, doc, typeMismatch, "https://example.test"); got.Effective != "unauthorized" {
		t.Fatal(got)
	}
	crossZonePage := obs
	crossZonePage.Zone = "Z5"
	if got := Evaluate("approved", "valid", 2, 2, doc, crossZonePage, "https://example.test"); got.Effective != "approved" {
		t.Fatal(got)
	}
	protectedDestination := *doc
	protectedDestination.Destination = "/jobs/queue"
	protectedObservation := obs
	protectedObservation.Destination = "https://example.test/jobs/queue"
	if got := Evaluate("approved", "valid", 2, 2, &protectedDestination, protectedObservation, "https://example.test"); got.Effective != "unauthorized" {
		t.Fatal(got)
	}
}

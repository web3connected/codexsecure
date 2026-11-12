package communications

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAuthorizationMismatchesAndCorrectionPreservePolicy(t *testing.T) {
	doc := Document{SchemaVersion: 1, Destination: "/products", DestinationMatch: "exact", Methods: []string{"GET", "POST"}, Type: "fetch", Zone: "Z1", Enabled: true}
	before, _ := json.Marshal(doc)
	original := Observation{Page: "https://example.test/", Destination: "https://example.test/products", Method: "GET", Type: "fetch", Zone: "Z1"}
	for _, tc := range []struct {
		field  string
		change func(*Observation)
	}{
		{"method", func(o *Observation) { o.Method = "DELETE" }},
		{"destination", func(o *Observation) { o.Destination = "https://example.test/contact" }},
		{"type", func(o *Observation) { o.Type = "form" }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			changed := original
			tc.change(&changed)
			result := Evaluate("approved", "valid", 1, 1, &doc, changed, "https://example.test/")
			if result.Effective != "unauthorized" || !reflect.DeepEqual(result.Mismatches, []string{tc.field + "_mismatch"}) {
				t.Fatal(result)
			}
			if restored := Evaluate("approved", "valid", 1, 1, &doc, original, "https://example.test/"); restored.Effective != "approved" {
				t.Fatal(restored)
			}
		})
	}
	crossZonePage := original
	crossZonePage.Zone = "Z5"
	if got := Evaluate("approved", "valid", 1, 1, &doc, crossZonePage, "https://example.test/"); got.Effective != "approved" {
		t.Fatalf("page zone affected destination authorization: %#v", got)
	}
	protectedDestination := doc
	protectedDestination.Destination = "/jobs/queue"
	protectedObservation := original
	protectedObservation.Destination = "https://example.test/jobs/queue"
	if got := Evaluate("approved", "valid", 1, 1, &protectedDestination, protectedObservation, "https://example.test/"); got.Effective != "unauthorized" || !reflect.DeepEqual(got.Mismatches, []string{"zone_mismatch"}) {
		t.Fatalf("Z1 communication authorized for a Z5 destination: %#v", got)
	}
	unassignedDestination := doc
	unassignedDestination.Destination = "/qzxv-9284"
	unassignedObservation := original
	unassignedObservation.Destination = "https://example.test/qzxv-9284"
	if got := Evaluate("approved", "valid", 1, 1, &unassignedDestination, unassignedObservation, "https://example.test/"); got.Effective != "unauthorized" || !reflect.DeepEqual(got.Mismatches, []string{"zone_mismatch"}) {
		t.Fatalf("unassigned destination was authorized: %#v", got)
	}
	externalDestination := doc
	externalDestination.Destination = "https://remote.test/products"
	externalObservation := original
	externalObservation.Destination = externalDestination.Destination
	if got := Evaluate("approved", "valid", 1, 1, &externalDestination, externalObservation, "https://example.test/"); got.Effective != "unauthorized" || !reflect.DeepEqual(got.Mismatches, []string{"zone_mismatch"}) {
		t.Fatalf("external destination borrowed a local zone assignment: %#v", got)
	}
	secondPage := original
	secondPage.Page = "https://example.test/contact"
	secondPage.Zone = "Z5"
	secondPage.Method = "POST"
	if got := Evaluate("approved", "valid", 1, 1, &doc, secondPage, "https://example.test/"); got.Effective != "approved" {
		t.Fatal(got)
	}
	if got := Evaluate("approved", "invalid", 2, 1, &doc, original, "https://example.test/"); got.Effective != "suspended" {
		t.Fatal(got)
	}
	if got := Evaluate("approved", "valid", 3, 1, &doc, original, "https://example.test/"); got.Effective != "reapproval_required" {
		t.Fatal(got)
	}
	disabled := doc
	disabled.Enabled = false
	if got := Evaluate("approved", "valid", 1, 1, &disabled, original, "https://example.test/"); got.Effective != "disabled" {
		t.Fatal(got)
	}
	if got := Evaluate("revoked", "valid", 1, 1, nil, original, "https://example.test/"); got.Effective != "revoked" {
		t.Fatal(got)
	}
	after, _ := json.Marshal(doc)
	if string(before) != string(after) {
		t.Fatal("evaluation mutated approved policy")
	}
}

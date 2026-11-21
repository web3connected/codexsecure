//go:build integration

package communications

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// HTTP Codex7FP Stage 3: a form's approved structural identity (Form ID, values-free field structure and
// CodexHash v2 digest) is stored with its draft, carried by the decision functions into the immutable
// approved revision, and returned by Find exactly as approved. A form approved without one is unchanged.
func TestPostgresApprovedStructureBinding(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	d := discoveryFixture()
	d.Component = ptr("admin-auth-login")
	d.Declaration = Declaration{ptr("universal"), ptr("admin-auth-login"), ptr("form"), ptr("form")}
	d.Detected = Detected{"https://example.test/admin/login", "POST", "POST", "form", "Z2"}
	d.BrowserIdentifier = "admin-auth-login"
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	decide := func(decision Decision) error {
		decision.Site, decision.Key, decision.Actor, decision.IdempotencyKey = site, d.RegistrationKey, "admin-1", uuid.NewString()
		_, e := s.Decide(ctx, decision)
		return e
	}
	hash := "ch2:inv-dual/1:512:" + strings.Repeat("5a", 64)
	binding := `{"schema":"codexsecure-communication-structure-v1","form_id":"admin_auth_login","fields":[` +
		`{"part":"form","path":"challenge_id","shape":"value","presence":"always"},{"part":"form","path":"remember","shape":"value","presence":"conditional"}],"codex_hash":"` + hash + `"}`
	document := `{"schema_version":1,"destination":"/admin/login","destination_match":"exact","methods":["POST"],"type":"form","zone":"Z2","enabled":true,"structure":` + binding + `}`

	// A binding carrying values never reaches the database (Go rejects it; the SQL validator would too).
	if e := decide(Decision{Action: "draft", Document: json.RawMessage(strings.Replace(document, `"codex_hash"`, `"values":{"email":"a@b.c"},"codex_hash"`, 1)), ExpectedVersion: ptr(int64(0))}); e == nil {
		t.Fatal("a structure carrying values was accepted")
	}
	// The SQL validator itself refuses a binding on a non-form document, whatever the caller validated.
	var valid bool
	if e := admin.QueryRow(ctx, `SELECT public.codex_communications_configuration_valid($1::jsonb)`, strings.Replace(document, `"type":"form"`, `"type":"fetch"`, 1)).Scan(&valid); e != nil || valid {
		t.Fatal("SQL accepted a structure on a fetch", e)
	}

	if e := decide(Decision{Action: "draft", Document: json.RawMessage(document), ExpectedVersion: ptr(int64(0))}); e != nil {
		t.Fatal(e)
	}
	if e := decide(Decision{Action: "approve", DraftVersion: ptr(int64(1)), StructureRevision: ptr(int64(1))}); e != nil {
		t.Fatal(e)
	}
	found, e := s.Find(ctx, site, d.RegistrationKey)
	if e != nil || found.ApprovalStatus != "approved" || found.Approved == nil || found.Approved.Document.Structure == nil {
		t.Fatal(found, e)
	}
	got := found.Approved.Document.Structure
	if got.FormID != "admin_auth_login" || got.CodexHash != hash || got.Schema != StructureSchema || len(got.Fields) != 2 || got.Fields[1].Presence != "conditional" {
		t.Fatalf("approved binding changed: %#v", got)
	}
	// Stored in the immutable approved revision, byte for byte what the administrator approved.
	var stored string
	if e = admin.QueryRow(ctx, `SELECT r.configuration->'structure'->>'codex_hash' FROM communication_configuration_revisions r JOIN communication_objects o ON o.approved_revision_id=r.id WHERE o.registration_key=$1::uuid`, d.RegistrationKey).Scan(&stored); e != nil || stored != hash {
		t.Fatal(stored, e)
	}
	if _, e = admin.Exec(ctx, `UPDATE communication_configuration_revisions SET configuration = jsonb_set(configuration,'{structure,codex_hash}','"x"') WHERE status='approved' AND communication_object_id=(SELECT id FROM communication_objects WHERE registration_key=$1::uuid)`, d.RegistrationKey); e == nil {
		t.Fatal("an approved revision's structure binding was rewritten")
	}

	// A changed structure is a new draft and a new approval; the previous binding stays in history.
	if e = decide(Decision{Action: "draft", Document: json.RawMessage(strings.Replace(document, hash, "ch2:inv-dual/1:512:"+strings.Repeat("6b", 64), 1)), ExpectedVersion: ptr(int64(0))}); e != nil {
		t.Fatal(e)
	}
	if found, e = s.Find(ctx, site, d.RegistrationKey); e != nil || found.Approved.Document.Structure.CodexHash != hash || found.Draft == nil || found.Draft.Document.Structure.CodexHash == hash {
		t.Fatal("the approved binding changed before approval", found, e)
	}
}

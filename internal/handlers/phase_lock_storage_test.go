package handlers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// Phase Lock storage API (CodexSecure P3): authentication, contract validation, CodexHash v2
// recomputation and outcome mapping. The store and the engine are fakes here; the SQL is
// exercised against PostgreSQL by internal/database/phase_lock_integration_test.go.

const (
	phaseLockTestBase = "/api/v1/sites/test-site/phase-lock"
	phaseLockTestHash = "ch2:inv-dual/1:512:" + "ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12" +
		"ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12"
	phaseLockOtherHash = "ch2:inv-dual/1:512:" + "cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34" +
		"cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34cd34"
)

type phaseLockHasherFake struct {
	inputs []string
	digest string
	err    error
}

func (h *phaseLockHasherFake) V2Hash(_ context.Context, text string) (codexhash.V2HashResult, error) {
	h.inputs = append(h.inputs, text)
	if h.err != nil {
		return codexhash.V2HashResult{}, h.err
	}
	return codexhash.V2HashResult{Digest: h.digest}, nil
}

type phaseLockStoreFake struct {
	records   []database.FormStructureRecord
	discovers []string
	err       error
	result    string
	pattern   *database.PhaseLockPattern
	entries   []database.PhaseLockEntry
}

func (s *phaseLockStoreFake) RecordFormStructure(_ context.Context, r database.FormStructureRecord) (string, database.FormStructurePackage, error) {
	s.records = append(s.records, r)
	if s.err != nil {
		return "", database.FormStructurePackage{}, s.err
	}
	return s.result, database.FormStructurePackage{ID: 7, Relationship: r.Relationship, Data: json.RawMessage(r.Canonical),
		CodexHash: r.CodexHash, TIU: r.TIU, CreatedAt: time.Unix(1_800_000_000, 0).UTC()}, nil
}

func (s *phaseLockStoreFake) RelationshipPatterns(_ context.Context, _ database.PhaseLockRelationship) ([]database.PhaseLockEntry, error) {
	return s.entries, s.err
}

func (s *phaseLockStoreFake) Pattern(_ context.Context, _ database.PhaseLockRelationship, _ string) (*database.PhaseLockPattern, error) {
	return s.pattern, s.err
}

func (s *phaseLockStoreFake) DiscoverPattern(_ context.Context, rel database.PhaseLockRelationship, hash, authority string) (string, *database.PhaseLockPattern, error) {
	s.discovers = append(s.discovers, authority)
	if s.err != nil {
		return "", nil, s.err
	}
	return s.result, s.pattern, nil
}

type phaseLockTestRouter struct {
	router *gin.Engine
	signer *alignmentSigner
}

func newPhaseLockRouter(t *testing.T, hasher v2Hasher, store phaseLockStore) phaseLockTestRouter {
	t.Helper()
	gin.SetMode(gin.TestMode)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SERVICE_API_KEY", alignmentTestKey)
	t.Setenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS", "gw-test:"+base64.RawURLEncoding.EncodeToString(public))
	router := gin.New()
	NewPhaseLockStorageHandler(hasher, store).RegisterRoutes(router.Group("/api/v1"))
	return phaseLockTestRouter{router: router, signer: &alignmentSigner{kid: "gw-test", key: private}}
}

// call sends body to the operation with the service key and a valid assertion for its path,
// unless claims overrides or headers say otherwise.
func (r phaseLockTestRouter) call(t *testing.T, operation, body string, claims map[string]any, headers map[string]*string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	path := phaseLockTestBase + operation
	merged := map[string]any{"path": path}
	for name, value := range claims {
		merged[name] = value
	}
	assertion := r.signer.sign(t, r.signer.claims(merged))
	key := alignmentTestKey
	sent := map[string]*string{"X-Service-Key": &key, "X-CodexSecure-Gateway-Assertion": &assertion}
	for name, value := range headers {
		sent[name] = value
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for name, value := range sent {
		if value != nil {
			req.Header.Set(name, *value)
		}
	}
	rec := httptest.NewRecorder()
	r.router.ServeHTTP(rec, req)
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec, payload
}

// phaseLockDocument is FormStructure::document() for a Contact form, keys deliberately unsorted.
func phaseLockDocument() map[string]any {
	return map[string]any{
		"method": "POST",
		"fields": []any{
			map[string]any{"name": "email", "part": "form", "controls": []any{"email"}, "presence": "always", "required": true},
			map[string]any{"name": "attachment", "part": "files", "controls": []any{"file"}, "presence": "conditional", "required": false},
		},
		"destination": map[string]any{"zone": "Z1", "route": "contact.store", "uri": "contact", "controller": "App\\Http\\Controllers\\ContactController", "action": "store"},
	}
}

const phaseLockCanonical = `{"destination":{"action":"store","controller":"App\\Http\\Controllers\\ContactController","route":"contact.store","uri":"contact","zone":"Z1"},` +
	`"fields":[{"controls":["email"],"name":"email","part":"form","presence":"always","required":true},` +
	`{"controls":["file"],"name":"attachment","part":"files","presence":"conditional","required":false}],"method":"POST"}`

func phaseLockRelationshipJSON() map[string]any {
	return map[string]any{"zone": "Z1", "route": "contact.store", "method": "POST"}
}

func phaseLockRecordBody(mutate func(body map[string]any)) string {
	doc := phaseLockDocument()
	submitted, _ := json.Marshal(map[string]any{"payload": doc})
	body := map[string]any{
		"relationship":   phaseLockRelationshipJSON(),
		"data":           doc,
		"submitted_json": string(submitted),
		"security":       map[string]any{"codexhash": phaseLockTestHash, "tiu": json.Number("216832585.431537")},
	}
	if mutate != nil {
		mutate(body)
	}
	encoded, _ := json.Marshal(body)
	return string(encoded)
}

func encodeJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func phaseLockCode(t *testing.T, rec *httptest.ResponseRecorder, payload map[string]any, status int, code string) {
	t.Helper()
	if rec.Code != status || payload["code"] != code {
		t.Fatalf("want %d %s, got %d %s", status, code, rec.Code, rec.Body.String())
	}
	if _, leaked := payload["data"]; leaked {
		t.Fatalf("refusal carries data: %s", rec.Body.String())
	}
}

func TestPhaseLockStorageAuthentication(t *testing.T) {
	hasher := &phaseLockHasherFake{digest: phaseLockTestHash}
	store := &phaseLockStoreFake{result: database.PhaseLockRecorded}
	r := newPhaseLockRouter(t, hasher, store)
	body := phaseLockRecordBody(nil)
	wrong := "wrong-key"
	forged := "eyJhbGciOiJFZERTQSJ9.e30.AAAA"

	rec, payload := r.call(t, "/structures/record", body, nil, map[string]*string{"X-Service-Key": nil})
	phaseLockCode(t, rec, payload, http.StatusUnauthorized, "SERVICE_KEY_INVALID")
	rec, payload = r.call(t, "/structures/record", body, nil, map[string]*string{"X-Service-Key": &wrong})
	phaseLockCode(t, rec, payload, http.StatusUnauthorized, "SERVICE_KEY_INVALID")
	rec, payload = r.call(t, "/structures/record", body, nil, map[string]*string{"X-CodexSecure-Gateway-Assertion": nil})
	phaseLockCode(t, rec, payload, http.StatusForbidden, "ADMIN_UNAUTHORIZED")
	rec, payload = r.call(t, "/structures/record", body, nil, map[string]*string{"X-CodexSecure-Gateway-Assertion": &forged})
	phaseLockCode(t, rec, payload, http.StatusForbidden, "ADMIN_UNAUTHORIZED")

	past := time.Now().Add(-5 * time.Minute).Unix()
	for name, claims := range map[string]map[string]any{
		"expired":        {"iat": past, "exp": past + 30},
		"other site":     {"channel_sites": []string{"other-site"}},
		"read grant":     {"channel_sites": nil, "read_sites": []string{alignmentTestSite}},
		"other path":     {"path": phaseLockTestBase + "/patterns/discover"},
		"other method":   {"method": http.MethodGet},
		"other audience": {"aud": "someone-else"},
	} {
		rec, payload = r.call(t, "/structures/record", body, claims, nil)
		if rec.Code != http.StatusForbidden || payload["code"] != "ADMIN_UNAUTHORIZED" {
			t.Fatalf("%s: got %d %s", name, rec.Code, rec.Body.String())
		}
	}

	// A replayed assertion (same rid) is refused the second time.
	assertion := r.signer.sign(t, r.signer.claims(map[string]any{"path": phaseLockTestBase + "/patterns/lookup", "rid": "replayed-rid"}))
	lookup := encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": phaseLockTestHash})
	if rec, _ := r.call(t, "/patterns/lookup", lookup, nil, map[string]*string{"X-CodexSecure-Gateway-Assertion": &assertion}); rec.Code != http.StatusOK {
		t.Fatalf("first use: %d %s", rec.Code, rec.Body.String())
	}
	rec, payload = r.call(t, "/patterns/lookup", lookup, nil, map[string]*string{"X-CodexSecure-Gateway-Assertion": &assertion})
	phaseLockCode(t, rec, payload, http.StatusForbidden, "ADMIN_UNAUTHORIZED")

	if len(hasher.inputs) != 0 || len(store.records) != 0 {
		t.Fatalf("unauthenticated requests reached the engine or store: %d %d", len(hasher.inputs), len(store.records))
	}

	// The key unset closes every route (ChannelAuth reads it when the routes are built).
	closed := newPhaseLockRouter(t, hasher, store)
	t.Setenv("SERVICE_API_KEY", "")
	closed.router = gin.New()
	NewPhaseLockStorageHandler(hasher, store).RegisterRoutes(closed.router.Group("/api/v1"))
	rec, payload = closed.call(t, "/structures/record", body, nil, nil)
	phaseLockCode(t, rec, payload, http.StatusUnauthorized, "SERVICE_KEY_INVALID")
}

func TestPhaseLockStorageRecordsVerifiedStructures(t *testing.T) {
	hasher := &phaseLockHasherFake{digest: phaseLockTestHash}
	store := &phaseLockStoreFake{result: database.PhaseLockRecorded}
	r := newPhaseLockRouter(t, hasher, store)

	rec, payload := r.call(t, "/structures/record", phaseLockRecordBody(nil), nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("recorded: %d %s", rec.Code, rec.Body.String())
	}
	// The engine hashed the canonical document (codexsecure-protect-payload-v1), not the request text.
	if len(hasher.inputs) != 1 || hasher.inputs[0] != phaseLockCanonical {
		t.Fatalf("hash input %q", hasher.inputs)
	}
	got := store.records[0]
	if got.Canonical != phaseLockCanonical || got.CodexHash != phaseLockTestHash || got.TIU != "216832585.431537" ||
		got.Relationship != (database.PhaseLockRelationship{Zone: "Z1", Route: "contact.store", Method: "POST"}) {
		t.Fatalf("store received %+v", got)
	}
	data := payload["data"].(map[string]any)
	pkg := data["package"].(map[string]any)
	security := pkg["security"].(map[string]any)
	if data["result"] != "recorded" || security["codexhash"] != phaseLockTestHash || security["tiu"] != 216832585.431537 ||
		pkg["data"].(map[string]any)["method"] != "POST" {
		t.Fatalf("answer %s", rec.Body.String())
	}
	if _, exposed := pkg["id"]; exposed {
		t.Fatal("package id exposed")
	}

	store.result = database.PhaseLockKnown
	rec, payload = r.call(t, "/structures/record", phaseLockRecordBody(nil), nil, nil)
	if rec.Code != http.StatusOK || payload["data"].(map[string]any)["result"] != "known" {
		t.Fatalf("known: %d %s", rec.Code, rec.Body.String())
	}

	// An unnamed destination is identified by its URI, as FormStructure::relationship().
	rec, _ = r.call(t, "/structures/record", phaseLockRecordBody(func(b map[string]any) {
		b["data"].(map[string]any)["destination"].(map[string]any)["route"] = nil
		b["relationship"].(map[string]any)["route"] = "contact"
		delete(b, "submitted_json")
	}), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("unnamed route: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPhaseLockStorageRefusesUnverifiedEvidence(t *testing.T) {
	store := &phaseLockStoreFake{result: database.PhaseLockRecorded}

	// The recomputed CodexHash differs from the reported one: refused, nothing stored.
	mismatch := &phaseLockHasherFake{digest: phaseLockOtherHash}
	r := newPhaseLockRouter(t, mismatch, store)
	rec, payload := r.call(t, "/structures/record", phaseLockRecordBody(nil), nil, nil)
	phaseLockCode(t, rec, payload, http.StatusUnprocessableEntity, "PHASE_LOCK_EVIDENCE_MISMATCH")

	// The engine is unavailable or answers outside the v2 contract: fail closed, no fallback.
	for _, cause := range []error{codexhash.ErrUnavailable, codexhash.ErrInvalidResponse} {
		r = newPhaseLockRouter(t, &phaseLockHasherFake{err: fmt.Errorf("%w: test", cause)}, store)
		rec, payload = r.call(t, "/structures/record", phaseLockRecordBody(nil), nil, nil)
		phaseLockCode(t, rec, payload, http.StatusServiceUnavailable, "PHASE_LOCK_CODEXHASH_UNAVAILABLE")
	}
	r = newPhaseLockRouter(t, nil, store)
	rec, payload = r.call(t, "/structures/record", phaseLockRecordBody(nil), nil, nil)
	phaseLockCode(t, rec, payload, http.StatusServiceUnavailable, "PHASE_LOCK_CODEXHASH_UNAVAILABLE")

	// A non-v2 (bare SHA-512 hex) report never reaches the engine.
	sha := &phaseLockHasherFake{digest: phaseLockTestHash}
	r = newPhaseLockRouter(t, sha, store)
	rec, payload = r.call(t, "/structures/record", phaseLockRecordBody(func(b map[string]any) {
		b["security"].(map[string]any)["codexhash"] = strings.Repeat("ab", 64)
	}), nil, nil)
	phaseLockCode(t, rec, payload, http.StatusUnprocessableEntity, "PHASE_LOCK_FIELD_INVALID")
	if len(store.records) != 0 || len(sha.inputs) != 0 {
		t.Fatalf("unverified evidence reached the store: %d", len(store.records))
	}
}

func TestPhaseLockStorageRefusesInvalidStructures(t *testing.T) {
	hasher := &phaseLockHasherFake{digest: phaseLockTestHash}
	store := &phaseLockStoreFake{result: database.PhaseLockRecorded}
	r := newPhaseLockRouter(t, hasher, store)

	field := func(b map[string]any) map[string]any {
		return b["data"].(map[string]any)["fields"].([]any)[0].(map[string]any)
	}
	destination := func(b map[string]any) map[string]any {
		return b["data"].(map[string]any)["destination"].(map[string]any)
	}
	cases := map[string]struct {
		mutate func(map[string]any)
		status int
		code   string
	}{
		"field value":          {func(b map[string]any) { field(b)["value"] = "visitor@example.com" }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"payload member":       {func(b map[string]any) { b["data"].(map[string]any)["payload"] = map[string]any{"email": "x"} }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"number":               {func(b map[string]any) { field(b)["required"] = 1 }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"number control":       {func(b map[string]any) { field(b)["controls"] = []any{1} }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"null control":         {func(b map[string]any) { field(b)["controls"] = []any{nil} }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"empty name":           {func(b map[string]any) { field(b)["name"] = "" }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"unknown part":         {func(b map[string]any) { field(b)["part"] = "query" }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"missing presence":     {func(b map[string]any) { delete(field(b), "presence") }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"missing route member": {func(b map[string]any) { delete(destination(b), "route") }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"null uri":             {func(b map[string]any) { destination(b)["uri"] = nil }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"fields null":          {func(b map[string]any) { b["data"].(map[string]any)["fields"] = nil }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"array data":           {func(b map[string]any) { b["data"] = []any{} }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"reserved codexhash":   {func(b map[string]any) { b["data"].(map[string]any)["codexhash"] = phaseLockTestHash }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"submitted_json other": {func(b map[string]any) { b["submitted_json"] = `{"payload":{"method":"POST"}}` }, 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		"method mismatch": {func(b map[string]any) {
			b["data"].(map[string]any)["method"] = "PUT"
			delete(b, "submitted_json")
		}, 422, "PHASE_LOCK_RELATIONSHIP_MISMATCH"},
		"zone mismatch": {func(b map[string]any) {
			destination(b)["zone"] = "Z2"
			delete(b, "submitted_json")
		}, 422, "PHASE_LOCK_RELATIONSHIP_MISMATCH"},
		"route mismatch":   {func(b map[string]any) { b["relationship"].(map[string]any)["route"] = "contact.other" }, 422, "PHASE_LOCK_RELATIONSHIP_MISMATCH"},
		"GET relationship": {func(b map[string]any) { b["relationship"].(map[string]any)["method"] = "GET" }, 422, "PHASE_LOCK_FIELD_INVALID"},
		"zone Z13":         {func(b map[string]any) { b["relationship"].(map[string]any)["zone"] = "Z13" }, 422, "PHASE_LOCK_FIELD_INVALID"},
		"blank route":      {func(b map[string]any) { b["relationship"].(map[string]any)["route"] = "  " }, 422, "PHASE_LOCK_FIELD_INVALID"},
		"tiu zero":         {func(b map[string]any) { b["security"].(map[string]any)["tiu"] = 0 }, 422, "PHASE_LOCK_FIELD_INVALID"},
		"tiu string":       {func(b map[string]any) { b["security"].(map[string]any)["tiu"] = "216832585.431537" }, 422, "PHASE_LOCK_FIELD_INVALID"},
		"client state":     {func(b map[string]any) { b["state"] = "approved" }, 400, "PHASE_LOCK_REQUEST_INVALID"},
		"visitor ip":       {func(b map[string]any) { b["ip"] = "203.0.113.9" }, 400, "PHASE_LOCK_REQUEST_INVALID"},
		"session":          {func(b map[string]any) { b["relationship"].(map[string]any)["session_id"] = "s" }, 400, "PHASE_LOCK_REQUEST_INVALID"},
		"user agent":       {func(b map[string]any) { b["security"].(map[string]any)["user_agent"] = "UA" }, 400, "PHASE_LOCK_REQUEST_INVALID"},
	}
	for name, c := range cases {
		rec, payload := r.call(t, "/structures/record", phaseLockRecordBody(c.mutate), nil, nil)
		if rec.Code != c.status || payload["code"] != c.code {
			t.Fatalf("%s: want %d %s, got %d %s", name, c.status, c.code, rec.Code, rec.Body.String())
		}
	}

	// Duplicate keys, trailing data, non-objects and oversize bodies.
	for name, body := range map[string]string{
		"duplicate key": strings.Replace(phaseLockRecordBody(nil), `"method":"POST"}`, `"method":"POST","method":"POST"}`, 1),
		"two objects":   phaseLockRecordBody(nil) + "{}",
		"array":         "[]",
		"empty":         "",
		"oversize":      `{"relationship":"` + strings.Repeat("a", phaseLockMaxBytes) + `"}`,
	} {
		rec, payload := r.call(t, "/structures/record", body, nil, nil)
		if rec.Code != http.StatusBadRequest && !(name == "duplicate key" && rec.Code == http.StatusUnprocessableEntity) {
			t.Fatalf("%s: got %d %s", name, rec.Code, rec.Body.String())
		}
		if payload["data"] != nil {
			t.Fatalf("%s: data in refusal", name)
		}
	}
	if len(store.records) != 0 || len(hasher.inputs) != 0 {
		t.Fatalf("invalid structures reached the engine (%d) or store (%d)", len(hasher.inputs), len(store.records))
	}
}

func TestPhaseLockStoragePatternsAndLookup(t *testing.T) {
	pending, rel := "pending", database.PhaseLockRelationship{Zone: "Z1", Route: "contact.store", Method: "POST"}
	pkg := func(hash string) database.FormStructurePackage {
		return database.FormStructurePackage{ID: 1, Relationship: rel, Data: json.RawMessage(phaseLockCanonical), CodexHash: hash, TIU: "216832585.431537"}
	}
	store := &phaseLockStoreFake{entries: []database.PhaseLockEntry{
		{Package: pkg(phaseLockTestHash), Pattern: &database.PhaseLockPattern{ID: 3, Relationship: rel, StructureCodexHash: phaseLockTestHash, Authority: database.PhaseLockDeveloperAdmin, State: &pending}},
		{Package: pkg(phaseLockOtherHash), Pattern: &database.PhaseLockPattern{ID: 4, Relationship: rel, StructureCodexHash: phaseLockOtherHash, Authority: database.PhaseLockCommunicationObject}},
		{Package: pkg(phaseLockTestHash[:len(phaseLockTestHash)-1] + "0")},
	}}
	r := newPhaseLockRouter(t, nil, store)

	rec, payload := r.call(t, "/relationships/patterns", encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON()}), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("patterns: %d %s", rec.Code, rec.Body.String())
	}
	entries := payload["data"].(map[string]any)["entries"].([]any)
	if len(entries) != 3 {
		t.Fatalf("entries %d", len(entries))
	}
	first := entries[0].(map[string]any)["pattern"].(map[string]any)
	second := entries[1].(map[string]any)["pattern"].(map[string]any)
	if first["state"] != "pending" || first["authority"] != "developer_admin" || second["state"] != nil || second["authority"] != "communication_object" {
		t.Fatalf("patterns %s", rec.Body.String())
	}
	if entries[2].(map[string]any)["pattern"] != nil {
		t.Fatal("undiscovered package has a pattern")
	}
	if want := []string{"relationship", "structure_codexhash", "authority", "state"}; len(first) != len(want) {
		t.Fatalf("pattern members %v", first)
	}

	store.pattern = store.entries[0].Pattern
	lookup := encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": phaseLockTestHash})
	rec, payload = r.call(t, "/patterns/lookup", lookup, nil, nil)
	if rec.Code != http.StatusOK || payload["data"].(map[string]any)["result"] != "found" {
		t.Fatalf("found: %d %s", rec.Code, rec.Body.String())
	}
	store.pattern = nil
	rec, payload = r.call(t, "/patterns/lookup", lookup, nil, nil)
	if rec.Code != http.StatusOK || payload["data"].(map[string]any)["result"] != "not_found" || payload["data"].(map[string]any)["pattern"] != nil {
		t.Fatalf("not_found: %d %s", rec.Code, rec.Body.String())
	}
	rec, payload = r.call(t, "/patterns/lookup", encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": "sha512:x"}), nil, nil)
	phaseLockCode(t, rec, payload, http.StatusUnprocessableEntity, "PHASE_LOCK_FIELD_INVALID")
}

func TestPhaseLockStorageDiscovery(t *testing.T) {
	pending, rel := "pending", database.PhaseLockRelationship{Zone: "Z1", Route: "contact.store", Method: "POST"}
	store := &phaseLockStoreFake{result: database.PhaseLockDiscovered,
		pattern: &database.PhaseLockPattern{Relationship: rel, StructureCodexHash: phaseLockTestHash, Authority: database.PhaseLockDeveloperAdmin, State: &pending}}
	r := newPhaseLockRouter(t, nil, store)
	body := func(authority any, extra map[string]any) string {
		b := map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": phaseLockTestHash, "authority": authority}
		for k, v := range extra {
			b[k] = v
		}
		return encodeJSON(b)
	}

	rec, payload := r.call(t, "/patterns/discover", body("developer_admin", nil), nil, nil)
	if rec.Code != http.StatusCreated || payload["data"].(map[string]any)["pattern"].(map[string]any)["state"] != "pending" {
		t.Fatalf("discovered: %d %s", rec.Code, rec.Body.String())
	}
	store.result = database.PhaseLockExisting
	rec, payload = r.call(t, "/patterns/discover", body("communication_object", nil), nil, nil)
	if rec.Code != http.StatusOK || payload["data"].(map[string]any)["result"] != "existing" {
		t.Fatalf("existing: %d %s", rec.Code, rec.Body.String())
	}
	store.result, store.pattern = database.PhaseLockNotFound, nil
	rec, payload = r.call(t, "/patterns/discover", body("developer_admin", nil), nil, nil)
	if rec.Code != http.StatusOK || payload["data"].(map[string]any)["result"] != "not_found" || payload["data"].(map[string]any)["pattern"] != nil {
		t.Fatalf("not_found: %d %s", rec.Code, rec.Body.String())
	}

	calls := len(store.discovers)
	for name, b := range map[string]string{
		"unknown authority": body("enterprise", nil),
		"null authority":    body(nil, nil),
	} {
		rec, payload = r.call(t, "/patterns/discover", b, nil, nil)
		if rec.Code != http.StatusUnprocessableEntity || payload["code"] != "PHASE_LOCK_FIELD_INVALID" {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	// No client-supplied approval: there is no state, decision or actor field.
	for _, extra := range []map[string]any{{"state": "approved"}, {"decided_by": "web3codex-admin:1"}, {"password": "x"}} {
		rec, payload = r.call(t, "/patterns/discover", body("developer_admin", extra), nil, nil)
		phaseLockCode(t, rec, payload, http.StatusBadRequest, "PHASE_LOCK_REQUEST_INVALID")
	}
	if len(store.discovers) != calls {
		t.Fatal("invalid discovery reached the store")
	}
}

func TestPhaseLockStorageStoreFailures(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("%w: secret cause at 10.0.0.5", database.ErrPhaseLockStoreUnavailable), 503, "PHASE_LOCK_STORE_UNAVAILABLE"},
		{errors.New("connection refused 10.0.0.5"), 503, "PHASE_LOCK_STORE_UNAVAILABLE"},
		{fmt.Errorf("%w: check", database.ErrPhaseLockStructureRejected), 422, "PHASE_LOCK_STRUCTURE_INVALID"},
		{database.ErrPhaseLockStructureConflict, 409, "PHASE_LOCK_STRUCTURE_CONFLICT"},
	} {
		store := &phaseLockStoreFake{err: c.err}
		r := newPhaseLockRouter(t, &phaseLockHasherFake{digest: phaseLockTestHash}, store)
		for operation, body := range map[string]string{
			"/structures/record":      phaseLockRecordBody(nil),
			"/relationships/patterns": encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON()}),
			"/patterns/lookup":        encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": phaseLockTestHash}),
			"/patterns/discover":      encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON(), "structure_codexhash": phaseLockTestHash, "authority": "developer_admin"}),
		} {
			rec, payload := r.call(t, operation, body, nil, nil)
			phaseLockCode(t, rec, payload, c.status, c.code)
			if strings.Contains(rec.Body.String(), "10.0.0.5") || strings.Contains(rec.Body.String(), "secret") {
				t.Fatalf("%s leaked the cause: %s", operation, rec.Body.String())
			}
		}
	}
	// No store configured.
	r := newPhaseLockRouter(t, &phaseLockHasherFake{digest: phaseLockTestHash}, nil)
	rec, payload := r.call(t, "/relationships/patterns", encodeJSON(map[string]any{"relationship": phaseLockRelationshipJSON()}), nil, nil)
	phaseLockCode(t, rec, payload, http.StatusServiceUnavailable, "PHASE_LOCK_STORE_UNAVAILABLE")
}

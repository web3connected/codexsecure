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
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/database"
)

const (
	alignmentTestKey  = "svc-alignment-test-key"
	alignmentTestSite = "test-site"
	alignmentTestPath = "/api/v1/sites/test-site/alignment/relationships/record"
	alignmentTestUUID = "e36a777f-5903-4f6b-b0ad-2a4c7ce434f0"
	alignmentTestHash = "ch2:inv-dual/1:512:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" +
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// alignmentStoreFake records every observation it receives and answers with err or a record.
type alignmentStoreFake struct {
	calls   []database.AlignmentObservation
	err     error
	created bool
	count   int64
}

func (s *alignmentStoreFake) RecordAlignment(_ context.Context, obs database.AlignmentObservation) (database.AlignmentRecord, error) {
	s.calls = append(s.calls, obs)
	if s.err != nil {
		return database.AlignmentRecord{}, s.err
	}
	s.count++
	return database.AlignmentRecord{Result: database.AlignmentAligned, Created: s.created, PageMetaUUID: obs.PageMetaUUID,
		Zone: obs.Zone, RouteName: obs.RouteName, HTTPMethod: obs.HTTPMethod, AlignmentCount: s.count}, nil
}

// alignmentSigner issues gateway assertions exactly as codex_gateway's issueAssertion does.
type alignmentSigner struct {
	kid string
	key ed25519.PrivateKey
	rid atomic.Int64
}

func (s *alignmentSigner) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	encode := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	input := encode(map[string]any{"alg": "EdDSA", "kid": s.kid, "typ": "codexsecure-gateway+jwt"}) + "." + encode(claims)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(s.key, []byte(input)))
}

func (s *alignmentSigner) claims(overrides map[string]any) map[string]any {
	now := time.Now().Unix()
	claims := map[string]any{
		"app": "demo-app", "aud": "codexsecure-api", "channel_sites": []string{alignmentTestSite},
		"exp": now + 30, "iat": now, "iss": "codex-gateway", "method": http.MethodPost,
		"path": alignmentTestPath, "rid": fmt.Sprintf("rid-%d", s.rid.Add(1)), "sub": "application:demo-app",
	}
	for name, value := range overrides {
		claims[name] = value
	}
	return claims
}

func alignmentRouter(t *testing.T, store alignmentStore) (*gin.Engine, *alignmentSigner) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SERVICE_API_KEY", alignmentTestKey)
	t.Setenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS", "gw-test:"+base64.RawURLEncoding.EncodeToString(public))
	router := gin.New()
	NewAlignmentHandler(store).RegisterRoutes(router.Group("/api/v1"))
	return router, &alignmentSigner{kid: "gw-test", key: private}
}

func alignmentBody(overrides map[string]any) string {
	body := map[string]any{
		"relationship": map[string]any{"pagemeta_uuid": alignmentTestUUID, "zone": "Z1", "route_name": "pages.home", "http_method": "GET"},
		"evidence":     map[string]any{"codexhash": alignmentTestHash, "tiu": 216832585.431537},
		"developer_intent": map[string]any{"intent": "pages.home", "controller": "App\\Http\\Controllers\\CodexHomeController",
			"action": "__invoke"},
	}
	for name, value := range overrides {
		if value == nil {
			delete(body, name)
			continue
		}
		body[name] = value
	}
	encoded, _ := json.Marshal(body)
	return string(encoded)
}

type alignmentAuth struct {
	key       *string
	assertion *string
	path      string
}

func alignmentCall(router *gin.Engine, auth alignmentAuth, body string) (*httptest.ResponseRecorder, map[string]any) {
	path := auth.path
	if path == "" {
		path = alignmentTestPath
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth.key != nil {
		req.Header.Set("X-Service-Key", *auth.key)
	}
	if auth.assertion != nil {
		req.Header.Set("X-CodexSecure-Gateway-Assertion", *auth.assertion)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec, payload
}

func ptr(value string) *string { return &value }

func (s *alignmentSigner) valid(t *testing.T) alignmentAuth {
	token := s.sign(t, s.claims(nil))
	return alignmentAuth{key: ptr(alignmentTestKey), assertion: &token}
}

// ─── Authentication: service key, then the signed gateway assertion ─────────

func TestAlignmentRecordRequiresTheServiceKey(t *testing.T) {
	store := &alignmentStoreFake{}
	router, signer := alignmentRouter(t, store)
	for name, key := range map[string]*string{"missing": nil, "wrong": ptr("svc-wrong"), "empty": ptr(""), "prefix": ptr(alignmentTestKey[:10])} {
		token := signer.sign(t, signer.claims(nil))
		rec, payload := alignmentCall(router, alignmentAuth{key: key, assertion: &token}, alignmentBody(nil))
		if rec.Code != http.StatusUnauthorized || payload["code"] != "SERVICE_KEY_INVALID" || strings.Contains(rec.Body.String(), alignmentTestKey) {
			t.Fatalf("%s key: status %d, body %s", name, rec.Code, rec.Body.String())
		}
	}
	if len(store.calls) != 0 {
		t.Fatalf("an unauthenticated request reached the store %d times", len(store.calls))
	}
}

func TestAlignmentRecordFailsClosedWithoutAConfiguredKey(t *testing.T) {
	store := &alignmentStoreFake{}
	_, signer := alignmentRouter(t, store)
	t.Setenv("SERVICE_API_KEY", "")
	router := gin.New()
	NewAlignmentHandler(store).RegisterRoutes(router.Group("/api/v1"))
	token := signer.sign(t, signer.claims(nil))
	rec, payload := alignmentCall(router, alignmentAuth{key: ptr("anything"), assertion: &token}, alignmentBody(nil))
	if rec.Code != http.StatusUnauthorized || payload["code"] != "SERVICE_KEY_INVALID" || len(store.calls) != 0 {
		t.Fatalf("SERVICE_API_KEY unset: status %d, body %s, store calls %d", rec.Code, rec.Body.String(), len(store.calls))
	}
}

func TestAlignmentRecordRequiresAValidGatewayAssertion(t *testing.T) {
	store := &alignmentStoreFake{}
	router, signer := alignmentRouter(t, store)
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	other := &alignmentSigner{kid: "gw-test", key: otherKey}
	now := time.Now().Unix()
	cases := map[string]*string{
		"missing":       nil,
		"malformed":     ptr("not.a.jws"),
		"unknown key":   ptr(other.sign(t, signer.claims(nil))),
		"expired":       ptr(signer.sign(t, signer.claims(map[string]any{"iat": now - 120, "exp": now - 90}))),
		"other path":    ptr(signer.sign(t, signer.claims(map[string]any{"path": "/api/v1/sites/test-site/channels"}))),
		"other method":  ptr(signer.sign(t, signer.claims(map[string]any{"method": http.MethodGet}))),
		"other site":    ptr(signer.sign(t, signer.claims(map[string]any{"channel_sites": []string{"other-site"}}))),
		"read grant":    ptr(signer.sign(t, signer.claims(map[string]any{"channel_sites": nil, "read_sites": []string{alignmentTestSite}}))),
		"wrong subject": ptr(signer.sign(t, signer.claims(map[string]any{"sub": "application:other-app"}))),
	}
	for name, assertion := range cases {
		rec, payload := alignmentCall(router, alignmentAuth{key: ptr(alignmentTestKey), assertion: assertion}, alignmentBody(nil))
		if rec.Code != http.StatusForbidden || payload["code"] != "ADMIN_UNAUTHORIZED" {
			t.Fatalf("%s assertion: status %d, body %s", name, rec.Code, rec.Body.String())
		}
	}
	// One assertion authorizes one request: a replay is refused.
	token := signer.sign(t, signer.claims(nil))
	if rec, _ := alignmentCall(router, alignmentAuth{key: ptr(alignmentTestKey), assertion: &token}, alignmentBody(nil)); rec.Code != http.StatusOK {
		t.Fatalf("first use: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec, payload := alignmentCall(router, alignmentAuth{key: ptr(alignmentTestKey), assertion: &token}, alignmentBody(nil)); rec.Code != http.StatusForbidden || payload["code"] != "ADMIN_UNAUTHORIZED" {
		t.Fatalf("replay: status %d, body %s", rec.Code, rec.Body.String())
	}
	if len(store.calls) != 1 {
		t.Fatalf("only the one authenticated request may reach the store, got %d", len(store.calls))
	}
}

// ─── Recording ──────────────────────────────────────────────────────────────

func TestAlignmentRecordRecordsAVerifiedObservation(t *testing.T) {
	store := &alignmentStoreFake{created: true}
	router, signer := alignmentRouter(t, store)
	rec, payload := alignmentCall(router, signer.valid(t), alignmentBody(nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("first observation: status %d, body %s", rec.Code, rec.Body.String())
	}
	data, _ := payload["data"].(map[string]any)
	relationship, _ := data["relationship"].(map[string]any)
	if data["recorded"] != true || data["result"] != "aligned" || data["created"] != true || relationship["pagemeta_uuid"] != alignmentTestUUID ||
		relationship["alignment_count"] != float64(1) || relationship["misalignment_count"] != float64(0) {
		t.Fatalf("first observation body: %s", rec.Body.String())
	}
	obs := store.calls[0]
	if obs.PageMetaUUID != alignmentTestUUID || obs.Zone != "Z1" || *obs.RouteName != "pages.home" || obs.HTTPMethod != "GET" ||
		obs.CodexHash != alignmentTestHash || obs.TIU != 216832585.431537 || *obs.Intent != "pages.home" || *obs.Action != "__invoke" {
		t.Fatalf("observation passed to the store: %+v", obs)
	}

	store.created = false
	rec, payload = alignmentCall(router, signer.valid(t), alignmentBody(map[string]any{"developer_intent": nil}))
	data, _ = payload["data"].(map[string]any)
	relationship, _ = data["relationship"].(map[string]any)
	if rec.Code != http.StatusOK || data["created"] != false || relationship["alignment_count"] != float64(2) {
		t.Fatalf("second observation: status %d, body %s", rec.Code, rec.Body.String())
	}
	if store.calls[1].Intent != nil || store.calls[1].Controller != nil {
		t.Fatal("developer intent is optional and must not be invented")
	}
}

func TestAlignmentRecordMapsStoreRefusals(t *testing.T) {
	for err, want := range map[error]struct {
		status int
		code   string
	}{
		database.ErrAlignmentPageMetaNotFound:                         {http.StatusNotFound, "ALIGNMENT_PAGEMETA_NOT_FOUND"},
		database.ErrAlignmentEvidenceMismatch:                         {http.StatusConflict, "ALIGNMENT_EVIDENCE_MISMATCH"},
		database.ErrAlignmentRelationshipConflict:                     {http.StatusConflict, "ALIGNMENT_RELATIONSHIP_CONFLICT"},
		&database.AlignmentMismatch{Field: "relationship.zone"}:       {http.StatusConflict, "ALIGNMENT_RELATIONSHIP_MISMATCH"},
		&database.AlignmentMismatch{Field: "relationship.route_name"}: {http.StatusConflict, "ALIGNMENT_RELATIONSHIP_MISMATCH"},
		database.ErrAlignmentStoreUnavailable:                         {http.StatusServiceUnavailable, "ALIGNMENT_STORE_UNAVAILABLE"},
		errors.New("connection refused to 10.0.0.5"):                  {http.StatusServiceUnavailable, "ALIGNMENT_STORE_UNAVAILABLE"},
	} {
		store := &alignmentStoreFake{err: err}
		router, signer := alignmentRouter(t, store)
		rec, payload := alignmentCall(router, signer.valid(t), alignmentBody(nil))
		if rec.Code != want.status || payload["code"] != want.code || payload["data"] != nil || strings.Contains(rec.Body.String(), "10.0.0.5") {
			t.Fatalf("%v: status %d, body %s", err, rec.Code, rec.Body.String())
		}
	}
}

// ─── Contract ───────────────────────────────────────────────────────────────

func TestAlignmentRecordRefusesVisitorDataAndMalformedRequests(t *testing.T) {
	store := &alignmentStoreFake{}
	router, signer := alignmentRouter(t, store)
	for name, body := range map[string]string{
		"ip address": alignmentBody(map[string]any{"ip": "203.0.113.9"}),
		"user agent": alignmentBody(map[string]any{"user_agent": "Visitor/1"}),
		"session":    alignmentBody(map[string]any{"session_id": "abc"}),
		"user":       alignmentBody(map[string]any{"user_id": 7}),
		"nested extra": alignmentBody(map[string]any{"relationship": map[string]any{"pagemeta_uuid": alignmentTestUUID, "zone": "Z1",
			"route_name": "pages.home", "http_method": "GET", "fingerprint": "x"}}),
		"not json":    "relationship=1",
		"array":       "[]",
		"two objects": alignmentBody(nil) + alignmentBody(nil),
		"too large":   `{"relationship":"` + strings.Repeat("a", 9000) + `"}`,
	} {
		rec, payload := alignmentCall(router, signer.valid(t), body)
		if rec.Code != http.StatusBadRequest || payload["code"] != "ALIGNMENT_REQUEST_INVALID" {
			t.Fatalf("%s: status %d, body %s", name, rec.Code, rec.Body.String())
		}
	}
	if len(store.calls) != 0 {
		t.Fatalf("a refused request reached the store %d times", len(store.calls))
	}
}

func TestAlignmentRecordValidatesTheRelationshipContract(t *testing.T) {
	store := &alignmentStoreFake{}
	router, signer := alignmentRouter(t, store)
	relationship := func(field string, value any) map[string]any {
		r := map[string]any{"pagemeta_uuid": alignmentTestUUID, "zone": "Z1", "route_name": "pages.home", "http_method": "GET"}
		r[field] = value
		return map[string]any{"relationship": r}
	}
	evidence := func(field string, value any) map[string]any {
		e := map[string]any{"codexhash": alignmentTestHash, "tiu": 216832585.431537}
		e[field] = value
		return map[string]any{"evidence": e}
	}
	for name, overrides := range map[string]map[string]any{
		"relationship.pagemeta_uuid": relationship("pagemeta_uuid", "not-a-uuid"),
		"relationship.zone":          relationship("zone", "Z13"),
		"relationship.http_method":   relationship("http_method", "POST"),
		"relationship.route_name":    relationship("route_name", strings.Repeat("r", 256)),
		"relationship":               {"relationship": nil},
		"evidence.codexhash":         evidence("codexhash", strings.Repeat("a", 128)),
		"evidence.tiu":               evidence("tiu", 0),
		"evidence":                   {"evidence": nil},
	} {
		rec, payload := alignmentCall(router, signer.valid(t), alignmentBody(overrides))
		if rec.Code != http.StatusUnprocessableEntity || payload["code"] != "ALIGNMENT_RELATIONSHIP_INVALID" || payload["error"] != name+" is invalid" {
			t.Fatalf("%s: status %d, body %s", name, rec.Code, rec.Body.String())
		}
	}
	// Step 3B records Z1 only; a TIU must fit numeric(20,6); canonical UUIDs only.
	for name, overrides := range map[string]map[string]any{
		"zone Z12":       relationship("zone", "Z12"),
		"tiu too large":  evidence("tiu", 1e14),
		"nil uuid":       relationship("pagemeta_uuid", "00000000-0000-0000-0000-000000000000"),
		"uppercase uuid": relationship("pagemeta_uuid", strings.ToUpper(alignmentTestUUID)),
	} {
		if rec, _ := alignmentCall(router, signer.valid(t), alignmentBody(overrides)); rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: status %d", name, rec.Code)
		}
	}
	if len(store.calls) != 0 {
		t.Fatalf("an invalid request reached the store %d times", len(store.calls))
	}
}

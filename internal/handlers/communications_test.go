package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

type communicationStub struct {
	calls    int
	decision communications.Decision
	failure  error
}

func (s *communicationStub) Find(context.Context, string, string) (communications.Configuration, error) {
	s.calls++
	return communications.Configuration{Registration: communications.Registration{Key: "d9fe68fe-b0dc-4aca-b625-56c3514479f2"}, Structure: communications.Structure{State: "valid", Revision: 1, Issues: []string{}}, ApprovalStatus: "unapproved", Effective: "unapproved", History: []communications.Event{}, Observations: []communications.ObservedAssessment{}}, s.failure
}
func (s *communicationStub) History(context.Context, string, string) (communications.History, error) {
	s.calls++
	return communications.History{History: []communications.Event{}, Revisions: []communications.Revision{}}, s.failure
}
func (s *communicationStub) Decide(_ context.Context, d communications.Decision) (json.RawMessage, error) {
	s.calls++
	s.decision = d
	result := `{"revision":1}`
	switch d.Action {
	case "draft":
		result = `{"draft_version":1,"based_on_revision":null}`
	case "approve":
		result = `{"revision":1,"structure_revision":1}`
	case "disable":
		result = `{"approval_status":"disabled"}`
	case "revoke":
		result = `{"approval_status":"revoked"}`
	}
	return json.RawMessage(result), s.failure
}
func (s *communicationStub) Discover(context.Context, string, string, communications.Discovery) (communications.DiscoveryResult, error) {
	s.calls++
	return communications.DiscoveryResult{}, s.failure
}
func (s *communicationStub) Assess(context.Context, string, communications.Coverage) (communications.Assessment, error) {
	s.calls++
	return communications.Assessment{Complete: false, Communications: []communications.Configuration{}}, s.failure
}
func communicationRouter(t *testing.T, s communications.Store, principal bool) *gin.Engine {
	t.Helper()
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if principal {
		r.Use(func(c *gin.Context) {
			middleware.SetCommunicationPrincipal(c, middleware.CommunicationPrincipal{GatewayAuthenticated: true, Actor: "admin-1", AdminSites: map[string]bool{"test-site": true}, DiscoverySites: map[string]bool{"test-site": true}})
		})
	}
	NewCommunicationHandler(s).RegisterRoutes(r.Group("/api/v1"))
	return r
}
func communicationRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", "test-service-key")
	req.Header.Set("X-CodexSecure-Actor", "admin-1")
	req.Header.Set("Idempotency-Key", "d9fe68fe-b0dc-4aca-b625-56c3514479f2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const configurationBase = "/api/v1/sites/test-site/communications/d9fe68fe-b0dc-4aca-b625-56c3514479f2"

func TestCommunicationSDKHTTPRouteDispatch(t *testing.T) {
	for _, tc := range []struct{ method, path, body, action string }{
		{"GET", "/configuration", "", ""}, {"GET", "/configuration/history", "", ""},
		{"PUT", "/configuration/draft", `{"document":{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true},"expected_version":0}`, "draft"},
		{"POST", "/configuration/approve", `{"draft_version":1,"structure_revision":2,"reason":null}`, "approve"},
		{"POST", "/configuration/reject", `{"draft_version":1,"reason":"review"}`, "reject"},
		{"POST", "/disable", `{"reason":null}`, "disable"}, {"POST", "/revoke", `{}`, "revoke"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s := &communicationStub{}
			r := communicationRouter(t, s, true)
			w := communicationRequest(r, tc.method, configurationBase+tc.path, tc.body)
			if w.Code != 200 || s.calls != 1 {
				t.Fatalf("status %d body %s", w.Code, w.Body.String())
			}
			var envelope map[string]json.RawMessage
			if e := json.Unmarshal(w.Body.Bytes(), &envelope); e != nil || len(envelope["data"]) == 0 {
				t.Fatal(w.Body.String())
			}
			if tc.action != "" && (s.decision.Action != tc.action || s.decision.Actor != "admin-1" || s.decision.Site != "test-site") {
				t.Fatal(s.decision)
			}
		})
	}
}
func TestCommunicationMalformedBodiesNeverReachStore(t *testing.T) {
	s := &communicationStub{}
	r := communicationRouter(t, s, true)
	for _, body := range []string{`{`, `null`, ` null `, `[]`, `{} {}`, `{"reason":1}`, `{"reason":"a","reason":"b"}`, `{"reason":null,"approval_status":"approved"}`, strings.Repeat(" ", 65537)} {
		w := communicationRequest(r, "POST", configurationBase+"/disable", body)
		if w.Code < 400 || s.calls != 0 {
			t.Fatalf("accepted malformed input: %d %s", w.Code, w.Body.String())
		}
	}
}
func TestCommunicationHTTPAuthority(t *testing.T) {
	s := &communicationStub{}
	r := communicationRouter(t, s, false)
	for _, apiKey := range []string{"", "invalid", "untrusted-client-claim"} {
		req := httptest.NewRequest("POST", configurationBase+"/disable", strings.NewReader(`{}`))
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("X-Service-Key", "test-service-key")
		req.Header.Set("X-CodexSecure-Actor", "admin-1")
		req.Header.Set("Idempotency-Key", "d9fe68fe-b0dc-4aca-b625-56c3514479f2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 || s.calls != 0 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestCommunicationHTTPErrorEnvelope(t *testing.T) {
	s := &communicationStub{failure: communications.APIError{Status: 409, Code: "STALE_REVISION", Message: "revision changed"}}
	r := communicationRouter(t, s, true)
	w := communicationRequest(r, "POST", configurationBase+"/disable", `{}`)
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"code":"STALE_REVISION"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestCommunicationAssessmentAndDiscoveryRoutes(t *testing.T) {
	s := &communicationStub{}
	r := communicationRouter(t, s, true)
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/api/v1/sites/test-site/communications/assessments?scope=page&page=https%3A%2F%2Fexample.test%2F", "", 200},
		{"GET", "/api/v1/sites/test-site/communications/assessments?scope=site&scope=page", "", 422},
		{"POST", "/api/v1/sites/test-site/communications/discovery", `{}`, 200},
		{"GET", "/api/v1/sites/test-site/communications/not-a-uuid/configuration", "", 422},
	} {
		w := communicationRequest(r, tc.method, tc.path, tc.body)
		if w.Code != tc.want {
			t.Fatal(tc.path, w.Code, w.Body.String())
		}
	}
}

func TestAssessmentProcessingFailureKeepsIncompleteData(t *testing.T) {
	s := &communicationStub{failure: errors.New("private database connection details")}
	r := communicationRouter(t, s, true)
	w := communicationRequest(r, "GET", "/api/v1/sites/test-site/communications/assessments?scope=site", "")
	if w.Code != 500 || strings.Contains(w.Body.String(), "private database") {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data communications.Assessment
		Code string
	}
	if e := json.Unmarshal(w.Body.Bytes(), &response); e != nil || response.Data.Complete || response.Code != "INTERNAL_ERROR" {
		t.Fatal(response, e)
	}
}

func (s *communicationStub) StartDiscovery(_ context.Context, site, key, actor string, d communications.DiscoveryStart) (json.RawMessage, error) {
	s.calls++
	s.decision = communications.Decision{Site: site, IdempotencyKey: key, Actor: actor}
	return json.RawMessage(`{"status":"open","complete":false}`), s.failure
}
func (s *communicationStub) RecordDiscoveryFailure(context.Context, string, string, communications.DiscoveryFailure) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"outcome":"rejected"}`), s.failure
}
func (s *communicationStub) FinalizeDiscovery(_ context.Context, site, key, actor string, d communications.DiscoveryFinalize) (json.RawMessage, error) {
	s.calls++
	s.decision = communications.Decision{Site: site, Key: key, Actor: actor}
	return json.RawMessage(`{"status":"complete","complete":true}`), s.failure
}

func TestDiscoveryBatchHTTPAuthorityAndDecoding(t *testing.T) {
	for _, route := range []struct{ suffix, body string }{
		{"", `{"scope":"site","expected_pages":["home"]}`},
		{"/d9fe68fe-b0dc-4aca-b625-56c3514479f2/failures", `{"sequence":1,"outcome":"rejected","code":"INVALID"}`},
		{"/d9fe68fe-b0dc-4aca-b625-56c3514479f2/finalize", `{"outcome":"complete","expected_observations":0}`},
	} {
		for _, tc := range []struct {
			name, site, key, replay, body string
			trusted, discovery, admin     bool
			want                          int
		}{
			{name: "discovery service", site: "test-site", key: "service", replay: "d9fe68fe-b0dc-4aca-b625-56c3514479f2", body: route.body, trusted: true, discovery: true, want: 200},
			{name: "admin grant insufficient", site: "test-site", key: "service", replay: "d9fe68fe-b0dc-4aca-b625-56c3514479f2", body: route.body, trusted: true, admin: true, want: 403},
			{name: "missing service key", site: "test-site", body: route.body, want: 401},
			{name: "unverified principal", site: "test-site", key: "service", body: route.body, discovery: true, want: 403},
			{name: "wrong site", site: "other-site", key: "service", body: route.body, trusted: true, discovery: true, want: 403},
			{name: "missing replay", site: "test-site", key: "service", body: route.body, trusted: true, discovery: true, want: 422},
			{name: "unknown field", site: "test-site", key: "service", replay: "d9fe68fe-b0dc-4aca-b625-56c3514479f2", body: `{"unexpected":true}`, trusted: true, discovery: true, want: 400},
		} {
			t.Run(route.suffix+"/"+tc.name, func(t *testing.T) {
				t.Setenv("SERVICE_API_KEY", "service")
				store := &communicationStub{}
				r := gin.New()
				r.Use(func(c *gin.Context) {
					middleware.SetCommunicationPrincipal(c, middleware.CommunicationPrincipal{GatewayAuthenticated: tc.trusted, Actor: "scanner", DiscoverySites: map[string]bool{"test-site": tc.discovery}, AdminSites: map[string]bool{"test-site": tc.admin}})
				})
				NewCommunicationHandler(store).RegisterRoutes(r.Group("/api/v1"))
				req := httptest.NewRequest("POST", "/api/v1/sites/"+tc.site+"/communications/discovery/batches"+route.suffix, strings.NewReader(tc.body))
				req.Header.Set("X-Service-Key", tc.key)
				req.Header.Set("Idempotency-Key", tc.replay)
				req.Header.Set("X-CodexSecure-Actor", "forged")
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != tc.want {
					t.Fatal(w.Code, w.Body.String())
				}
				if tc.want != 200 && store.calls != 0 {
					t.Fatal("unauthorized or malformed request reached store")
				}
				if tc.want == 200 && route.suffix != "/d9fe68fe-b0dc-4aca-b625-56c3514479f2/failures" && store.decision.Actor != "scanner" {
					t.Fatal("unverified actor used", store.decision)
				}
			})
		}
	}
}

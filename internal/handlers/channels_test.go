package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/web3connected/codexsecure-api/internal/channels"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

const channelID = "3f0e1c2a-8b4d-4e6f-9a1b-2c3d4e5f6a7b"

type channelStub struct {
	calls   int
	site    string
	zone    string
	channel uuid.UUID
	user    uuid.UUID
	result  channels.Result
	err     error
}

func (s *channelStub) record(site, zone string, channel uuid.UUID) (channels.Result, error) {
	s.calls++
	s.site, s.zone, s.channel = site, zone, channel
	return s.result, s.err
}
func (s *channelStub) Open(_ context.Context, site, zone string) (channels.Result, error) {
	return s.record(site, zone, uuid.Nil)
}
func (s *channelStub) Get(_ context.Context, site string, channel uuid.UUID) (channels.Result, error) {
	return s.record(site, "", channel)
}
func (s *channelStub) Close(_ context.Context, site string, channel uuid.UUID) (channels.Result, error) {
	return s.record(site, "", channel)
}
func (s *channelStub) Activate(_ context.Context, site string, channel uuid.UUID) (channels.Result, error) {
	return s.record(site, "activate", channel)
}
func (s *channelStub) Renew(_ context.Context, site string, channel uuid.UUID) (channels.Result, error) {
	return s.record(site, "renew", channel)
}
func (s *channelStub) BindIdentity(_ context.Context, site string, channel, user uuid.UUID) (channels.Result, error) {
	s.user = user
	return s.record(site, "bind", channel)
}

func stubChannel(status string) *channels.Channel {
	now := time.Date(2026, 10, 27, 14, 0, 0, 0, time.UTC)
	c := &channels.Channel{ID: uuid.MustParse(channelID), Site: "test-site", Status: status, Zone: "Z1", OpenedAt: now, LastActivityAt: now, ExpiresAt: now.Add(30 * time.Minute), Protection: channels.Protection{Status: "not_protected"}}
	if status == channels.StatusClosed {
		c.ClosedAt = &now
	}
	return c
}
func channelRouter(t *testing.T, s *channelStub, p *middleware.CommunicationPrincipal) *gin.Engine {
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if p != nil {
		r.Use(func(c *gin.Context) { middleware.SetCommunicationPrincipal(c, *p) })
	}
	NewChannelHandler(s).RegisterRoutes(r.Group("/api/v1"))
	return r
}
func channelPrincipal() *middleware.CommunicationPrincipal {
	return &middleware.CommunicationPrincipal{GatewayAuthenticated: true, Actor: "app", ChannelSites: map[string]bool{"test-site": true}}
}

// Channel requests carry neither X-CodexSecure-Actor nor Idempotency-Key.
func channelRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", "test-service-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e, w.Body.String())
	}
	return out
}

const channelsBase = "/api/v1/sites/test-site/channels"

func TestChannelAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name      string
		key       string
		principal *middleware.CommunicationPrincipal
		want      int
		code      string
	}{
		{"missing service key", "", channelPrincipal(), 401, "SERVICE_KEY_INVALID"},
		{"wrong service key", "wrong", channelPrincipal(), 401, "SERVICE_KEY_INVALID"},
		{"no gateway principal", "test-service-key", nil, 403, "ADMIN_UNAUTHORIZED"},
		{"unauthenticated principal", "test-service-key", &middleware.CommunicationPrincipal{ChannelSites: map[string]bool{"test-site": true}}, 403, "ADMIN_UNAUTHORIZED"},
		{"other site granted", "test-service-key", &middleware.CommunicationPrincipal{GatewayAuthenticated: true, ChannelSites: map[string]bool{"other-site": true}}, 403, "ADMIN_UNAUTHORIZED"},
		{"admin and read grants are not channel grants", "test-service-key", &middleware.CommunicationPrincipal{GatewayAuthenticated: true, Actor: "admin-1", ReadSites: map[string]bool{"test-site": true}, AdminSites: map[string]bool{"test-site": true}, DiscoverySites: map[string]bool{"test-site": true}}, 403, "ADMIN_UNAUTHORIZED"},
	} {
		for _, route := range []struct{ method, path, body string }{
			{"POST", channelsBase, `{"zone":"Z1"}`},
			{"GET", channelsBase + "/" + channelID, ""},
			{"POST", channelsBase + "/" + channelID + "/close", ""},
			{"POST", channelsBase + "/" + channelID + "/activate", ""},
			{"POST", channelsBase + "/" + channelID + "/renew", ""},
			{"POST", channelsBase + "/" + channelID + "/bind-identity", `{"identity":{"type":"authenticated","user_uuid":"7c81b7f2-9f54-4c12-8a5b-6d203e7f4190"}}`},
		} {
			t.Run(tc.name+" "+route.method+" "+route.path, func(t *testing.T) {
				s := &channelStub{result: channels.Result{Outcome: channels.Opened, Channel: stubChannel("pending")}}
				r := channelRouter(t, s, tc.principal)
				req := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
				req.Header.Set("X-Service-Key", tc.key)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != tc.want || decodeEnvelope(t, w)["code"] != tc.code {
					t.Fatal(w.Code, w.Body.String())
				}
				if s.calls != 0 {
					t.Fatal("unauthorized request reached the manager")
				}
			})
		}
	}
	t.Run("missing SERVICE_API_KEY denies", func(t *testing.T) {
		s := &channelStub{}
		t.Setenv("SERVICE_API_KEY", "")
		r := gin.New()
		r.Use(func(c *gin.Context) { middleware.SetCommunicationPrincipal(c, *channelPrincipal()) })
		NewChannelHandler(s).RegisterRoutes(r.Group("/api/v1"))
		req := httptest.NewRequest("GET", channelsBase+"/"+channelID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 401 || s.calls != 0 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
}

func TestChannelOpen(t *testing.T) {
	s := &channelStub{result: channels.Result{Outcome: channels.Opened, Channel: stubChannel("pending")}}
	r := channelRouter(t, s, channelPrincipal())
	w := channelRequest(r, "POST", channelsBase, `{"zone":"Z3"}`)
	if w.Code != 201 || s.site != "test-site" || s.zone != "Z3" {
		t.Fatal(w.Code, w.Body.String(), s)
	}
	data := decodeEnvelope(t, w)["data"].(map[string]any)
	channel := data["channel"].(map[string]any)
	if data["result"] != "opened" || channel["status"] != "pending" || channel["channel"] != channelID || channel["protection"].(map[string]any)["status"] != "not_protected" {
		t.Fatal(w.Body.String())
	}
	for _, body := range []string{``, `null`, `[]`, `"Z1"`, `{"zone":"Z1","channel":"` + channelID + `"}`, `{"zone":"Z1","lifetime_seconds":60}`, `{"zone":"Z1","expires_at":"2030-01-01T00:00:00Z"}`, `{"zone":"Z1","zone":"Z2"}`, `{"zone":1}`, `{"zone":"Z1"} {}`} {
		s.calls = 0
		if w := channelRequest(r, "POST", channelsBase, body); w.Code != 400 || decodeEnvelope(t, w)["code"] != "CHANNEL_REQUEST_INVALID" || s.calls != 0 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	if w := channelRequest(r, "POST", channelsBase, `{}`); w.Code != 422 || s.calls != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := channelRequest(r, "POST", channelsBase, `{"zone":"`+strings.Repeat("Z", 5000)+`"}`); w.Code != 413 || s.calls != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestChannelIdentifiers(t *testing.T) {
	s := &channelStub{result: channels.Result{Outcome: channels.Found, Channel: stubChannel("pending")}}
	r := channelRouter(t, s, channelPrincipal())
	for _, id := range []string{"not-a-uuid", "00000000-0000-0000-0000-000000000000", "{" + channelID + "}", "urn:uuid:" + channelID, strings.ReplaceAll(channelID, "-", ""), channelID + "0"} {
		for _, route := range []struct{ method, suffix string }{{"GET", ""}, {"POST", "/close"}, {"POST", "/activate"}} {
			w := channelRequest(r, route.method, channelsBase+"/"+id+route.suffix, "")
			if w.Code != 422 || decodeEnvelope(t, w)["code"] != "CHANNEL_IDENTIFIER_INVALID" || s.calls != 0 {
				t.Fatal(id, route, w.Code, w.Body.String())
			}
		}
	}
	if w := channelRequest(r, "GET", channelsBase+"/"+strings.ToUpper(channelID), ""); w.Code != 200 || s.channel.String() != channelID {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestChannelOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, method, suffix string
		result               channels.Result
		want                 int
		code                 string
		data                 bool
	}{
		{"get found", "GET", "", channels.Result{Outcome: channels.Found, Channel: stubChannel("pending")}, 200, "", true},
		{"get expired is found", "GET", "", channels.Result{Outcome: channels.Found, Channel: stubChannel("expired")}, 200, "", true},
		{"get not found", "GET", "", channels.Result{Outcome: channels.NotFound}, 404, "CHANNEL_NOT_FOUND", false},
		{"close", "POST", "/close", channels.Result{Outcome: channels.Closed, Channel: stubChannel("closed")}, 200, "", true},
		{"close already closed", "POST", "/close", channels.Result{Outcome: channels.AlreadyClosed, Channel: stubChannel("closed")}, 409, "CHANNEL_CLOSED", true},
		{"close expired", "POST", "/close", channels.Result{Outcome: channels.Expired, Channel: stubChannel("expired")}, 409, "CHANNEL_EXPIRED", true},
		{"close not found", "POST", "/close", channels.Result{Outcome: channels.NotFound}, 404, "CHANNEL_NOT_FOUND", false},
		{"activate", "POST", "/activate", channels.Result{Outcome: channels.Activated, Channel: stubChannel("active")}, 200, "", true},
		{"activate repeated is idempotent", "POST", "/activate", channels.Result{Outcome: channels.AlreadyActive, Channel: stubChannel("active")}, 200, "", true},
		{"activate closed", "POST", "/activate", channels.Result{Outcome: channels.Closed, Channel: stubChannel("closed")}, 409, "CHANNEL_CLOSED", true},
		{"activate expired", "POST", "/activate", channels.Result{Outcome: channels.Expired, Channel: stubChannel("expired")}, 409, "CHANNEL_EXPIRED", true},
		{"activate not found", "POST", "/activate", channels.Result{Outcome: channels.NotFound}, 404, "CHANNEL_NOT_FOUND", false},
		{"activate unexpected outcome", "POST", "/activate", channels.Result{Outcome: channels.ProtectionMismatch, Channel: stubChannel("pending")}, 500, "INTERNAL_ERROR", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &channelStub{result: tc.result}
			w := channelRequest(channelRouter(t, s, channelPrincipal()), tc.method, channelsBase+"/"+channelID+tc.suffix, "")
			out := decodeEnvelope(t, w)
			if w.Code != tc.want || (tc.code != "" && out["code"] != tc.code) || (out["data"] != nil) != tc.data {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	t.Run("activate body", func(t *testing.T) {
		s := &channelStub{result: channels.Result{Outcome: channels.Activated, Channel: stubChannel("active")}}
		r := channelRouter(t, s, channelPrincipal())
		for _, body := range []string{"", " ", "{}"} {
			if w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/activate", body); w.Code != 200 || s.zone != "activate" || s.site != "test-site" || s.channel.String() != channelID {
				t.Fatal(body, w.Code, w.Body.String())
			}
		}
		s.calls = 0
		// Protection is never accepted from a client.
		for _, body := range []string{`{"codexhash":"` + strings.Repeat("a", 128) + `"}`, `{"tiu":"1.000000"}`, `{"tiu_scheme":"codextime-tiu-2020/1"}`, `{"site":"other"}`, `[]`, `null`, `x`, `{} {}`} {
			if w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/activate", body); w.Code != 400 || decodeEnvelope(t, w)["code"] != "CHANNEL_REQUEST_INVALID" || s.calls != 0 {
				t.Fatal(body, w.Code, w.Body.String())
			}
		}
		if w := channelRequest(r, "GET", channelsBase+"/"+channelID+"/activate", ""); w.Code != 404 || s.calls != 0 {
			t.Fatal("GET activate", w.Code)
		}
	})
	t.Run("close body", func(t *testing.T) {
		s := &channelStub{result: channels.Result{Outcome: channels.Closed, Channel: stubChannel("closed")}}
		r := channelRouter(t, s, channelPrincipal())
		for _, body := range []string{"", " ", "{}"} {
			if w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/close", body); w.Code != 200 {
				t.Fatal(body, w.Code, w.Body.String())
			}
		}
		s.calls = 0
		for _, body := range []string{`{"reason":"x"}`, `[]`, `null`, `x`} {
			if w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/close", body); w.Code != 400 || s.calls != 0 {
				t.Fatal(body, w.Code, w.Body.String())
			}
		}
	})
}

func TestChannelErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
		code string
	}{
		{"site not found", &pgconn.PgError{Code: "CX404", Message: "Site not found"}, 404, "SITE_NOT_FOUND"},
		{"invalid request", &pgconn.PgError{Code: "CX422", Message: "Channel request is invalid"}, 422, "CHANNEL_REQUEST_INVALID"},
		{"guard conflict is internal", &pgconn.PgError{Code: "CX409", Message: "Invalid channel lifecycle transition"}, 500, "INTERNAL_ERROR"},
		{"constraint is internal", &pgconn.PgError{Code: "23514", Message: "violates check constraint", Detail: "secret detail"}, 500, "INTERNAL_ERROR"},
		{"privilege is internal", &pgconn.PgError{Code: "42501", Message: "permission denied for table communication_channels"}, 500, "INTERNAL_ERROR"},
		{"connection", errors.New("dial tcp 10.0.0.1:5432: connection refused"), 500, "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &channelStub{err: tc.err}
			w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", channelsBase, `{"zone":"Z1"}`)
			if w.Code != tc.want || decodeEnvelope(t, w)["code"] != tc.code {
				t.Fatal(w.Code, w.Body.String())
			}
			for _, leak := range []string{"communication_channels", "secret", "10.0.0.1", "constraint", "transition"} {
				if strings.Contains(w.Body.String(), leak) {
					t.Fatal("database detail leaked", w.Body.String())
				}
			}
		})
	}
}

// Expiry is manager-internal: no route may reach it. Activation and renewal (C10)
// have their own routes and nothing else changes protection or the deadline.
func TestChannelNoExpiryRoutes(t *testing.T) {
	s := &channelStub{}
	r := channelRouter(t, s, channelPrincipal())
	for _, route := range []struct{ method, path string }{
		{"POST", channelsBase + "/" + channelID + "/expire"},
		{"POST", channelsBase + "/" + channelID + "/protect"},
		{"PUT", channelsBase + "/" + channelID},
		{"PATCH", channelsBase + "/" + channelID},
		{"DELETE", channelsBase + "/" + channelID},
		{"GET", channelsBase},
	} {
		if w := channelRequest(r, route.method, route.path, `{}`); w.Code != 404 || s.calls != 0 {
			t.Fatal(route, w.Code, w.Body.String())
		}
	}
}

// LoadAPIRoutes registers both route sets on the same v1 group under /sites/:site.
func TestChannelAndCommunicationRoutesCoexist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	NewCommunicationHandler(&communicationStub{}).RegisterRoutes(v1)
	NewChannelHandler(&channelStub{}).RegisterRoutes(v1)
	routes := map[string]bool{}
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{"POST /api/v1/sites/:site/channels", "GET /api/v1/sites/:site/channels/:channel", "POST /api/v1/sites/:site/channels/:channel/close", "POST /api/v1/sites/:site/channels/:channel/activate", "POST /api/v1/sites/:site/channels/:channel/renew", "GET /api/v1/sites/:site/communications/:key/configuration"} {
		if !routes[want] {
			t.Fatal("missing route", want)
		}
	}
}

// Activation failures are structured and carry no service or database detail.
func TestChannelActivateErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
		code string
	}{
		{"codexhash unavailable", fmt.Errorf("%w: http: dial tcp 10.0.0.1:8001: connection refused", channels.ErrCodexHashUnavailable), 503, "CODEXHASH_UNAVAILABLE"},
		{"codexhash invalid", fmt.Errorf("%w: algo \"sha256\"", channels.ErrCodexHashInvalid), 502, "CODEXHASH_INVALID_RESPONSE"},
		{"codextime unavailable", fmt.Errorf("%w: codextime status 500", channels.ErrCodexTimeUnavailable), 503, "CODEXTIME_UNAVAILABLE"},
		{"codextime invalid", fmt.Errorf("%w: secret detail", channels.ErrCodexTimeInvalid), 502, "CODEXTIME_INVALID_RESPONSE"},
		{"database", &pgconn.PgError{Code: "CX409", Message: "Channel protection does not match the channel", Detail: "secret detail"}, 500, "INTERNAL_ERROR"},
		{"identifier", channels.ErrIdentifierInvalid, 422, "CHANNEL_IDENTIFIER_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &channelStub{err: tc.err}
			w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", channelsBase+"/"+channelID+"/activate", "")
			out := decodeEnvelope(t, w)
			if w.Code != tc.want || out["code"] != tc.code || out["data"] != nil {
				t.Fatal(w.Code, w.Body.String())
			}
			for _, leak := range []string{"10.0.0.1", "secret", "sha256", "status 500", "protection does not match"} {
				if strings.Contains(w.Body.String(), leak) {
					t.Fatal("detail leaked", w.Body.String())
				}
			}
		})
	}
}

// Client headers cannot stand in for the trusted gateway principal.
func TestChannelActivateHeadersCannotImpersonateGateway(t *testing.T) {
	s := &channelStub{result: channels.Result{Outcome: channels.Activated, Channel: stubChannel("active")}}
	r := channelRouter(t, s, nil)
	req := httptest.NewRequest("POST", channelsBase+"/"+channelID+"/activate", nil)
	req.Header.Set("X-Service-Key", "test-service-key")
	for header, value := range map[string]string{
		"X-Gateway": "true", "X-CodexSecure-Actor": "admin-1", "X-Codex-Zone": "Z1", "X-Forwarded-For": "127.0.0.1",
		"X-Api-Key": "w3c_app_test", "Authorization": "Bearer gateway", "X-CodexSecure-Principal": `{"gateway_authenticated":true,"channel_sites":{"test-site":true}}`,
	} {
		req.Header.Set(header, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 || decodeEnvelope(t, w)["code"] != "ADMIN_UNAUTHORIZED" || s.calls != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
}

// Renew (C10): only a renewed result succeeds; every other lifecycle result keeps
// its existing code, with the channel's state when there is one.
func TestChannelRenew(t *testing.T) {
	active := stubChannel(channels.StatusActive)
	s := &channelStub{result: channels.Result{Outcome: channels.Renewed, Channel: active}}
	r := channelRouter(t, s, channelPrincipal())
	for _, body := range []string{"", "{}", " {} "} {
		s.calls = 0
		w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/renew", body)
		data, _ := decodeEnvelope(t, w)["data"].(map[string]any)
		if w.Code != 200 || data["result"] != "renewed" || s.calls != 1 || s.site != "test-site" || s.zone != "renew" || s.channel.String() != channelID {
			t.Fatal(body, w.Code, w.Body.String(), s)
		}
	}
	for _, body := range []string{`{"expires_at":"2030-01-01T00:00:00Z"}`, `{"last_activity_at":"x"}`, `[]`, `null`, `{"tiu":"1"}`} {
		s.calls = 0
		if w := channelRequest(r, "POST", channelsBase+"/"+channelID+"/renew", body); w.Code != 400 || decodeEnvelope(t, w)["code"] != "CHANNEL_REQUEST_INVALID" || s.calls != 0 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	if w := channelRequest(r, "POST", channelsBase+"/not-a-uuid/renew", ""); w.Code != 422 || s.calls != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, tc := range []struct {
		outcome string
		channel *channels.Channel
		want    int
		code    string
	}{
		{channels.Pending, stubChannel(channels.StatusPending), 409, "CHANNEL_NOT_ACTIVE"},
		{channels.Closed, stubChannel(channels.StatusClosed), 409, "CHANNEL_CLOSED"},
		{channels.Expired, stubChannel(channels.StatusExpired), 409, "CHANNEL_EXPIRED"},
		{channels.NotFound, nil, 404, "CHANNEL_NOT_FOUND"},
	} {
		s := &channelStub{result: channels.Result{Outcome: tc.outcome, Channel: tc.channel}, err: channels.OutcomeError(tc.outcome)}
		w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", channelsBase+"/"+channelID+"/renew", "")
		out := decodeEnvelope(t, w)
		data, _ := out["data"].(map[string]any)
		if w.Code != tc.want || out["code"] != tc.code || (tc.channel == nil) != (data == nil) || (data != nil && data["result"] != tc.outcome) {
			t.Fatal(tc.outcome, w.Code, w.Body.String())
		}
	}
	s = &channelStub{err: &pgconn.PgError{Code: "XX000", Message: "secret detail"}}
	if w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", channelsBase+"/"+channelID+"/renew", ""); w.Code != 500 || decodeEnvelope(t, w)["code"] != "INTERNAL_ERROR" || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w.Code, w.Body.String())
	}
}

// Bind identity (C11-H): the strict body, the account UUID passed through unchanged,
// and every refusal with its code and the channel's state.
func TestChannelBindIdentity(t *testing.T) {
	const user = "7c81b7f2-9f54-4c12-8a5b-6d203e7f4190"
	path := channelsBase + "/" + channelID + "/bind-identity"
	valid := `{"identity":{"type":"authenticated","user_uuid":"` + user + `"}}`
	for _, outcome := range []string{channels.Bound, channels.AlreadyBound} {
		s := &channelStub{result: channels.Result{Outcome: outcome, Channel: stubChannel(channels.StatusActive)}}
		w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", path, valid)
		data, _ := decodeEnvelope(t, w)["data"].(map[string]any)
		if w.Code != 200 || data["result"] != outcome || s.calls != 1 || s.site != "test-site" || s.zone != "bind" || s.channel.String() != channelID || s.user.String() != user {
			t.Fatal(outcome, w.Code, w.Body.String(), s)
		}
	}
	s := &channelStub{result: channels.Result{Outcome: channels.Bound, Channel: stubChannel(channels.StatusActive)}}
	r := channelRouter(t, s, channelPrincipal())
	for body, want := range map[string]int{
		"": 400, "{}": 400, "null": 400, "[]": 400, "x": 400, `{"user_uuid":"` + user + `"}`: 400,
		`{"identity":{"user_uuid":"` + user + `"}}`:                                         400,
		`{"identity":{"type":"authenticated"}}`:                                             400,
		`{"identity":{"type":"authenticated","user_uuid":"` + user + `","email":"a@b.c"}}`:  400,
		`{"identity":{"type":"authenticated","user_uuid":"` + user + `"},"site":"other"}`:   400,
		`{"identity":{"type":"authenticated","user_uuid":"` + user + `"},"codexhash":"aa"}`: 400,
		valid + ` {}`: 400,
		`{"identity":{"type":"authenticated","user_uuid":"` + user + `"},"identity":{"type":"authenticated","user_uuid":"` + user + `"}}`: 400,
		`{"identity":{"type":"guest","user_uuid":"` + user + `"}}`:                                                                        422,
		`{"identity":{"type":"authenticated","user_uuid":"` + strings.ToUpper(user) + `"}}`:                                               422,
		`{"identity":{"type":"authenticated","user_uuid":"{` + user + `}"}}`:                                                              422,
		`{"identity":{"type":"authenticated","user_uuid":"urn:uuid:` + user + `"}}`:                                                       422,
		`{"identity":{"type":"authenticated","user_uuid":"` + strings.ReplaceAll(user, "-", "") + `"}}`:                                   422,
		`{"identity":{"type":"authenticated","user_uuid":"00000000-0000-0000-0000-000000000000"}}`:                                        422,
		`{"identity":{"type":"authenticated","user_uuid":"someone@example.com"}}`:                                                         422,
		`{"identity":{"type":"authenticated","user_uuid":"42"}}`:                                                                          422,
		`{"identity":{"type":"authenticated","user_uuid":42}}`:                                                                            400,
	} {
		s.calls = 0
		if w := channelRequest(r, "POST", path, body); w.Code != want || decodeEnvelope(t, w)["code"] != "CHANNEL_REQUEST_INVALID" || s.calls != 0 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	if w := channelRequest(r, "GET", path, ""); w.Code != 404 || s.calls != 0 {
		t.Fatal("GET bind-identity", w.Code)
	}
	if w := channelRequest(r, "POST", channelsBase+"/not-a-uuid/bind-identity", valid); w.Code != 422 || s.calls != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, tc := range []struct {
		outcome string
		channel *channels.Channel
		want    int
		code    string
	}{
		{channels.Pending, stubChannel(channels.StatusPending), 409, "CHANNEL_NOT_ACTIVE"},
		{channels.Closed, stubChannel(channels.StatusClosed), 409, "CHANNEL_CLOSED"},
		{channels.Expired, stubChannel(channels.StatusExpired), 409, "CHANNEL_EXPIRED"},
		{channels.IdentityConflict, stubChannel(channels.StatusActive), 409, "CHANNEL_IDENTITY_CONFLICT"},
		{channels.BindingUnsupported, stubChannel(channels.StatusActive), 409, "CHANNEL_BINDING_UNSUPPORTED"},
		{channels.NotFound, nil, 404, "CHANNEL_NOT_FOUND"},
		{channels.ProtectionMismatch, stubChannel(channels.StatusActive), 500, "INTERNAL_ERROR"},
	} {
		s := &channelStub{result: channels.Result{Outcome: tc.outcome, Channel: tc.channel}}
		w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", path, valid)
		out := decodeEnvelope(t, w)
		data, _ := out["data"].(map[string]any)
		if w.Code != tc.want || out["code"] != tc.code || (tc.want == 409) != (data != nil) || (data != nil && data["result"] != tc.outcome) {
			t.Fatal(tc.outcome, w.Code, w.Body.String())
		}
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{channels.ErrUserIdentifierInvalid, 422, "CHANNEL_REQUEST_INVALID"},
		{channels.ErrCodexHashUnavailable, 503, "CODEXHASH_UNAVAILABLE"},
		{channels.ErrCodexHashInvalid, 502, "CODEXHASH_INVALID_RESPONSE"},
		{&pgconn.PgError{Code: "XX000", Message: "secret detail"}, 500, "INTERNAL_ERROR"},
	} {
		s := &channelStub{err: tc.err}
		w := channelRequest(channelRouter(t, s, channelPrincipal()), "POST", path, valid)
		if w.Code != tc.status || decodeEnvelope(t, w)["code"] != tc.code || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(tc.err, w.Code, w.Body.String())
		}
	}
}

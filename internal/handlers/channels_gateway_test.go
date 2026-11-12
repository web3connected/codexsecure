package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3connected/codexsecure-api/internal/channels"
)

// Phase C6: channel routes authenticated by the gateway assertion alone, as
// registered by RegisterRoutes (GatewayAssertion → ChannelAuth → handler).

const gatewaySite = "web3codex-local"
const gatewayBase = "/api/v1/sites/" + gatewaySite + "/channels"

type gatewaySigner struct {
	kid string
	key ed25519.PrivateKey
}

func newGatewaySigner(t *testing.T, kid string) (gatewaySigner, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return gatewaySigner{kid, priv}, kid + ":" + base64.RawURLEncoding.EncodeToString(pub)
}

// sign produces the C4 format (codex_gateway src/auth/gatewayAssertion.ts).
func (g gatewaySigner) sign(t *testing.T, app, site, method, path string, iat time.Time) string {
	t.Helper()
	enc := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	input := enc(map[string]any{"alg": "EdDSA", "kid": g.kid, "typ": "codexsecure-gateway+jwt"}) + "." + enc(map[string]any{
		"app": app, "aud": "codexsecure-api", "channel_sites": []string{site}, "exp": iat.Unix() + 30, "iat": iat.Unix(),
		"iss": "codex-gateway", "method": method, "path": path, "rid": uuid.NewString(), "sub": "application:" + app,
	})
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(g.key, []byte(input)))
}

func gatewayRouter(t *testing.T, s *channelStub, keys string) *gin.Engine {
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	t.Setenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS", keys)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewChannelHandler(s).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func gatewayRequest(r *gin.Engine, method, path, body, serviceKey string, assertions ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if serviceKey != "" {
		req.Header.Set("X-Service-Key", serviceKey)
	}
	for _, a := range assertions {
		req.Header.Add("X-CodexSecure-Gateway-Assertion", a)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var gatewayRoutes = []struct {
	name, method, path, body string
	want                     int
	result                   string
}{
	{"open", "POST", gatewayBase, `{"zone":"Z1"}`, 201, channels.Opened},
	{"get", "GET", gatewayBase + "/" + channelID, "", 200, channels.Found},
	{"activate", "POST", gatewayBase + "/" + channelID + "/activate", "", 200, channels.Activated},
	{"close", "POST", gatewayBase + "/" + channelID + "/close", "", 200, channels.Closed},
	{"renew", "POST", gatewayBase + "/" + channelID + "/renew", "", 200, channels.Renewed},
}

func TestChannelGatewayAssertionAccepted(t *testing.T) {
	signer, keys := newGatewaySigner(t, "gateway-2026-10")
	for _, route := range gatewayRoutes {
		t.Run(route.name, func(t *testing.T) {
			ch := stubChannel("pending")
			ch.Site = gatewaySite
			s := &channelStub{result: channels.Result{Outcome: route.result, Channel: ch}}
			r := gatewayRouter(t, s, keys)
			w := gatewayRequest(r, route.method, route.path, route.body, "test-service-key",
				signer.sign(t, "web3connected-demo", gatewaySite, route.method, route.path, time.Now()))
			if w.Code != route.want || s.calls != 1 || s.site != gatewaySite {
				t.Fatal(w.Code, w.Body.String(), s.calls, s.site)
			}
			if route.name != "open" && s.channel.String() != channelID {
				t.Fatal("channel", s.channel)
			}
		})
	}
}

func TestChannelGatewayAssertionRejected(t *testing.T) {
	signer, keys := newGatewaySigner(t, "gateway-2026-10")
	untrusted, _ := newGatewaySigner(t, "gateway-2026-10")
	now := time.Now()
	for _, route := range gatewayRoutes {
		valid := func() string { return signer.sign(t, "web3connected-demo", gatewaySite, route.method, route.path, now) }
		otherMethod := map[string]string{"GET": "POST", "POST": "GET"}[route.method]
		otherPath := strings.Replace(route.path, gatewaySite, "other-site", 1)
		for _, tc := range []struct {
			name       string
			path       string
			serviceKey string
			keys       string
			assertions []string
			want       int
			code       string
		}{
			{"missing assertion", route.path, "test-service-key", keys, nil, 403, "ADMIN_UNAUTHORIZED"},
			{"invalid signature", route.path, "test-service-key", keys, []string{valid()[:len(valid())-4] + "AAAA"}, 403, "ADMIN_UNAUTHORIZED"},
			{"expired", route.path, "test-service-key", keys, []string{signer.sign(t, "web3connected-demo", gatewaySite, route.method, route.path, now.Add(-40*time.Second))}, 403, "ADMIN_UNAUTHORIZED"},
			{"modified site", route.path, "test-service-key", keys, []string{signer.sign(t, "web3connected-demo", "other-site", route.method, route.path, now)}, 403, "ADMIN_UNAUTHORIZED"},
			{"modified application", route.path, "test-service-key", keys, []string{tamperClaim(t, valid(), "app", "other-app")}, 403, "ADMIN_UNAUTHORIZED"},
			{"modified method", route.path, "test-service-key", keys, []string{signer.sign(t, "web3connected-demo", gatewaySite, otherMethod, route.path, now)}, 403, "ADMIN_UNAUTHORIZED"},
			{"modified path", route.path, "test-service-key", keys, []string{signer.sign(t, "web3connected-demo", gatewaySite, route.method, gatewayBase+"/"+uuid.NewString(), now)}, 403, "ADMIN_UNAUTHORIZED"},
			{"untrusted gateway key", route.path, "test-service-key", keys, []string{untrusted.sign(t, "web3connected-demo", gatewaySite, route.method, route.path, now)}, 403, "ADMIN_UNAUTHORIZED"},
			{"no keys configured", route.path, "test-service-key", "", []string{valid()}, 403, "ADMIN_UNAUTHORIZED"},
			{"invalid key configuration", route.path, "test-service-key", "gateway-2026-10:not-a-key", []string{valid()}, 403, "ADMIN_UNAUTHORIZED"},
			{"missing service key", route.path, "", keys, []string{valid()}, 401, "SERVICE_KEY_INVALID"},
			{"wrong service key", route.path, "wrong", keys, []string{valid()}, 401, "SERVICE_KEY_INVALID"},
			// An assertion for web3codex-local does not open another site's channels.
			{"cross-site", otherPath, "test-service-key", keys, []string{valid()}, 403, "ADMIN_UNAUTHORIZED"},
			{"cross-site with rewritten path", otherPath, "test-service-key", keys, []string{signer.sign(t, "web3connected-demo", gatewaySite, route.method, otherPath, now)}, 403, "ADMIN_UNAUTHORIZED"},
		} {
			t.Run(route.name+" "+tc.name, func(t *testing.T) {
				s := &channelStub{result: channels.Result{Outcome: route.result, Channel: stubChannel("pending")}}
				r := gatewayRouter(t, s, tc.keys)
				w := gatewayRequest(r, route.method, tc.path, route.body, tc.serviceKey, tc.assertions...)
				if w.Code != tc.want || decodeEnvelope(t, w)["code"] != tc.code || s.calls != 0 {
					t.Fatal(w.Code, w.Body.String(), s.calls)
				}
			})
		}
	}
	t.Run("replayed assertion", func(t *testing.T) {
		s := &channelStub{result: channels.Result{Outcome: channels.Found, Channel: stubChannel("pending")}}
		r := gatewayRouter(t, s, keys)
		path := gatewayBase + "/" + channelID
		token := signer.sign(t, "web3connected-demo", gatewaySite, "GET", path, now)
		if w := gatewayRequest(r, "GET", path, "", "test-service-key", token); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if w := gatewayRequest(r, "GET", path, "", "test-service-key", token); w.Code != 403 || s.calls != 1 {
			t.Fatal(w.Code, w.Body.String(), s.calls)
		}
	})
}

// tamperClaim changes a claim without re-signing.
func tamperClaim(t *testing.T, token, name string, value any) string {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	claims := map[string]any{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	claims[name] = value
	raw, _ = json.Marshal(claims)
	return parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw) + "." + parts[2]
}

// The assertion is verified only on channel routes; it grants nothing on the
// communication routes, which keep requiring their own principal.
func TestGatewayAssertionIgnoredOutsideChannels(t *testing.T) {
	signer, keys := newGatewaySigner(t, "gateway-2026-10")
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	t.Setenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS", keys)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	NewCommunicationHandler(&communicationStub{}).RegisterRoutes(v1)
	NewChannelHandler(&channelStub{}).RegisterRoutes(v1)
	path := "/api/v1/sites/" + gatewaySite + "/communications/d9fe68fe-b0dc-4aca-b625-56c3514479f2/configuration"
	w := gatewayRequest(r, "GET", path, "", "test-service-key", signer.sign(t, "web3connected-demo", gatewaySite, "GET", path, time.Now()))
	if w.Code != 403 || decodeEnvelope(t, w)["code"] != "ADMIN_UNAUTHORIZED" {
		t.Fatal(w.Code, w.Body.String())
	}
}

package middleware

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// testdata/c4_gateway_assertions.json was produced by the real C4 signer
// (codex_gateway dist/auth/gatewayAssertion.js issueAssertion) with a throwaway
// key whose private half was discarded.
type c4Fixture struct {
	Kid        string `json:"kid"`
	PublicKeyX string `json:"public_key_x"`
	Iat        int64  `json:"iat"`
	Assertions map[string]struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Rid    string `json:"rid"`
		Token  string `json:"token"`
	} `json:"assertions"`
}

const fixtureSite = "web3codex-local"

func loadC4Fixture(t *testing.T) c4Fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/c4_gateway_assertions.json")
	if err != nil {
		t.Fatal(err)
	}
	var f c4Fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func fixtureVerifier(t *testing.T, f c4Fixture, at time.Time) *GatewayVerifier {
	t.Helper()
	keys, err := ParseGatewayAssertionKeys(f.Kid + ":" + f.PublicKeyX)
	if err != nil {
		t.Fatal(err)
	}
	return NewGatewayVerifier(keys, func() time.Time { return at })
}

func TestGatewayAssertionVerifiesRealC4Signer(t *testing.T) {
	f := loadC4Fixture(t)
	for _, op := range []string{"open", "get", "activate", "close"} {
		a := f.Assertions[op]
		t.Run(op, func(t *testing.T) {
			v := fixtureVerifier(t, f, time.Unix(f.Iat+1, 0))
			claims, err := v.Verify(a.Token, a.Method, a.Path, fixtureSite)
			if err != nil {
				t.Fatal(err)
			}
			if claims.App != "web3connected-demo" || claims.Sub != "application:web3connected-demo" || claims.Rid != a.Rid || *claims.Exp-*claims.Iat != 30 {
				t.Fatalf("%+v", claims)
			}
		})
	}
	t.Run("validity window", func(t *testing.T) {
		a := f.Assertions["get"]
		for offset, ok := range map[int64]bool{-6: false, -5: true, 0: true, 30: true, 35: true, 36: false} {
			_, err := fixtureVerifier(t, f, time.Unix(f.Iat+offset, 0)).Verify(a.Token, a.Method, a.Path, fixtureSite)
			if (err == nil) != ok {
				t.Fatal(offset, err)
			}
		}
	})
	t.Run("bound to its request", func(t *testing.T) {
		a := f.Assertions["activate"]
		for name, req := range map[string][3]string{
			"method":     {"GET", a.Path, fixtureSite},
			"path":       {a.Method, strings.TrimSuffix(a.Path, "/activate") + "/close", fixtureSite},
			"site":       {a.Method, a.Path, "other-site"},
			"encoding":   {a.Method, strings.Replace(a.Path, "web3codex-local", "web3codex%2Dlocal", 1), fixtureSite},
			"other path": {a.Method, f.Assertions["open"].Path, fixtureSite},
		} {
			v := fixtureVerifier(t, f, time.Unix(f.Iat, 0))
			if _, err := v.Verify(a.Token, req[0], req[1], req[2]); err == nil {
				t.Fatal(name, "accepted")
			}
		}
	})
	t.Run("rid is single use", func(t *testing.T) {
		a := f.Assertions["open"]
		v := fixtureVerifier(t, f, time.Unix(f.Iat, 0))
		if _, err := v.Verify(a.Token, a.Method, a.Path, fixtureSite); err != nil {
			t.Fatal(err)
		}
		if _, err := v.Verify(a.Token, a.Method, a.Path, fixtureSite); err == nil || err.Error() != "replayed rid" {
			t.Fatal("replay accepted", err)
		}
	})
	t.Run("tampered segments", func(t *testing.T) {
		a := f.Assertions["open"]
		parts := strings.Split(a.Token, ".")
		claims := decodeTestSegment(t, parts[1])
		header := decodeTestSegment(t, parts[0])
		for name, token := range map[string]string{
			"site":      parts[0] + "." + reencode(t, claims, "channel_sites", []string{"other-site"}) + "." + parts[2],
			"app":       parts[0] + "." + reencode(t, claims, "app", "other-app") + "." + parts[2],
			"exp":       parts[0] + "." + reencode(t, claims, "exp", f.Iat+3600) + "." + parts[2],
			"kid":       reencode(t, header, "kid", "gw-other") + "." + parts[1] + "." + parts[2],
			"signature": parts[0] + "." + parts[1] + "." + flipFirst(parts[2]),
		} {
			v := fixtureVerifier(t, f, time.Unix(f.Iat, 0))
			if _, err := v.Verify(token, a.Method, a.Path, fixtureSite); err == nil {
				t.Fatal(name, "accepted")
			}
		}
	})
	t.Run("untrusted gateway key under the same kid", func(t *testing.T) {
		a := f.Assertions["open"]
		other, _, _ := ed25519.GenerateKey(rand.Reader)
		v := NewGatewayVerifier(map[string]ed25519.PublicKey{f.Kid: other}, func() time.Time { return time.Unix(f.Iat, 0) })
		if _, err := v.Verify(a.Token, a.Method, a.Path, fixtureSite); err == nil {
			t.Fatal("accepted")
		}
	})
}

// testAssertion mirrors issueAssertion for cases the gateway never signs.
type testAssertion struct {
	header map[string]any
	claims map[string]any
}

func newTestAssertion(kid, method, path, site, rid string, iat int64) testAssertion {
	return testAssertion{
		header: map[string]any{"alg": "EdDSA", "kid": kid, "typ": "codexsecure-gateway+jwt"},
		claims: map[string]any{"app": "web3connected-demo", "aud": "codexsecure-api", "channel_sites": []string{site}, "exp": iat + 30, "iat": iat,
			"iss": "codex-gateway", "method": method, "path": path, "rid": rid, "sub": "application:web3connected-demo"},
	}
}

func (a testAssertion) sign(t *testing.T, key ed25519.PrivateKey) string {
	t.Helper()
	input := encodeTestSegment(t, a.header) + "." + encodeTestSegment(t, a.claims)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(input)))
}

func encodeTestSegment(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v) // map keys are sorted, as the gateway writes them
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeTestSegment(t *testing.T, segment string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func reencode(t *testing.T, m map[string]any, key string, value any) string {
	copied := map[string]any{}
	for k, v := range m {
		copied[k] = v
	}
	copied[key] = value
	return encodeTestSegment(t, copied)
}

func flipFirst(s string) string {
	if s[0] == 'A' {
		return "B" + s[1:]
	}
	return "A" + s[1:]
}

func TestGatewayAssertionRejections(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	const iat = int64(1793361600)
	const path = "/api/v1/sites/web3codex-local/channels"
	verifier := func() *GatewayVerifier {
		return NewGatewayVerifier(map[string]ed25519.PublicKey{"gw-1": pub}, func() time.Time { return time.Unix(iat, 0) })
	}
	base := func() testAssertion { return newTestAssertion("gw-1", "POST", path, fixtureSite, "rid-1", iat) }
	if _, err := verifier().Verify(base().sign(t, priv), "POST", path, fixtureSite); err != nil {
		t.Fatal("baseline", err)
	}
	for name, mutate := range map[string]func(*testAssertion){
		"alg none":            func(a *testAssertion) { a.header["alg"] = "none" },
		"alg Ed25519":         func(a *testAssertion) { a.header["alg"] = "Ed25519" },
		"typ JWT":             func(a *testAssertion) { a.header["typ"] = "JWT" },
		"unknown kid":         func(a *testAssertion) { a.header["kid"] = "gw-2" },
		"extra header member": func(a *testAssertion) { a.header["crit"] = []string{"exp"} },
		"iss":                 func(a *testAssertion) { a.claims["iss"] = "someone-else" },
		"aud":                 func(a *testAssertion) { a.claims["aud"] = "codexhash-api" },
		"extra claim":         func(a *testAssertion) { a.claims["admin_sites"] = []string{fixtureSite} },
		"missing exp":         func(a *testAssertion) { delete(a.claims, "exp") },
		"missing iat":         func(a *testAssertion) { delete(a.claims, "iat") },
		"fractional iat":      func(a *testAssertion) { a.claims["iat"] = float64(iat) + 0.5 },
		"lifetime over 60s":   func(a *testAssertion) { a.claims["iat"] = iat - 40; a.claims["exp"] = iat + 30 },
		"exp before iat":      func(a *testAssertion) { a.claims["exp"] = iat - 1 },
		"expired":             func(a *testAssertion) { a.claims["iat"] = iat - 40; a.claims["exp"] = iat - 10 },
		"future iat":          func(a *testAssertion) { a.claims["iat"] = iat + 10; a.claims["exp"] = iat + 40 },
		"sub not app":         func(a *testAssertion) { a.claims["sub"] = "application:other-app" },
		"sub administrator":   func(a *testAssertion) { a.claims["sub"] = "administrator:web3connected-demo" },
		"invalid app": func(a *testAssertion) {
			a.claims["app"] = "Web3Connected"
			a.claims["sub"] = "application:Web3Connected"
		},
		"method":           func(a *testAssertion) { a.claims["method"] = "GET" },
		"lowercase method": func(a *testAssertion) { a.claims["method"] = "post" },
		"path":             func(a *testAssertion) { a.claims["path"] = "/api/v1/sites/other-site/channels" },
		"path with query":  func(a *testAssertion) { a.claims["path"] = path + "?x=1" },
		"other site":       func(a *testAssertion) { a.claims["channel_sites"] = []string{"other-site"} },
		"two sites":        func(a *testAssertion) { a.claims["channel_sites"] = []string{fixtureSite, "other-site"} },
		"no sites":         func(a *testAssertion) { a.claims["channel_sites"] = []string{} },
		"empty rid":        func(a *testAssertion) { a.claims["rid"] = "" },
	} {
		a := base()
		mutate(&a)
		if _, err := verifier().Verify(a.sign(t, priv), "POST", path, fixtureSite); err == nil {
			t.Fatal(name, "accepted")
		}
	}
	token := base().sign(t, priv)
	parts := strings.Split(token, ".")
	for name, bad := range map[string]string{
		"empty":           "",
		"two segments":    parts[0] + "." + parts[1],
		"four segments":   token + ".x",
		"padded":          parts[0] + "." + parts[1] + "." + base64.URLEncoding.EncodeToString(mustDecode(t, parts[2])),
		"std base64":      strings.NewReplacer("-", "+", "_", "/").Replace(token) + "=",
		"short signature": parts[0] + "." + parts[1] + "." + parts[2][:20],
		"oversized":       token + strings.Repeat("A", 5000),
	} {
		if _, err := verifier().Verify(bad, "POST", path, fixtureSite); err == nil {
			t.Fatal(name, "accepted")
		}
	}
	if _, err := NewGatewayVerifier(nil, nil).Verify(token, "POST", path, fixtureSite); err == nil {
		t.Fatal("verifier without keys accepted")
	}
}

func mustDecode(t *testing.T, s string) []byte {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestParseGatewayAssertionKeys(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	x := base64.RawURLEncoding.EncodeToString(pub)
	for _, bad := range []string{"", " ", "gw-1", "gw-1:", ":" + x, "bad kid:" + x, "gw-1:" + x + ",gw-1:" + x,
		"gw-1:" + base64.StdEncoding.EncodeToString(pub), "gw-1:" + x[:20], "gw-1:" + base64.RawURLEncoding.EncodeToString(append(pub, 0))} {
		if _, err := ParseGatewayAssertionKeys(bad); err == nil {
			t.Fatal(bad, "accepted")
		}
	}
	keys, err := ParseGatewayAssertionKeys(" gw-1:" + x + ", gw-2026-10:" + x)
	if err != nil || len(keys) != 2 || !keys["gw-2026-10"].Equal(pub) {
		t.Fatal(keys, err)
	}
}

// The middleware installs a principal only for a verified assertion, and only
// with the asserted site as a channel grant.
func TestGatewayAssertionMiddleware(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	gin.SetMode(gin.TestMode)
	run := func(v *GatewayVerifier, method, path string, assertions ...string) (CommunicationPrincipal, bool) {
		var got CommunicationPrincipal
		var installed bool
		r := gin.New()
		r.Any("/api/v1/sites/:site/channels", v.Middleware(), func(c *gin.Context) {
			value, ok := c.Get(communicationPrincipalKey)
			got, installed = value.(CommunicationPrincipal)
			installed = installed && ok
		})
		req := httptest.NewRequest(method, path, nil)
		for _, a := range assertions {
			req.Header.Add(GatewayAssertionHeader, a)
		}
		r.ServeHTTP(httptest.NewRecorder(), req)
		return got, installed
	}
	now := time.Now()
	v := NewGatewayVerifier(map[string]ed25519.PublicKey{"gw-1": pub}, nil)
	path := "/api/v1/sites/" + fixtureSite + "/channels"
	sign := func(rid string) string {
		return newTestAssertion("gw-1", "POST", path, fixtureSite, rid, now.Unix()).sign(t, priv)
	}

	p, ok := run(v, http.MethodPost, path, sign("rid-a"))
	if !ok || !p.GatewayAuthenticated || p.Actor != "application:web3connected-demo" || len(p.ChannelSites) != 1 || !p.ChannelSites[fixtureSite] ||
		len(p.ReadSites) != 0 || len(p.AdminSites) != 0 || len(p.DiscoverySites) != 0 {
		t.Fatalf("%v %+v", ok, p)
	}
	if _, ok := run(v, http.MethodPost, path); ok {
		t.Fatal("principal without assertion")
	}
	if _, ok := run(v, http.MethodPost, path, sign("rid-b"), sign("rid-c")); ok {
		t.Fatal("principal with two assertion headers")
	}
	if _, ok := run(v, http.MethodPost, path+"?site=other-site", sign("rid-d")); !ok {
		t.Fatal("the query is not part of the bound path")
	}
	if _, ok := run(v, http.MethodPost, "/api/v1/sites/other-site/channels", sign("rid-e")); ok {
		t.Fatal("principal for another site")
	}
	if _, ok := run(NewGatewayVerifier(nil, nil), http.MethodPost, path, sign("rid-f")); ok {
		t.Fatal("principal without configured keys")
	}
}

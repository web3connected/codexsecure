package middleware

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommunicationGatewayPrincipal(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	t.Setenv("SERVICE_API_KEY", "test-service")
	gin.SetMode(gin.TestMode)
	const site = "web3codex-local"
	const actor = "web3codex-admin:1"
	const idem = "3f0e1c2a-8b4d-4e6f-9a1b-2c3d4e5f6a7b"
	for _, tc := range []struct {
		name, method, grant, service, token, requestSite, body string
		status                                                 int
	}{
		{"site read", "GET", "read_sites", "test-service", "valid", site, "", 204},
		{"verified administrator", "POST", "admin_sites", "test-service", "valid", site, "{}", 204},
		{"body changed", "POST", "admin_sites", "test-service", "valid", site, "{\"reason\":\"tampered\"}", 403},
		{"read cannot administer", "POST", "read_sites", "test-service", "valid", site, "{}", 403},
		{"channel cannot read", "GET", "channel_sites", "test-service", "valid", site, "", 403},
		{"missing service", "GET", "read_sites", "", "valid", site, "", 401},
		{"wrong service", "GET", "read_sites", "wrong", "valid", site, "", 401},
		{"missing assertion", "GET", "read_sites", "test-service", "", site, "", 403},
		{"forged assertion", "GET", "read_sites", "test-service", "forged", site, "", 403},
		{"different site", "GET", "read_sites", "test-service", "valid", "other-site", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/sites/" + site + "/communications/assessments"
			assertion := newTestAssertion("gw-test", tc.method, path, site, strings.ReplaceAll(tc.name, " ", "-"), time.Now().Unix())
			delete(assertion.claims, "channel_sites")
			assertion.claims[tc.grant] = []string{site}
			if tc.grant == "admin_sites" {
				assertion.claims["sub"] = actor
				digest := sha256.Sum256([]byte("{}"))
				assertion.claims["body_sha256"] = hex.EncodeToString(digest[:])
			}
			token := tc.token
			if token == "valid" {
				token = assertion.sign(t, private)
			}
			router := gin.New()
			verifier := NewGatewayVerifier(map[string]ed25519.PublicKey{"gw-test": public}, nil)
			router.Any("/api/v1/sites/:site/communications/assessments", verifier.Middleware(), CommunicationAuth(), func(c *gin.Context) {
				if tc.grant == "admin_sites" && CommunicationActor(c) != actor {
					t.Fatal("verified actor lost")
				}
				c.Status(204)
			})
			request := httptest.NewRequest(tc.method, strings.Replace(path, site, tc.requestSite, 1), strings.NewReader(tc.body))
			request.Header.Set("X-Service-Key", tc.service)
			request.Header.Set(GatewayAssertionHeader, token)
			request.Header.Set("X-CodexSecure-Actor", actor)
			request.Header.Set("Idempotency-Key", idem)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

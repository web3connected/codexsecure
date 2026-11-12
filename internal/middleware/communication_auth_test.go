package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommunicationAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	p := CommunicationPrincipal{GatewayAuthenticated: true, Actor: "admin-1", AdminSites: map[string]bool{"site-a": true}}
	cases := []struct {
		name, key, provided, actor, replay, site string
		principal                                *CommunicationPrincipal
		want                                     int
	}{
		{"absent configured key", "", "", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 401},
		{"missing service key", "secret", "", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 401},
		{"invalid service key", "secret", "bad", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 401},
		{"header alone is untrusted", "secret", "secret", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", nil, 403},
		{"missing actor", "secret", "secret", "", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 403},
		{"spoofed actor", "secret", "secret", "other", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 403},
		{"missing replay key", "secret", "secret", "admin-1", "", "site-a", &p, 422},
		{"cross site", "secret", "secret", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-b", &p, 403},
		{"authorized context", "secret", "secret", "admin-1", "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "site-a", &p, 204},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SERVICE_API_KEY", tc.key)
			r := gin.New()
			if tc.principal != nil {
				r.Use(func(c *gin.Context) { SetCommunicationPrincipal(c, *tc.principal) })
			}
			r.POST("/sites/:site/action", CommunicationAuth(), func(c *gin.Context) { c.Status(204) })
			req := httptest.NewRequest(http.MethodPost, "/sites/"+tc.site+"/action", nil)
			req.Header.Set("X-Service-Key", tc.provided)
			req.Header.Set("X-CodexSecure-Actor", tc.actor)
			req.Header.Set("Idempotency-Key", tc.replay)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCommunicationAuthorityAcrossAllDecisions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, route := range []struct{ method, path string }{{"PUT", "/configuration/draft"}, {"POST", "/configuration/approve"}, {"POST", "/configuration/reject"}, {"POST", "/disable"}, {"POST", "/revoke"}} {
		for _, tc := range []struct {
			name           string
			gateway, admin bool
			actor          string
			want           int
		}{{"unverified gateway", false, true, "admin-1", 403}, {"read and discovery grants are not admin", true, false, "admin-1", 403}, {"forged actor", true, true, "other-admin", 403}, {"trusted administrator", true, true, "admin-1", 204}} {
			t.Run(route.path+"/"+tc.name, func(t *testing.T) {
				t.Setenv("SERVICE_API_KEY", "test-service")
				r := gin.New()
				r.Use(func(c *gin.Context) {
					SetCommunicationPrincipal(c, CommunicationPrincipal{GatewayAuthenticated: tc.gateway, Actor: "admin-1", AdminSites: map[string]bool{"site-a": tc.admin}, ReadSites: map[string]bool{"site-a": true}, DiscoverySites: map[string]bool{"site-a": true}})
				})
				r.Handle(route.method, "/sites/:site"+route.path, CommunicationAuth(), func(c *gin.Context) {
					if CommunicationActor(c) != "admin-1" {
						t.Error("audit actor was not verified")
					}
					c.Status(204)
				})
				req := httptest.NewRequest(route.method, "/sites/site-a"+route.path, nil)
				req.Header.Set("X-Service-Key", "test-service")
				req.Header.Set("X-CodexSecure-Actor", tc.actor)
				req.Header.Set("Idempotency-Key", "d9fe68fe-b0dc-4aca-b625-56c3514479f2")
				req.Header.Set("X-Gateway", "forged")
				req.Header.Set("X-API-Key", "unverified-client-key")
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != tc.want {
					t.Fatal(w.Code, w.Body.String())
				}
			})
		}
	}
}
func TestCommunicationPrincipalGrantsAreRequestLocal(t *testing.T) {
	t.Setenv("SERVICE_API_KEY", "test-service")
	r := gin.New()
	r.Use(func(c *gin.Context) {
		p := CommunicationPrincipal{GatewayAuthenticated: true, Actor: "admin-1", AdminSites: map[string]bool{"site-a": true}}
		SetCommunicationPrincipal(c, p)
		p.AdminSites["site-b"] = true
	})
	r.POST("/sites/:site/approve", CommunicationAuth(), func(c *gin.Context) { c.Status(204) })
	req := httptest.NewRequest("POST", "/sites/site-b/approve", nil)
	req.Header.Set("X-Service-Key", "test-service")
	req.Header.Set("X-CodexSecure-Actor", "admin-1")
	req.Header.Set("Idempotency-Key", "d9fe68fe-b0dc-4aca-b625-56c3514479f2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
}

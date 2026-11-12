package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/analyzer"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/internal/schema"
)

func TestPhaseLockMiddleware_DetectsRolesByValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := schema.NewRegistry(&schema.Artifact{
		ArtifactVersion: "1.0",
		SchemaSetID:     "test",
		Application:     schema.ApplicationMeta{Name: "test-app", Environment: "test"},
		Generation:      schema.GenerationMeta{GeneratedAt: "now", TrustMode: "test"},
		Records: []schema.Record{
			{
				SchemaID:      "s1",
				SchemaVersion: "1",
				Route: schema.RouteMeta{
					Name:     "auth.login",
					Family:   "auth",
					URI:      "/api/v1/zones/z2/login",
					Methods:  []string{"POST"},
					ZoneHint: "Z2",
				},
				Intent:  schema.IntentMeta{Type: "auth.login", Operation: "submit"},
				Status:  schema.RecordStatus{Trusted: true, Active: true, ReviewState: "trusted"},
			},
		},
	})

	pol := policy.NewRegistry()
	routeAnalyzer := analyzer.NewAnalyzer(pol)
	fp := fingerprint.NewAnalyzer(nil)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(RequestIDMiddleware())

	chain := NewChain(routeAnalyzer, pol, fp, nil)
	z2 := v1.Group("/zones/z2")
	z2.Use(PhaseLockMiddleware(fp, reg, PhaseLockConfig{Enabled: true, MaxBodyBytes: 64 * 1024, EmitResponseHeaders: true}))
	for _, mw := range chain.BuildWithExecutionSync("2.0.0") {
		z2.Use(mw)
	}
	z2.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	payload := map[string]any{
		"fieldA": "Test@Example.com",
		"fieldB": "P@ssw0rd123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/zones/z2/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get(HeaderPhaseLockStrongLock); got != "true" {
		t.Fatalf("expected %s=true, got %q", HeaderPhaseLockStrongLock, got)
	}
	if got := w.Header().Get(HeaderPhaseLockScore); got == "" {
		t.Fatalf("expected %s to be set", HeaderPhaseLockScore)
	}
}

func TestPhaseLockMiddleware_WeakWhenSecretMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := schema.NewRegistry(&schema.Artifact{
		ArtifactVersion: "1.0",
		SchemaSetID:     "test",
		Application:     schema.ApplicationMeta{Name: "test-app", Environment: "test"},
		Generation:      schema.GenerationMeta{GeneratedAt: "now", TrustMode: "test"},
		Records: []schema.Record{
			{
				SchemaID:      "s1",
				SchemaVersion: "1",
				Route: schema.RouteMeta{
					Name:     "auth.login",
					Family:   "auth",
					URI:      "/api/v1/zones/z2/login",
					Methods:  []string{"POST"},
					ZoneHint: "Z2",
				},
				Intent: schema.IntentMeta{Type: "auth.login", Operation: "submit"},
				Status: schema.RecordStatus{Trusted: true, Active: true, ReviewState: "trusted"},
			},
		},
	})

	pol := policy.NewRegistry()
	routeAnalyzer := analyzer.NewAnalyzer(pol)
	fp := fingerprint.NewAnalyzer(nil)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(RequestIDMiddleware())

	chain := NewChain(routeAnalyzer, pol, fp, nil)
	z2 := v1.Group("/zones/z2")
	z2.Use(PhaseLockMiddleware(fp, reg, PhaseLockConfig{Enabled: true, MaxBodyBytes: 64 * 1024, EmitResponseHeaders: true}))
	for _, mw := range chain.BuildWithExecutionSync("2.0.0") {
		z2.Use(mw)
	}
	z2.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	payload := map[string]any{
		"anything": "someone@example.com",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/zones/z2/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get(HeaderPhaseLockStrongLock); got != "false" {
		t.Fatalf("expected %s=false, got %q", HeaderPhaseLockStrongLock, got)
	}
}

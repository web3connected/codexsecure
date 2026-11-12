package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExecutionSyncMiddlewareAcceptedCycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var report ExecutionSyncReport
	var ok bool

	router.Use(func(c *gin.Context) {
		c.Next()
		report, ok = GetExecutionSyncReport(c)
	})
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneKey, "identity")
		c.Set(ContextKeyZoneName, "Identity")
		c.Set(ContextKeyZoneTier, "business")
		c.Set(ContextKeyZoneSource, "route_module")
		c.Set(ContextKeyZoneConfidence, 1.0)
		c.Next()
	})
	router.Use(ExecutionSyncMiddleware("dev-build"))
	router.GET("/zones/z2/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequest(http.MethodGet, "/zones/z2/login", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Header().Get(HeaderResult) != "accepted" {
		t.Fatalf("expected accepted result header, got %q", recorder.Header().Get(HeaderResult))
	}
	if recorder.Header().Get(HeaderZoneTier) != "business" {
		t.Fatalf("expected business zone tier, got %q", recorder.Header().Get(HeaderZoneTier))
	}
	if recorder.Header().Get(HeaderStage) != "execution_sync_middleware" {
		t.Fatalf("expected stage header, got %q", recorder.Header().Get(HeaderStage))
	}
	if recorder.Header().Get("X-Codex-Zone-Source") != "route_module" {
		t.Fatalf("expected route_module source header, got %q", recorder.Header().Get("X-Codex-Zone-Source"))
	}
	if !ok {
		t.Fatal("expected execution report in context")
	}
	if !report.Accepted {
		t.Fatal("expected accepted execution report")
	}
	if report.FinalStatus != "accepted" {
		t.Fatalf("expected accepted final status, got %q", report.FinalStatus)
	}
	if report.ZoneKey != "identity" {
		t.Fatalf("expected identity zone key, got %q", report.ZoneKey)
	}
	if report.ZoneSource != "route_module" {
		t.Fatalf("expected route_module zone source, got %q", report.ZoneSource)
	}
	if report.ZoneConfidence != 1.0 {
		t.Fatalf("expected 1.0 zone confidence, got %v", report.ZoneConfidence)
	}
}

func TestExecutionSyncMiddlewareRejectedCycleAppliesNoCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var report ExecutionSyncReport
	var ok bool

	router.Use(func(c *gin.Context) {
		c.Next()
		report, ok = GetExecutionSyncReport(c)
	})
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneKey, "administration")
		c.Set(ContextKeyZoneName, "Administration")
		c.Set(ContextKeyZoneTier, "critical")
		c.Next()
	})
	router.Use(ExecutionSyncMiddleware("dev-build"))
	router.POST("/zones/z5/execute", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	})

	request := httptest.NewRequest(http.MethodPost, "/zones/z5/execute", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
	if recorder.Header().Get(HeaderResult) != "rejected" {
		t.Fatalf("expected rejected result header, got %q", recorder.Header().Get(HeaderResult))
	}
	if recorder.Header().Get("Cache-Control") == "" {
		t.Fatal("expected no-cache headers for rejected critical zone")
	}
	if !ok {
		t.Fatal("expected execution report in context")
	}
	if report.Accepted {
		t.Fatal("expected rejected execution report")
	}
	if report.FinalStatus != "rejected" {
		t.Fatalf("expected rejected final status, got %q", report.FinalStatus)
	}
	if report.StatusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden status code, got %d", report.StatusCode)
	}
}
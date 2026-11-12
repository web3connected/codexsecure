package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/fingerprint"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

func makeSignedTIUToken(t *testing.T, secret string, iat int64, exp int64, deviceUID string) string {
	t.Helper()
	payload := "v=1&iat=" + strconv.FormatInt(iat, 10) + "&exp=" + strconv.FormatInt(exp, 10)
	if deviceUID != "" {
		payload += "&device_uid=" + url.QueryEscape(deviceUID)
	}
	payloadBytes := []byte(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestTIUMiddlewareAllowsIdentityBootstrapWithoutSessionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := policy.NewRegistry()
	zone, _ := registry.GetZone(2)

	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneID, zone.ID)
		c.Set(ContextKeyZoneKey, zone.Key)
		c.Set(ContextKeyZoneName, zone.Name)
		c.Set(ContextKeyZoneTier, zone.Tier)
		c.Set(ContextKeyZoneConfig, zone)
		c.Next()
	})
	router.Use(TIUMiddleware(fingerprint.NewAnalyzer(nil), nil))
	router.GET("/zones/z2/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequest(http.MethodGet, "/zones/z2/login", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected login bootstrap request to pass, got %d", recorder.Code)
	}
}

func TestTIUMiddlewareRejectsCriticalRequestWithoutSessionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := policy.NewRegistry()
	zone, _ := registry.GetZone(12)

	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneID, zone.ID)
		c.Set(ContextKeyZoneKey, zone.Key)
		c.Set(ContextKeyZoneName, zone.Name)
		c.Set(ContextKeyZoneTier, zone.Tier)
		c.Set(ContextKeyZoneConfig, zone)
		c.Next()
	})
	router.Use(TIUMiddleware(fingerprint.NewAnalyzer(nil), nil))
	router.POST("/zones/z12/control", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequest(http.MethodPost, "/zones/z12/control", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected critical request without token to be rejected, got %d", recorder.Code)
	}
}

func TestTIUMiddlewareRequiresSignedTIUTokenWhenSecretConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	os.Setenv("TIU_HMAC_SECRET", secret)
	defer os.Unsetenv("TIU_HMAC_SECRET")

	router := gin.New()
	registry := policy.NewRegistry()
	zone, _ := registry.GetZone(12)

	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneID, zone.ID)
		c.Set(ContextKeyZoneKey, zone.Key)
		c.Set(ContextKeyZoneName, zone.Name)
		c.Set(ContextKeyZoneTier, zone.Tier)
		c.Set(ContextKeyZoneConfig, zone)
		c.Next()
	})
	router.Use(TIUMiddleware(fingerprint.NewAnalyzer(nil), nil))
	router.POST("/zones/z12/control", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequest(http.MethodPost, "/zones/z12/control", nil)
	request.Header.Set("X-Codex-Token", "session")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != StatusTIURequired {
		t.Fatalf("expected request without TIU token to be rejected, got %d", recorder.Code)
	}
}

func TestTIUMiddlewareAcceptsValidSignedTIUToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	os.Setenv("TIU_HMAC_SECRET", secret)
	defer os.Unsetenv("TIU_HMAC_SECRET")

	router := gin.New()
	registry := policy.NewRegistry()
	zone, _ := registry.GetZone(12)

	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyZoneID, zone.ID)
		c.Set(ContextKeyZoneKey, zone.Key)
		c.Set(ContextKeyZoneName, zone.Name)
		c.Set(ContextKeyZoneTier, zone.Tier)
		c.Set(ContextKeyZoneConfig, zone)
		c.Next()
	})
	router.Use(TIUMiddleware(fingerprint.NewAnalyzer(nil), nil))
	router.POST("/zones/z12/control", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	now := time.Now().UTC().Unix()
	token := makeSignedTIUToken(t, secret, now, now+30, "")

	request := httptest.NewRequest(http.MethodPost, "/zones/z12/control", nil)
	request.Header.Set("X-Codex-Token", "session")
	request.Header.Set("X-Codex-TIU", token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected request with TIU token to pass, got %d", recorder.Code)
	}
}
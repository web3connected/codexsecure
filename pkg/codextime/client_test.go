package codextime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrentTIUParsesCodexTimeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/time/current-tiu" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tiu":213912663.431585,"timestamp":"2026-10-11T20:11:03.431627Z","epoch":"2020-01-01 00:00:00","cached":false}`))
	}))
	defer server.Close()
	t.Setenv("CODEXTIME_API_URL", server.URL)

	result, err := NewClient().CurrentTIU(context.Background())
	if err != nil {
		t.Fatalf("CurrentTIU: %v", err)
	}
	if result.TIU != 213912663.431585 || result.Epoch != "2020-01-01 00:00:00" || result.Source != "api" {
		t.Errorf("result = %+v", result)
	}
}

// A CodexTime with CODEXTIME_API_KEYS configured rejects keyless requests; the client sends CODEXTIME_API_KEY.
func TestRequestsSendTheConfiguredAPIKey(t *testing.T) {
	const key = "test-codextime-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"tiu":213912663.431585,"timestamp":"2026-10-11T20:11:03.431627Z","epoch":"2020-01-01 00:00:00","cached":false}`))
	}))
	defer server.Close()
	t.Setenv("CODEXTIME_API_URL", server.URL)
	t.Setenv("CODEXTIME_FAIL_OPEN", "false")

	t.Setenv("CODEXTIME_API_KEY", "")
	if _, err := NewClient().CurrentTIU(context.Background()); err == nil {
		t.Fatal("expected a keyless request to be rejected")
	}

	t.Setenv("CODEXTIME_API_KEY", key)
	client := NewClient()
	if result, err := client.CurrentTIU(context.Background()); err != nil || result.Source != "api" {
		t.Fatalf("CurrentTIU with key: %+v, %v", result, err)
	}
	if _, err := client.CurrentTIUExact(context.Background()); err != nil {
		t.Fatalf("CurrentTIUExact with key: %v", err)
	}
	if !client.Health(context.Background()) {
		t.Fatal("Health with key: expected true")
	}
}

// Documents the existing fail-open behaviour that callers must not treat as a TIU.
func TestCurrentTIUFailOpenReturnsZero(t *testing.T) {
	t.Setenv("CODEXTIME_API_URL", "http://127.0.0.1:1")
	result, err := NewClient().CurrentTIU(context.Background())
	if err != nil || result.TIU != 0 || result.Source != "unavailable" {
		t.Fatalf("expected fail-open zero TIU, got %+v, %v", result, err)
	}
}

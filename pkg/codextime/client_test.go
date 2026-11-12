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

// Documents the existing fail-open behaviour that callers must not treat as a TIU.
func TestCurrentTIUFailOpenReturnsZero(t *testing.T) {
	t.Setenv("CODEXTIME_API_URL", "http://127.0.0.1:1")
	result, err := NewClient().CurrentTIU(context.Background())
	if err != nil || result.TIU != 0 || result.Source != "unavailable" {
		t.Fatalf("expected fail-open zero TIU, got %+v, %v", result, err)
	}
}

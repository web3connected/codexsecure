package codexhash

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPayloadHashPostsToStatelessEndpoint(t *testing.T) {
	var got payloadHashRequest
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"payload_hash":"abc","algo":"sha512","hash_size":512}`))
	}))
	defer server.Close()
	t.Setenv("CODEXHASH_API_URL", server.URL)

	result, err := NewClient().PayloadHash(context.Background(), `{"a":1}`, "sha512", 512)
	if err != nil {
		t.Fatalf("PayloadHash: %v", err)
	}
	if path != "POST /api/hash/payload" {
		t.Errorf("request = %s, want POST /api/hash/payload (stateless, not the chaining POST /api/hash/)", path)
	}
	if got.Data != `{"a":1}` || got.Algo != "sha512" || got.HashSize != 512 {
		t.Errorf("request body = %+v", got)
	}
	if result.PayloadHash != "abc" || result.Algo != "sha512" || result.HashSize != 512 {
		t.Errorf("result = %+v", result)
	}
}

// PayloadHash must never substitute a local hash, even with fail-open enabled.
func TestPayloadHashFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"db down"}`, http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("CODEXHASH_API_URL", server.URL)
	t.Setenv("CODEXHASH_FAIL_OPEN", "true")

	result, err := NewClient().PayloadHash(context.Background(), "x", "sha512", 512)
	if !errors.Is(err, ErrUnavailable) || result.PayloadHash != "" {
		t.Fatalf("expected ErrUnavailable and no hash, got %+v, %v", result, err)
	}

	t.Setenv("CODEXHASH_ENABLED", "false")
	if _, err := NewClient().PayloadHash(context.Background(), "x", "sha512", 512); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("disabled client must fail closed, got %v", err)
	}
}

func TestPayloadHashUnreachable(t *testing.T) {
	t.Setenv("CODEXHASH_API_URL", "http://127.0.0.1:1")
	if _, err := NewClient().PayloadHash(context.Background(), "x", "sha512", 512); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

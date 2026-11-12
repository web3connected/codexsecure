package handlers

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

const fakeSHA512 = "7d225e21ff80847f8d612b011c208fb15e1f519829c5243e22b7de2339968c122d4c8148f3a9a6b1846d5da23c0e099aa3e5e7941aabb263a078bdf8f4d628b0"

type fakeHasher struct {
	calls  int
	data   []string
	algo   string
	size   int
	result codexhash.PayloadHashResult
	err    error
}

func (f *fakeHasher) PayloadHash(_ context.Context, data string, algo string, hashSize int) (codexhash.PayloadHashResult, error) {
	f.calls++
	f.data = append(f.data, data)
	f.algo, f.size = algo, hashSize
	return f.result, f.err
}

type fakeTIU struct {
	calls  int
	result codextime.TIUResponse
	err    error
}

func (f *fakeTIU) CurrentTIU(context.Context) (codextime.TIUResponse, error) {
	f.calls++
	return f.result, f.err
}

func newProtectRouter(h *ProtectHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/protect/codexhash", h.GenerateCodexHash)
	router.POST("/api/v1/protect/tiu", h.IssueTIU)
	return router
}

func postProtect(router *gin.Engine, path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func okHasher() *fakeHasher {
	return &fakeHasher{result: codexhash.PayloadHashResult{PayloadHash: fakeSHA512, Algo: "sha512", HashSize: 512}}
}

func okTIU() *fakeTIU {
	return &fakeTIU{result: codextime.TIUResponse{TIU: 213912663.431585, Timestamp: "2026-10-11T20:11:03.431627Z", Epoch: "2020-01-01 00:00:00", Source: "api"}}
}

func TestGenerateCodexHashReturnsServiceHash(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protect", "laravel_get_payload.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	hasher := okHasher()
	recorder := postProtect(newProtectRouter(NewProtectHandler(hasher, okTIU())), "/api/v1/protect/codexhash", string(body))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	if got["codexhash"] != fakeSHA512 || got["algorithm"] != "sha512" || got["representation"] != protect.Representation {
		t.Errorf("response = %v", got)
	}
	if _, ok := got["codexhash_event_id"]; ok || len(got) != 3 {
		t.Errorf("stateless response must contain only codexhash, algorithm, representation: %v", got)
	}

	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	_ = json.Unmarshal(body, &envelope)
	canonical, _ := protect.CanonicalPayload(envelope.Payload)
	if hasher.calls != 1 || hasher.data[0] != canonical || hasher.algo != "sha512" || hasher.size != 512 {
		t.Errorf("hasher got calls=%d algo=%s size=%d data=%s", hasher.calls, hasher.algo, hasher.size, hasher.data)
	}
}

// Equivalent payloads reach CodexHash as identical bytes, so CodexHash's
// deterministic payload hash is identical.
func TestGenerateCodexHashSendsEquivalentPayloadsIdentically(t *testing.T) {
	hasher := okHasher()
	router := newProtectRouter(NewProtectHandler(hasher, okTIU()))
	postProtect(router, "/api/v1/protect/codexhash", `{"payload":{"communication_uuid":"u-1","method":"GET","route":"example.route","zone":"z1"}}`)
	postProtect(router, "/api/v1/protect/codexhash", `{"payload":{"zone":"z1","route":"example.route","method":"GET","communication_uuid":"u-1"}}`)

	if hasher.calls != 2 || hasher.data[0] != hasher.data[1] {
		t.Fatalf("equivalent payloads sent differently: %q", hasher.data)
	}
	if want := `{"communication_uuid":"u-1","method":"GET","route":"example.route","zone":"z1"}`; hasher.data[0] != want {
		t.Errorf("hash input = %s, want %s", hasher.data[0], want)
	}
}

func TestProtectRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		code  int
		error string
	}{
		{"invalid json", `{"payload":`, http.StatusBadRequest, "invalid_request"},
		{"not an object", `[1]`, http.StatusBadRequest, "invalid_request"},
		{"missing payload", `{}`, http.StatusBadRequest, "invalid_request"},
		{"null payload", `{"payload":null}`, http.StatusUnprocessableEntity, "invalid_payload"},
		{"string payload", `{"payload":"route"}`, http.StatusUnprocessableEntity, "invalid_payload"},
		{"empty payload", `{"payload":{}}`, http.StatusUnprocessableEntity, "invalid_payload"},
		{"codexhash in input", `{"payload":{"route":"r","codexhash":"secret-hash-value"}}`, http.StatusUnprocessableEntity, "reserved_field"},
		{"tiu in input", `{"payload":{"route":"r","tiu":213912663.4}}`, http.StatusUnprocessableEntity, "reserved_field"},
		{"duplicate key", `{"payload":{"route":"secret-route","route":"r"}}`, http.StatusUnprocessableEntity, "duplicate_key"},
		{"too large", `{"payload":{"x":"` + strings.Repeat("a", protectMaxBodyBytes) + `"}}`, http.StatusRequestEntityTooLarge, "payload_too_large"},
	}

	for _, path := range []string{"/api/v1/protect/codexhash", "/api/v1/protect/tiu"} {
		hasher, tiu := okHasher(), okTIU()
		router := newProtectRouter(NewProtectHandler(hasher, tiu))
		for _, tc := range cases {
			recorder := postProtect(router, path, tc.body)
			var got map[string]any
			_ = json.Unmarshal(recorder.Body.Bytes(), &got)
			if recorder.Code != tc.code || got["error"] != tc.error {
				t.Errorf("%s %s: status=%d error=%v, want %d %s", path, tc.name, recorder.Code, got["error"], tc.code, tc.error)
			}
			if strings.Contains(recorder.Body.String(), "secret") {
				t.Errorf("%s %s: error response echoes payload content: %s", path, tc.name, recorder.Body.String())
			}
		}
		if hasher.calls != 0 || tiu.calls != 0 {
			t.Errorf("%s: services called for invalid requests (hash=%d tiu=%d)", path, hasher.calls, tiu.calls)
		}
	}
}

func TestGenerateCodexHashServiceFailures(t *testing.T) {
	cases := []struct {
		name   string
		hasher *fakeHasher
		code   int
		error  string
	}{
		{"unavailable", &fakeHasher{err: errors.New("codexhash unavailable: http: dial tcp codexhash-api:8001")}, http.StatusServiceUnavailable, "codexhash_unavailable"},
		{"wrong algorithm", &fakeHasher{result: codexhash.PayloadHashResult{PayloadHash: strings.Repeat("a", 64), Algo: "sha256"}}, http.StatusBadGateway, "invalid_service_response"},
		{"malformed hash", &fakeHasher{result: codexhash.PayloadHashResult{PayloadHash: "not-hex", Algo: "sha512"}}, http.StatusBadGateway, "invalid_service_response"},
	}
	for _, tc := range cases {
		recorder := postProtect(newProtectRouter(NewProtectHandler(tc.hasher, okTIU())), "/api/v1/protect/codexhash", `{"payload":{"route":"r"}}`)
		var got map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &got)
		if recorder.Code != tc.code || got["error"] != tc.error || got["codexhash"] != nil {
			t.Errorf("%s: status=%d body=%s", tc.name, recorder.Code, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "codexhash-api") || strings.Contains(recorder.Body.String(), "dial") {
			t.Errorf("%s: internal detail exposed: %s", tc.name, recorder.Body.String())
		}
	}
}

func TestIssueTIUReturnsCodexTimeTIU(t *testing.T) {
	tiu := okTIU()
	recorder := postProtect(newProtectRouter(NewProtectHandler(okHasher(), tiu)), "/api/v1/protect/tiu", `{"payload":{"communication_uuid":"u-1","method":"GET","route":"example.route","zone":"z1"}}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	// Formatting is preserved: a plain decimal, not exponent notation or a string.
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"tiu":213912663.431585`)) {
		t.Errorf("tiu formatting changed: %s", recorder.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	if got["epoch"] != "2020-01-01 00:00:00" || got["timestamp"] != "2026-10-11T20:11:03.431627Z" || got["source"] != "codextime" {
		t.Errorf("response = %v", got)
	}
	if tiu.calls != 1 {
		t.Errorf("CodexTime called %d times, want 1", tiu.calls)
	}
}

// The CodexTime client fails open with TIU 0; that must never be returned as a TIU.
func TestIssueTIUServiceFailures(t *testing.T) {
	cases := map[string]*fakeTIU{
		"fail-open zero":   {result: codextime.TIUResponse{TIU: 0, Source: "unavailable"}},
		"disabled":         {result: codextime.TIUResponse{Source: "disabled"}},
		"client error":     {err: errors.New("codextime status 500")},
		"non-positive tiu": {result: codextime.TIUResponse{TIU: -1, Source: "api"}},
	}
	for name, tiu := range cases {
		recorder := postProtect(newProtectRouter(NewProtectHandler(okHasher(), tiu)), "/api/v1/protect/tiu", `{"payload":{"route":"r"}}`)
		var got map[string]any
		_ = json.Unmarshal(recorder.Body.Bytes(), &got)
		if recorder.Code != http.StatusServiceUnavailable || got["error"] != "tiu_unavailable" || got["tiu"] != nil {
			t.Errorf("%s: status=%d body=%s", name, recorder.Code, recorder.Body.String())
		}
	}
}

// No idempotency store exists: each request reads the current TIU once and
// creates no CodexTime event (GET /api/time/current-tiu has no side effects).
func TestIssueTIUDuplicateRequestsReadCurrentTIUEachTime(t *testing.T) {
	tiu := okTIU()
	router := newProtectRouter(NewProtectHandler(okHasher(), tiu))
	body := `{"payload":{"communication_uuid":"u-1","route":"r"}}`
	first := postProtect(router, "/api/v1/protect/tiu", body)
	second := postProtect(router, "/api/v1/protect/tiu", body)

	if first.Code != http.StatusOK || second.Code != http.StatusOK || tiu.calls != 2 {
		t.Fatalf("statuses %d/%d, CodexTime calls %d", first.Code, second.Code, tiu.calls)
	}
}

// TIU and CodexHash are independent: issuing a TIU never calls CodexHash.
func TestProtectOperationsAreIndependent(t *testing.T) {
	hasher, tiu := okHasher(), okTIU()
	router := newProtectRouter(NewProtectHandler(hasher, tiu))
	postProtect(router, "/api/v1/protect/tiu", `{"payload":{"route":"r"}}`)
	postProtect(router, "/api/v1/protect/codexhash", `{"payload":{"route":"r"}}`)
	if hasher.calls != 1 || tiu.calls != 1 {
		t.Errorf("hash calls=%d tiu calls=%d, want 1 each", hasher.calls, tiu.calls)
	}
}

// End to end within Go: the real codexhash.Client against a stand-in for
// codexhash-api's stateless POST /api/hash/payload, which hashes exactly the data it receives.
func TestGenerateCodexHashConsumesStatelessCodexHashService(t *testing.T) {
	var requests []string
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		var body struct {
			Data     string `json:"data"`
			Algo     string `json:"algo"`
			HashSize int    `json:"hash_size"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sum := sha512.Sum512([]byte(body.Data))
		_ = json.NewEncoder(w).Encode(map[string]any{"payload_hash": hex.EncodeToString(sum[:]), "algo": body.Algo, "hash_size": body.HashSize})
	}))
	defer service.Close()
	t.Setenv("CODEXHASH_API_URL", service.URL)

	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protect", "laravel_get_payload.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	recorder := postProtect(newProtectRouter(NewProtectHandler(codexhash.NewClient(), okTIU())), "/api/v1/protect/codexhash", string(body))

	var got map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	// fakeSHA512 is the Python-computed SHA-512 of the fixture's canonical form.
	if recorder.Code != http.StatusOK || got["codexhash"] != fakeSHA512 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(requests) != 1 || requests[0] != "POST /api/hash/payload" {
		t.Errorf("CodexHash requests = %v, want only the stateless POST /api/hash/payload", requests)
	}
}

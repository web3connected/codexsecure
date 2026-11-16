package codexhash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// I01: CodexSecure protection evidence comes from CodexHash v2, never from conventional SHA.

const v2Hex = "3ed79969f0d4fd65b24c2847d8b44c36d1eedc9834f3d80cbfb571c57d7b5b306033a5e517bbc535c7d67bb2b55e3110e977adf291808a5b77ebb320b733ad74"

func v2Response(profile string, bits int, digest, hex string) string {
	body, _ := json.Marshal(map[string]any{"algorithm": "ch2-tuplehash256", "version": 2, "profile": profile, "output_bits": bits,
		"output_bytes": bits / 8, "digest": digest, "digest_hex": hex, "context": nil})
	return string(body)
}

func TestV2HashPostsTheDocumentedContract(t *testing.T) {
	var path string
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request)
		_, _ = w.Write([]byte(v2Response("inv-dual/1", 512, "ch2:inv-dual/1:512:"+v2Hex, v2Hex)))
	}))
	defer server.Close()
	t.Setenv("CODEXHASH_API_URL", server.URL)

	result, err := NewClient().V2Hash(context.Background(), `{"amount":19.99,"tiu":1763028000.123456}`)

	if err != nil {
		t.Fatalf("V2Hash: %v", err)
	}
	if path != "/api/v2/hash" {
		t.Fatalf("path = %s", path)
	}
	want := map[string]any{"payload": map[string]any{"encoding": "utf8", "data": `{"amount":19.99,"tiu":1763028000.123456}`},
		"output_bits": float64(512), "profile": "inv-dual/1"}
	if got, _ := json.Marshal(request); string(got) != mustJSON(want) {
		t.Fatalf("request = %s, want %s", got, mustJSON(want))
	}
	if result.Digest != "ch2:inv-dual/1:512:"+v2Hex || result.Profile != "inv-dual/1" || result.OutputBits != 512 {
		t.Fatalf("result = %+v", result)
	}
}

func TestV2HashRejectsResponsesOutsideTheContract(t *testing.T) {
	for name, body := range map[string]string{
		"weave profile":         v2Response("inv-weave/1", 512, "ch2:inv-weave/1:512:"+v2Hex, v2Hex),
		"other size":            v2Response("inv-dual/1", 256, "ch2:inv-dual/1:256:"+v2Hex[:64], v2Hex[:64]),
		"digest and hex differ": v2Response("inv-dual/1", 512, "ch2:inv-dual/1:512:"+v2Hex, strings.Repeat("0", 128)),
		"bare legacy sha512":    v2Response("inv-dual/1", 512, v2Hex, v2Hex),
		"uppercase hex":         v2Response("inv-dual/1", 512, "ch2:inv-dual/1:512:"+strings.ToUpper(v2Hex), strings.ToUpper(v2Hex)),
		"not json":              "nope",
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			t.Setenv("CODEXHASH_API_URL", server.URL)

			result, err := NewClient().V2Hash(context.Background(), "x")

			if !errors.Is(err, ErrInvalidResponse) || result.Digest != "" {
				t.Fatalf("err = %v result = %+v", err, result)
			}
		})
	}
}

// V2Hash never substitutes a local or conventional hash, even with fail-open enabled.
func TestV2HashFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"unsupported_json_number"}}`))
	}))
	defer server.Close()
	t.Setenv("CODEXHASH_API_URL", server.URL)
	t.Setenv("CODEXHASH_FAIL_OPEN", "true")

	result, err := NewClient().V2Hash(context.Background(), "x")
	if !errors.Is(err, ErrUnavailable) || result.Digest != "" {
		t.Fatalf("err = %v result = %+v", err, result)
	}

	t.Setenv("CODEXHASH_API_URL", "http://127.0.0.1:1")
	if _, err := NewClient().V2Hash(context.Background(), "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unreachable: err = %v", err)
	}
	t.Setenv("CODEXHASH_ENABLED", "false")
	if _, err := NewClient().V2Hash(context.Background(), "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("disabled: err = %v", err)
	}
}

// Live CodexHash v2 (skipped unless CODEXHASH_V2_LIVE_URL is set): exact decimal and fractional numbers are
// hashed, the unchanged text verifies through /api/v2/hash/verify, and a modified text does not.
func TestV2HashLiveService(t *testing.T) {
	url := os.Getenv("CODEXHASH_V2_LIVE_URL")
	if url == "" {
		t.Skip("CODEXHASH_V2_LIVE_URL not set")
	}
	t.Setenv("CODEXHASH_API_URL", url)
	text := `{"payload":{"form":{"amount":19.99,"name":"Ada"}},"validatedHash":["ch2:inv-dual/1:512:` + v2Hex + `",{"validated":true,"validatedAt":"2026-11-13T10:15:42.123456Z","validationTiu":1763028000.123456},1763028000.5]}`

	result, err := NewClient().V2Hash(context.Background(), text)
	if err != nil {
		t.Fatalf("V2Hash: %v", err)
	}
	again, err := NewClient().V2Hash(context.Background(), text)
	if err != nil || again.Digest != result.Digest {
		t.Fatalf("not deterministic: %v %s %s", err, result.Digest, again.Digest)
	}
	for _, tc := range []struct {
		text  string
		valid bool
	}{{text, true}, {strings.Replace(text, "19.99", "19.98", 1), false}} {
		body, _ := json.Marshal(map[string]any{"payload": map[string]any{"encoding": "utf8", "data": tc.text}, "digest": result.Digest})
		resp, err := http.Post(url+"/api/v2/hash/verify", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
		var verified struct {
			Valid bool `json:"valid"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&verified)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || verified.Valid != tc.valid {
			t.Fatalf("verify status = %d valid = %v, want %v", resp.StatusCode, verified.Valid, tc.valid)
		}
	}
	t.Logf("live digest %s", result.Digest)
}

func mustJSON(value any) string {
	body, _ := json.Marshal(value)
	return string(body)
}

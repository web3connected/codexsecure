package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// I01: the Seven-Fold Send Hash and validatedHash use CodexHash v2 through POST /api/v1/protect/codexhash/v2.

const fakeV2Hex = "3ed79969f0d4fd65b24c2847d8b44c36d1eedc9834f3d80cbfb571c57d7b5b306033a5e517bbc535c7d67bb2b55e3110e977adf291808a5b77ebb320b733ad74"

type fakeV2Hasher struct {
	fakeHasher
	v2Calls  int
	v2Data   []string
	v2Result codexhash.V2HashResult
	v2Err    error
}

func (f *fakeV2Hasher) V2Hash(_ context.Context, text string) (codexhash.V2HashResult, error) {
	f.v2Calls++
	f.v2Data = append(f.v2Data, text)
	return f.v2Result, f.v2Err
}

func okV2Hasher() *fakeV2Hasher {
	return &fakeV2Hasher{v2Result: codexhash.V2HashResult{Algorithm: codexhash.V2Algorithm, Version: 2, Profile: codexhash.V2Profile,
		OutputBits: codexhash.V2OutputBits, Digest: "ch2:inv-dual/1:512:" + fakeV2Hex, DigestHex: fakeV2Hex}}
}

func newProtectV2Router(h *ProtectHandler) *gin.Engine {
	router := newProtectRouter(h)
	router.POST("/api/v1/protect/codexhash/v2", h.GenerateCodexHashV2)
	return router
}

func TestGenerateCodexHashV2HashesTheCanonicalPayloadWithCodexHashV2(t *testing.T) {
	hasher := okV2Hasher()
	router := newProtectV2Router(NewProtectHandler(hasher, okTIU()))
	// Decimal and fractional numbers (a form value, a CodexTime TIU) are kept exactly as written.
	body := `{"payload":{"z":1,"amount":19.99,"validation":{"tiu":1763028000.123456},"a":{"y":"Ö/<>","x":[2,1]}}}`

	recorder := postProtect(router, "/api/v1/protect/codexhash/v2", body)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	want, err := protect.CanonicalPayload(json.RawMessage(`{"z":1,"amount":19.99,"validation":{"tiu":1763028000.123456},"a":{"y":"Ö/<>","x":[2,1]}}`))
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	if hasher.v2Calls != 1 || hasher.v2Data[0] != want || hasher.calls != 0 {
		t.Fatalf("v2 calls = %d data = %q legacy calls = %d", hasher.v2Calls, hasher.v2Data, hasher.calls)
	}
	if !strings.Contains(want, `"amount":19.99`) || !strings.Contains(want, `"validation":{"tiu":1763028000.123456}`) {
		t.Fatalf("canonical form changed numbers: %s", want)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	expected := map[string]any{"codexhash": "ch2:inv-dual/1:512:" + fakeV2Hex, "algorithm": "ch2-tuplehash256", "profile": "inv-dual/1",
		"output_bits": float64(512), "encoding": "utf8", "representation": protect.Representation}
	if fmt.Sprint(response) != fmt.Sprint(expected) {
		t.Fatalf("response = %v, want %v", response, expected)
	}
}

func TestGenerateCodexHashV2FailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"service unavailable": {fmt.Errorf("%w: codexhash status 503", codexhash.ErrUnavailable), http.StatusServiceUnavailable, "codexhash_unavailable"},
		"invalid response":    {fmt.Errorf("%w: not a digest", codexhash.ErrInvalidResponse), http.StatusBadGateway, "invalid_service_response"},
	} {
		t.Run(name, func(t *testing.T) {
			hasher := okV2Hasher()
			hasher.v2Err = tc.err
			router := newProtectV2Router(NewProtectHandler(hasher, okTIU()))

			recorder := postProtect(router, "/api/v1/protect/codexhash/v2", `{"payload":{"a":1}}`)

			if recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), tc.code) || strings.Contains(recorder.Body.String(), "ch2:") {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			if hasher.calls != 0 {
				t.Fatalf("legacy SHA-512 path used as a fallback: %d calls", hasher.calls)
			}
		})
	}
}

func TestGenerateCodexHashV2RejectsInvalidPayloadsBeforeHashing(t *testing.T) {
	hasher := okV2Hasher()
	router := newProtectV2Router(NewProtectHandler(hasher, okTIU()))
	for _, body := range []string{`{"payload":{}}`, `{"payload":{"codexhash":"x"}}`, `{"payload":[1]}`, `not json`} {
		if recorder := postProtect(router, "/api/v1/protect/codexhash/v2", body); recorder.Code < 400 {
			t.Fatalf("%s: status = %d", body, recorder.Code)
		}
	}
	if hasher.v2Calls != 0 {
		t.Fatalf("hashed an invalid payload")
	}
}

func TestLegacyCodexHashRouteIsUnchangedForPendingConsumers(t *testing.T) {
	hasher := okV2Hasher()
	hasher.result = codexhash.PayloadHashResult{PayloadHash: fakeSHA512, Algo: "sha512", HashSize: 512}
	router := newProtectV2Router(NewProtectHandler(hasher, okTIU()))

	recorder := postProtect(router, "/api/v1/protect/codexhash", `{"payload":{"a":1}}`)

	if recorder.Code != http.StatusOK || hasher.calls != 1 || hasher.v2Calls != 0 || !strings.Contains(recorder.Body.String(), fakeSHA512) {
		t.Fatalf("status = %d legacy = %d v2 = %d", recorder.Code, hasher.calls, hasher.v2Calls)
	}
}

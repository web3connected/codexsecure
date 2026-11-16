package handlers

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// pageMetaV2 stands in for CodexHash v2 (POST /api/v2/hash) behind codexhash.Client.V2Hash.
// For the captured homepage data it returns the digest the live engine produced (the fixture's
// security.codexhash). For any other text it returns a TEST DOUBLE digest in the v2 form, so that
// changed data yields a different digest; it is not CodexHash.
type pageMetaV2 struct {
	calls int
	err   error
	known map[string]string
}

func (h *pageMetaV2) V2Hash(_ context.Context, text string) (codexhash.V2HashResult, error) {
	h.calls++
	if h.err != nil {
		return codexhash.V2HashResult{}, h.err
	}
	digest, ok := h.known[text]
	if !ok {
		sum := sha512.Sum512([]byte(text)) // TEST DOUBLE stand-in, never accepted as evidence on its own form
		digest = "ch2:inv-dual/1:512:" + hex.EncodeToString(sum[:])
	}
	return codexhash.V2HashResult{Algorithm: codexhash.V2Algorithm, Version: 2, Profile: codexhash.V2Profile,
		OutputBits: codexhash.V2OutputBits, Digest: digest, DigestHex: strings.TrimPrefix(digest, "ch2:inv-dual/1:512:")}, nil
}

// newPageMetaV2 knows the captured homepage package's canonical data and its live CodexHash v2 digest.
func newPageMetaV2(t *testing.T) *pageMetaV2 {
	t.Helper()
	pkg := homepagePackage(t)
	raw, err := json.Marshal(pkg["data"])
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := protect.CanonicalPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &pageMetaV2{known: map[string]string{canonical: pkg["security"].(map[string]any)["codexhash"].(string)}}
}

type fakePageMetaStore struct {
	calls   []database.PageMetaRecord
	results []database.PageMetaResult
	err     error
}

func (s *fakePageMetaStore) RecordPageMeta(_ context.Context, record database.PageMetaRecord) (database.PageMetaResult, error) {
	s.calls = append(s.calls, record)
	if s.err != nil {
		return database.PageMetaResult{}, s.err
	}
	result := s.results[0]
	if len(s.results) > 1 {
		s.results = s.results[1:]
	}
	return result, nil
}

func readPackage(t *testing.T, name string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pagemeta", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	var pkg map[string]any
	if err := decoder.Decode(&pkg); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// homepagePackage is the homepage package Laravel sent and the live service recorded with
// CodexHash v2 evidence (captured from codex_zones after the I01 PageMeta correction).
func homepagePackage(t *testing.T) map[string]any {
	return readPackage(t, "homepage_package.v2.captured.json")
}

// legacyHomepagePackage is the same page captured in October 2026 with the SHA-512 hash
// that I01 corrected; it must now be refused.
func legacyHomepagePackage(t *testing.T) map[string]any {
	return readPackage(t, "homepage_package.captured.json")
}

func postPageMeta(h *PageMetaHandler, body any) (*http.Response, map[string]any) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/pagemeta/record", h.Record)
	var raw []byte
	switch b := body.(type) {
	case string:
		raw = []byte(b)
	default:
		raw, _ = json.Marshal(map[string]any{"package": b})
	}
	recorder := postProtect(router, "/api/v1/pagemeta/record", string(raw))
	var decoded map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Result(), decoded
}

var recordedPackage = database.PageMetaPackage{
	UUID: "0f8fad5b-d9cb-469f-a165-70867728950e", Hash: "ch2:inv-dual/1:512:d40f91fc118fc83178b28db60a9bbe65036c43e8ad54232c15bff551fa590d4115963dff949b35d395d3669f2fdde65bd9d0468664fd6c2b55b651d751a3faad",
	TIU: "214014006.591195", CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC(),
}

func TestPageMetaRecordsHomepagePackage(t *testing.T) {
	store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaRecorded, Recorded: true, Package: &recordedPackage, KnownPackages: []database.PageMetaPackage{}}}}
	hasher := newPageMetaV2(t)
	pkg := homepagePackage(t)

	res, body := postPageMeta(NewPageMetaHandler(hasher, store), pkg)

	if res.StatusCode != http.StatusCreated || body["result"] != "recorded" || body["recorded"] != true {
		t.Fatalf("status=%d body=%v", res.StatusCode, body)
	}
	if len(store.calls) != 1 || hasher.calls != 1 {
		t.Fatalf("store calls=%d hasher calls=%d", len(store.calls), hasher.calls)
	}
	record := store.calls[0]
	security := pkg["security"].(map[string]any)
	if record.Method != "GET" || record.URI != "/" || record.Hash != security["codexhash"] || !codexhash.IsV2Evidence(record.Hash) {
		t.Errorf("record = %+v", record)
	}
	if record.TIU != security["tiu"].(json.Number).String() {
		t.Errorf("tiu = %s", record.TIU)
	}
	var stored map[string]any
	if err := json.Unmarshal(record.Payload, &stored); err != nil || stored["package_uuid"] != nil || stored["data"] == nil || stored["submitted_json"] == nil {
		t.Errorf("payload must be the complete package as received: %s", record.Payload)
	}
	page := body["page"].(map[string]any)
	if page["method"] != "GET" || page["uri"] != "/" || page["route_name"] != "home" {
		t.Errorf("page = %v", page)
	}
	if body["package"].(map[string]any)["uuid"] != recordedPackage.UUID {
		t.Errorf("package = %v", body["package"])
	}
}

func TestPageMetaReportsKnownAndChangedResults(t *testing.T) {
	store := &fakePageMetaStore{results: []database.PageMetaResult{
		{Result: database.PageMetaKnown, Package: &recordedPackage, KnownPackages: []database.PageMetaPackage{}},
		{Result: database.PageMetaConfigurationChanged, KnownPackages: []database.PageMetaPackage{recordedPackage}},
	}}
	handler := NewPageMetaHandler(newPageMetaV2(t), store)

	res, body := postPageMeta(handler, homepagePackage(t))
	if res.StatusCode != http.StatusOK || body["result"] != "known" || body["recorded"] != false || body["package"].(map[string]any)["uuid"] != recordedPackage.UUID {
		t.Fatalf("known: status=%d body=%v", res.StatusCode, body)
	}

	res, body = postPageMeta(handler, homepagePackage(t))
	if res.StatusCode != http.StatusOK || body["result"] != "configuration_changed" || body["recorded"] != false || body["package"] != nil {
		t.Fatalf("changed: status=%d body=%v", res.StatusCode, body)
	}
	if known := body["known_packages"].([]any); len(known) != 1 || !strings.Contains(body["message"].(string), "not recorded") {
		t.Errorf("changed body = %v", body)
	}
}

func TestPageMetaRejectsInvalidPackages(t *testing.T) {
	mutate := func(f func(map[string]any)) map[string]any {
		pkg := homepagePackage(t)
		f(pkg)
		return pkg
	}
	data := func(pkg map[string]any) map[string]any { return pkg["data"].(map[string]any) }
	hash := func(pkg map[string]any) string { return pkg["security"].(map[string]any)["codexhash"].(string) }
	setHash := func(v string) func(map[string]any) {
		return func(p map[string]any) { p["security"].(map[string]any)["codexhash"] = v }
	}
	hexPart := strings.TrimPrefix(hash(homepagePackage(t)), "ch2:inv-dual/1:512:")
	cases := []struct {
		name   string
		body   any
		status int
		code   string
	}{
		{"not json", `{"package":`, 400, "invalid_request"},
		{"missing package", `{}`, 400, "invalid_request"},
		{"package not object", `{"package":"x"}`, 400, "invalid_request"},
		{"data missing", mutate(func(p map[string]any) { delete(p, "data") }), 422, "invalid_package"},
		{"data empty", mutate(func(p map[string]any) { p["data"] = map[string]any{} }), 422, "invalid_package"},
		{"lowercase method", mutate(func(p map[string]any) { data(p)["method"] = "get" }), 422, "invalid_package"},
		{"route uri missing", mutate(func(p map[string]any) { delete(data(p)["route"].(map[string]any), "uri") }), 422, "invalid_package"},
		{"bad zone", mutate(func(p map[string]any) { data(p)["declared_zone"] = "Z13" }), 422, "invalid_package"},
		{"reserved tiu in data", mutate(func(p map[string]any) { data(p)["tiu"] = 1 }), 422, "reserved_field"},
		{"bad package_uuid", mutate(func(p map[string]any) { p["package_uuid"] = "not-a-uuid" }), 422, "invalid_package"},
		{"validation not object", mutate(func(p map[string]any) { p["validation"] = "valid" }), 422, "invalid_package"},
		{"submitted_json for other data", mutate(func(p map[string]any) { p["submitted_json"] = `{"payload":{"method":"GET"}}` }), 422, "submitted_json_mismatch"},
		// I01: only CodexHash v2 evidence (ch2:inv-dual/1:512:<lowercase hex>) is accepted.
		{"uppercase hash", mutate(setHash(strings.ToUpper(hash(homepagePackage(t))))), 422, "invalid_package"},
		{"uppercase hex part", mutate(setHash("ch2:inv-dual/1:512:" + strings.ToUpper(hexPart))), 422, "invalid_package"},
		{"bare 128-hex (SHA-512 form)", mutate(setHash(hexPart)), 422, "invalid_package"},
		{"256-bit CodexHash form", mutate(setHash("ch2:inv-dual/1:256:" + hexPart[:64])), 422, "invalid_package"},
		{"installation weave profile", mutate(setHash("ch2:inv-weave/1:512:" + hexPart)), 422, "invalid_package"},
		{"short hex", mutate(setHash("ch2:inv-dual/1:512:" + hexPart[1:])), 422, "invalid_package"},
		{"missing security", mutate(func(p map[string]any) { delete(p, "security") }), 422, "invalid_package"},
		{"tiu zero", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = 0 }), 422, "invalid_package"},
		{"tiu negative", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = -1 }), 422, "invalid_package"},
		{"tiu too precise", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = json.Number("1.1234567") }), 422, "invalid_package"},
		{"tiu as text", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = "soon" }), 422, "invalid_package"},
		{"legacy SHA-512 captured package", legacyHomepagePackage(t), 422, "invalid_package"},
	}

	for _, tc := range cases {
		store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaKnown}}}
		hasher := newPageMetaV2(t)
		res, body := postPageMeta(NewPageMetaHandler(hasher, store), tc.body)
		if res.StatusCode != tc.status || body["error"] != tc.code {
			t.Errorf("%s: status=%d error=%v, want %d %s", tc.name, res.StatusCode, body["error"], tc.status, tc.code)
		}
		if len(store.calls) != 0 || hasher.calls != 0 {
			t.Errorf("%s: invalid package reached CodexHash (%d) or the database (%d)", tc.name, hasher.calls, len(store.calls))
		}
	}
}

// A hash that does not belong to the data (for example data changed after Protect),
// or altered evidence, is rejected before anything is recorded.
func TestPageMetaRejectsHashNotMatchingData(t *testing.T) {
	changed := homepagePackage(t)
	changed["data"].(map[string]any)["view"] = "welcome"
	changed["submitted_json"] = nil

	altered := homepagePackage(t)
	security := altered["security"].(map[string]any)
	h := security["codexhash"].(string)
	last := "a"
	if strings.HasSuffix(h, "a") {
		last = "b"
	}
	security["codexhash"] = h[:len(h)-1] + last

	for name, pkg := range map[string]map[string]any{"data changed": changed, "evidence altered": altered} {
		store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaKnown}}}
		res, body := postPageMeta(NewPageMetaHandler(newPageMetaV2(t), store), pkg)
		if res.StatusCode != http.StatusUnprocessableEntity || body["error"] != "hash_mismatch" || len(store.calls) != 0 {
			t.Fatalf("%s: status=%d body=%v store calls=%d", name, res.StatusCode, body, len(store.calls))
		}
	}
}

func TestPageMetaServiceFailuresAreControlled(t *testing.T) {
	var logs bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&logs)
	defer func() { log.Logger = previous }()

	secret := "postgres://codex_zones_runtime:s3cr3t-runtime@host.docker.internal:5432/codex_zones"

	// CodexHash v2 unavailable: fail closed, no conventional-hash fallback, nothing recorded.
	store := &fakePageMetaStore{}
	res, body := postPageMeta(NewPageMetaHandler(&pageMetaV2{err: errors.Join(codexhash.ErrUnavailable, errors.New("dial "+secret))}, store), homepagePackage(t))
	if res.StatusCode != http.StatusServiceUnavailable || body["error"] != "codexhash_unavailable" || len(store.calls) != 0 {
		t.Fatalf("codexhash down: status=%d body=%v", res.StatusCode, body)
	}

	// A response that breaks the CodexHash v2 contract.
	res, body = postPageMeta(NewPageMetaHandler(&pageMetaV2{err: codexhash.ErrInvalidResponse}, store), homepagePackage(t))
	if res.StatusCode != http.StatusBadGateway || body["error"] != "invalid_service_response" || len(store.calls) != 0 {
		t.Fatalf("invalid response: status=%d body=%v", res.StatusCode, body)
	}

	// No CodexHash v2 client at all.
	res, body = postPageMeta(NewPageMetaHandler(nil, store), homepagePackage(t))
	if res.StatusCode != http.StatusServiceUnavailable || body["error"] != "codexhash_unavailable" || len(store.calls) != 0 {
		t.Fatalf("no client: status=%d body=%v", res.StatusCode, body)
	}

	store = &fakePageMetaStore{err: errors.New("codexzone database unavailable: " + secret)}
	res, body = postPageMeta(NewPageMetaHandler(newPageMetaV2(t), store), homepagePackage(t))
	if res.StatusCode != http.StatusServiceUnavailable || body["error"] != "database_unavailable" {
		t.Fatalf("database down: status=%d body=%v", res.StatusCode, body)
	}
	for _, output := range []string{logs.String(), body["message"].(string)} {
		if strings.Contains(output, "s3cr3t") || strings.Contains(output, "postgres://") {
			t.Errorf("credentials exposed: %s", output)
		}
	}
}

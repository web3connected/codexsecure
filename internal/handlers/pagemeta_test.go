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
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// sha512Hasher stands in for CodexHash's stateless POST /api/hash/payload.
type sha512Hasher struct {
	calls int
	err   error
}

func (h *sha512Hasher) PayloadHash(_ context.Context, data string, algo string, size int) (codexhash.PayloadHashResult, error) {
	h.calls++
	if h.err != nil {
		return codexhash.PayloadHashResult{}, h.err
	}
	sum := sha512.Sum512([]byte(data))
	return codexhash.PayloadHashResult{PayloadHash: hex.EncodeToString(sum[:]), Algo: algo, HashSize: size}, nil
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

func homepagePackage(t *testing.T) map[string]any {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pagemeta", "homepage_package.captured.json"))
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
	UUID: "0f8fad5b-d9cb-469f-a165-70867728950e", Hash: "76794aa325110fc995dde9eb01198c61a3133f6f6c79a6c085da1dc25db8f3ab8b51d3549d96819b63314ed13784f61f6f7c2974364bbda923d39d34a233b7da",
	TIU: "214014006.591195", CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC(),
}

func TestPageMetaRecordsHomepagePackage(t *testing.T) {
	store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaRecorded, Recorded: true, Package: &recordedPackage, KnownPackages: []database.PageMetaPackage{}}}}
	hasher := &sha512Hasher{}
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
	if record.Method != "GET" || record.URI != "/" || record.Hash != security["codexhash"] || record.TIU != "214014006.591195" {
		t.Errorf("record = %+v", record)
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
	handler := NewPageMetaHandler(&sha512Hasher{}, store)

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
		{"uppercase hash", mutate(func(p map[string]any) {
			s := p["security"].(map[string]any)
			s["codexhash"] = strings.ToUpper(s["codexhash"].(string))
		}), 422, "invalid_package"},
		{"missing security", mutate(func(p map[string]any) { delete(p, "security") }), 422, "invalid_package"},
		{"tiu zero", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = 0 }), 422, "invalid_package"},
		{"tiu negative", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = -1 }), 422, "invalid_package"},
		{"tiu too precise", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = json.Number("1.1234567") }), 422, "invalid_package"},
		{"tiu as text", mutate(func(p map[string]any) { p["security"].(map[string]any)["tiu"] = "soon" }), 422, "invalid_package"},
	}

	for _, tc := range cases {
		store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaKnown}}}
		res, body := postPageMeta(NewPageMetaHandler(&sha512Hasher{}, store), tc.body)
		if res.StatusCode != tc.status || body["error"] != tc.code {
			t.Errorf("%s: status=%d error=%v, want %d %s", tc.name, res.StatusCode, body["error"], tc.status, tc.code)
		}
		if len(store.calls) != 0 {
			t.Errorf("%s: invalid package reached the database", tc.name)
		}
	}
}

// A hash that does not belong to the data (for example data changed after Protect)
// is rejected before anything is recorded.
func TestPageMetaRejectsHashNotMatchingData(t *testing.T) {
	pkg := homepagePackage(t)
	pkg["data"].(map[string]any)["view"] = "welcome"
	pkg["submitted_json"] = nil
	store := &fakePageMetaStore{results: []database.PageMetaResult{{Result: database.PageMetaKnown}}}

	res, body := postPageMeta(NewPageMetaHandler(&sha512Hasher{}, store), pkg)
	if res.StatusCode != http.StatusUnprocessableEntity || body["error"] != "hash_mismatch" || len(store.calls) != 0 {
		t.Fatalf("status=%d body=%v store calls=%d", res.StatusCode, body, len(store.calls))
	}
}

func TestPageMetaServiceFailuresAreControlled(t *testing.T) {
	var logs bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&logs)
	defer func() { log.Logger = previous }()

	secret := "postgres://codex_zones_runtime:s3cr3t-runtime@host.docker.internal:5432/codex_zones"

	store := &fakePageMetaStore{}
	res, body := postPageMeta(NewPageMetaHandler(&sha512Hasher{err: errors.New("dial " + secret)}, store), homepagePackage(t))
	if res.StatusCode != http.StatusServiceUnavailable || body["error"] != "codexhash_unavailable" || len(store.calls) != 0 {
		t.Fatalf("codexhash down: status=%d body=%v", res.StatusCode, body)
	}

	store = &fakePageMetaStore{err: errors.New("codexzone database unavailable: " + secret)}
	res, body = postPageMeta(NewPageMetaHandler(&sha512Hasher{}, store), homepagePackage(t))
	if res.StatusCode != http.StatusServiceUnavailable || body["error"] != "database_unavailable" {
		t.Fatalf("database down: status=%d body=%v", res.StatusCode, body)
	}
	for _, output := range []string{logs.String(), body["message"].(string)} {
		if strings.Contains(output, "s3cr3t") || strings.Contains(output, "postgres://") {
			t.Errorf("credentials exposed: %s", output)
		}
	}
}

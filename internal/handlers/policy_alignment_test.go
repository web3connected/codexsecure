package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/web3connected/codexsecure-api/internal/policy"
)

func newPolicyRouter(registry *policy.Registry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHandler(registry)
	router := gin.New()
	router.GET("/api/v1/policy/registry", handler.GetRegistry)
	router.POST("/api/v1/policy/alignment", handler.CheckAlignment)
	return router
}

func TestCheckAlignmentReturnsRegistryComparison(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zones", "canonical_registry.json"))
	if err != nil {
		t.Fatalf("read canonical registry: %v", err)
	}

	registry := policy.NewRegistry()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	newPolicyRouter(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var got policy.AlignmentResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var fixture struct {
		Zones map[string]string `json:"zones"`
	}
	_ = json.Unmarshal(body, &fixture)
	expected, _ := policy.ParseExpectedRegistry(fixture.Zones)
	want := registry.Alignment(expected)

	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if !bytes.Equal(wantJSON, gotJSON) {
		t.Errorf("endpoint result differs from registry comparison\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestCheckAlignmentRejectsMalformedRegistries(t *testing.T) {
	cases := []struct {
		name string
		body string
		code int
	}{
		{"invalid json", `{"zones":`, http.StatusBadRequest},
		{"missing zones", `{}`, http.StatusBadRequest},
		{"bad identifier", `{"zones":{"Zx":"public"}}`, http.StatusUnprocessableEntity},
		{"duplicate zone", `{"zones":{"Z3":"ops","3":"ops"}}`, http.StatusUnprocessableEntity},
	}

	router := newPolicyRouter(policy.NewRegistry())
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment", bytes.NewBufferString(tc.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)

		if recorder.Code != tc.code {
			t.Errorf("%s: status = %d, want %d (body %s)", tc.name, recorder.Code, tc.code, recorder.Body.String())
		}
	}
}

func TestGetRegistryReturnsCanonicalEntriesAndSignature(t *testing.T) {
	registry := policy.NewRegistry()
	recorder := httptest.NewRecorder()
	newPolicyRouter(registry).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/policy/registry", nil))

	var got struct {
		Signature string                 `json:"signature"`
		Total     int                    `json:"total"`
		Zones     []policy.RegistryEntry `json:"zones"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 13 || len(got.Zones) != 13 {
		t.Fatalf("expected 13 zones, got %d", len(got.Zones))
	}
	for i, entry := range got.Zones {
		if entry.ZoneID != i {
			t.Errorf("zones[%d] has zone_id %d; entries must be ordered by ID", i, entry.ZoneID)
		}
	}
	if got.Signature != policy.RegistrySignature(registry.KeyRegistry()) {
		t.Error("signature does not match the registry")
	}
}

func newDefinitionRouter(registry *policy.Registry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHandler(registry)
	router := gin.New()
	router.GET("/api/v1/policy/definitions", handler.GetDefinitions)
	router.POST("/api/v1/policy/alignment/definitions", handler.CheckDefinitionAlignment)
	return router
}

func TestCheckDefinitionAlignmentReturnsRegistryComparison(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zones", "canonical_definitions.json"))
	if err != nil {
		t.Fatalf("read canonical definitions: %v", err)
	}

	registry := policy.NewRegistry()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment/definitions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	newDefinitionRouter(registry).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var fixture struct {
		Zones map[string]policy.ExpectedDefinition `json:"zones"`
	}
	_ = json.Unmarshal(body, &fixture)
	expected, err := policy.ParseExpectedDefinitions(fixture.Zones)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	wantJSON, _ := json.Marshal(registry.DefinitionAlignment(expected))

	var got map[string]any
	var want map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	_ = json.Unmarshal(wantJSON, &want)
	gotJSON, _ := json.Marshal(got)
	wantNormalized, _ := json.Marshal(want)
	if !bytes.Equal(gotJSON, wantNormalized) {
		t.Errorf("endpoint result differs from registry comparison\n got: %s\nwant: %s", gotJSON, wantNormalized)
	}
}

func TestCheckDefinitionAlignmentRejectsMalformedDefinitions(t *testing.T) {
	cases := []struct {
		name string
		body string
		code int
	}{
		{"invalid json", `{"zones":`, http.StatusBadRequest},
		{"missing zones", `{}`, http.StatusBadRequest},
		{"wrong type", `{"zones":{"Z1":"public"}}`, http.StatusBadRequest},
		{"missing read_only", `{"zones":{"Z1":{"key":"public","name":"Public","rules":{"methods_allowed":["GET"]}}}}`, http.StatusUnprocessableEntity},
		{"invalid method", `{"zones":{"Z1":{"key":"public","name":"Public","rules":{"read_only":false,"methods_allowed":["FETCH"]}}}}`, http.StatusUnprocessableEntity},
	}

	router := newDefinitionRouter(policy.NewRegistry())
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment/definitions", bytes.NewBufferString(tc.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)

		if recorder.Code != tc.code {
			t.Errorf("%s: status = %d, want %d (body %s)", tc.name, recorder.Code, tc.code, recorder.Body.String())
		}
	}
}

func TestGetDefinitionsReturnsContractsAndSignature(t *testing.T) {
	registry := policy.NewRegistry()
	recorder := httptest.NewRecorder()
	newDefinitionRouter(registry).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/policy/definitions", nil))

	var got struct {
		SignatureAlgorithm string `json:"signature_algorithm"`
		Signature          string `json:"signature"`
		Zones              []struct {
			ZoneID     int                       `json:"zone_id"`
			Definition policy.DefinitionContract `json:"definition"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Zones) != 13 || got.SignatureAlgorithm != policy.DefinitionSignatureAlgorithm {
		t.Fatalf("zones=%d algorithm=%s", len(got.Zones), got.SignatureAlgorithm)
	}
	if got.Signature != policy.DefinitionSignature(registry.DefinitionContracts()) {
		t.Error("signature does not match the registry definitions")
	}
	if got.Zones[0].Definition.Role == "" {
		t.Error("Z0 definition must carry its role")
	}
}

// The registry alignment response is validated field-by-field by the Laravel
// SDK; its shape must not change when definition alignment is added.
func TestRegistryAlignmentResponseShapeIsUnchanged(t *testing.T) {
	body, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "zones", "canonical_registry.json"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	newPolicyRouter(policy.NewRegistry()).ServeHTTP(recorder, request)

	var got map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	want := []string{"aligned", "status", "policy_version", "signature_algorithm", "signature", "expected_signature", "zones_checked", "matched", "differences"}
	if len(got) != len(want) {
		t.Errorf("registry alignment fields = %d, want %d: %v", len(got), len(want), got)
	}
	for _, field := range want {
		if _, ok := got[field]; !ok {
			t.Errorf("registry alignment response is missing %q", field)
		}
	}
	if got["signature_algorithm"] != policy.SignatureAlgorithm {
		t.Errorf("signature_algorithm = %v", got["signature_algorithm"])
	}
}

func postAlignment(t *testing.T, body string) (int, map[string]any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/policy/alignment", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	newPolicyRouter(policy.NewRegistry()).ServeHTTP(recorder, request)

	var got map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &got)
	return recorder.Code, got
}

// "admin" and "administration" are the same Z12; the alias is reported, not mismatched.
func TestCheckAlignmentTreatsAliasesAsTheSameZone(t *testing.T) {
	body := `{"zones":{"Z0":"boundary","Z1":"public","Z2":"identity","Z3":"operations","Z4":"protected",
		"Z5":"processing","Z6":"logging","Z7":"communications","Z8":"payments","Z9":"analytics",
		"Z10":"streaming","Z11":"synchronization","Z12":"admin"}}`
	code, got := postAlignment(t, body)

	if code != http.StatusOK || got["aligned"] != true || got["status"] != "aligned" {
		t.Fatalf("status = %d, body = %v", code, got)
	}
	equivalents, _ := got["equivalents"].([]any)
	if len(equivalents) != 1 {
		t.Fatalf("equivalents = %v", got["equivalents"])
	}
	if e := equivalents[0].(map[string]any); e["zone_code"] != "Z12" || e["submitted"] != "admin" || e["canonical"] != "administration" {
		t.Errorf("equivalent = %v", e)
	}
}

// A registry still naming Z8/Z10/Z11 by their former responsibilities is a genuine mismatch.
func TestCheckAlignmentReportsFormerZoneResponsibilities(t *testing.T) {
	body := `{"zones":{"Z0":"negative","Z1":"public","Z2":"identity","Z3":"ops","Z4":"protected",
		"Z5":"system","Z6":"eventlogs","Z7":"comms","Z8":"resourcesync","Z9":"analytics",
		"Z10":"behavior","Z11":"aimesh","Z12":"admin"}}`
	code, got := postAlignment(t, body)

	if code != http.StatusOK || got["aligned"] != false || got["status"] != "mismatch" {
		t.Fatalf("status = %d, body = %v", code, got)
	}
	differences, _ := got["differences"].([]any)
	codes := []string{}
	for _, d := range differences {
		codes = append(codes, d.(map[string]any)["zone_code"].(string))
	}
	if len(codes) != 3 || codes[0] != "Z8" || codes[1] != "Z10" || codes[2] != "Z11" {
		t.Errorf("differences = %v, want Z8, Z10, Z11", codes)
	}
}

// Execution envelopes from callers still sending a former key resolve to the same zone.
func TestResolveExecutionZoneAcceptsAliases(t *testing.T) {
	handler := &PipelineHandler{policyRegistry: policy.NewRegistry()}
	twelve := 12

	for _, envelope := range []PipelineZoneEnvelope{
		{ZoneKey: "admin"},
		{ZoneKey: "administration"},
		{ZoneID: &twelve, ZoneKey: "admin"},
		{ZoneID: &twelve, ZoneKey: "Z12"},
	} {
		zone, err := handler.resolveExecutionZone(&PipelineExecutionRequest{Zone: envelope})
		if err != nil || zone.ID != 12 {
			t.Errorf("%+v resolved to Z%d (%v), want Z12", envelope, zone.ID, err)
		}
	}

	// A key naming another zone still conflicts with the declared zone ID.
	if _, err := handler.resolveExecutionZone(&PipelineExecutionRequest{Zone: PipelineZoneEnvelope{ZoneID: &twelve, ZoneKey: "ops"}}); err == nil {
		t.Error("zone_key ops must not match zone_id 12")
	}
	if _, err := handler.resolveExecutionZone(&PipelineExecutionRequest{Zone: PipelineZoneEnvelope{ZoneKey: "resourcesync"}}); err == nil {
		t.Error("former Z8 key resourcesync must not resolve")
	}
}

// The registry Web3Codex submits today (Z0 "lobby") aligns through the Boundary alias.
func TestCheckAlignmentAcceptsWeb3CodexCurrentPayload(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zones", "web3codex_registry.json"))
	if err != nil {
		t.Fatalf("read Web3Codex registry: %v", err)
	}
	code, got := postAlignment(t, string(body))

	if code != http.StatusOK || got["aligned"] != true || got["zones_checked"] != float64(13) {
		t.Fatalf("status = %d, body = %v", code, got)
	}
	equivalents, _ := got["equivalents"].([]any)
	if len(equivalents) != 1 || equivalents[0].(map[string]any)["submitted"] != "lobby" {
		t.Errorf("equivalents = %v, want Z0 lobby -> boundary", got["equivalents"])
	}
	if got["signature"] == got["expected_signature"] {
		t.Error("signatures must still describe each registry as named")
	}
}

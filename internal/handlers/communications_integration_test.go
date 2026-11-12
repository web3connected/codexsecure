//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

func TestCommunicationHTTPPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("COMMUNICATION_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNICATION_TEST_DSN not set")
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid disposable DSN")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" || cfg.ConnConfig.Database != "codexsecure_configuration_test" {
		t.Fatal("requires dedicated local disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	// The unit helper's verified principal grants test-site. This fixture supplies
	// trusted authentication context; it does not emulate gateway key verification.
	// A random schema-independent site key avoids deleting immutable audit history.
	site := "http-" + uuid.NewString()
	if _, e = admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'HTTP test','https://example.test')`, site); e != nil {
		t.Fatal(e)
	}
	runtimeCfg := cfg.Copy()
	runtimeCfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, e := c.Exec(ctx, "SET ROLE codex_zones_runtime")
		return e
	}
	runtime, e := pgxpool.NewWithConfig(ctx, runtimeCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer runtime.Close()
	store := communications.NewPostgresStore(runtime, map[string]bool{"Z1": true})
	router := integrationCommunicationRouter(t, store, site)
	key := uuid.NewString()
	base := "/api/v1/sites/" + site + "/communications/" + key
	call := func(method, path, body, replay string, want int) *httptest.ResponseRecorder {
		t.Helper()
		if replay == "" {
			replay = uuid.NewString()
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Service-Key", "test-service-key")
		req.Header.Set("X-CodexSecure-Actor", "admin-1")
		req.Header.Set("Idempotency-Key", replay)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d body %s", method, path, w.Code, want, w.Body.String())
		}
		return w
	}
	discovery := `{"registration_key":"` + key + `","scope":"universal","component":"main-navigation","page":{"key":"home","url":"https://example.test/","zone":"Z1"},"detected":{"destination":"https://example.test/products","method":"GET","transport_method":"GET","type":"fetch","zone":"Z1"},"declaration":{"scope":"universal","component":"main-navigation","item":"fetch","item_identifier":"products-link"},"browser_object_identifier":"products-link","expected_structure_revision":0}`
	call("POST", "/api/v1/sites/"+site+"/communications/discovery", discovery, "", 200)
	call("GET", base+"/configuration", "", "", 200)
	document := `{"schema_version":1,"destination":"/products","destination_match":"exact","methods":["GET","POST"],"type":"fetch","zone":"Z1","enabled":true}`
	call("PUT", base+"/configuration/draft", `{"document":`+document+`}`, "", 422)
	call("PUT", base+"/configuration/draft", `{"document":`+document+`,"expected_version":null}`, "", 422)
	call("PUT", base+"/configuration/draft", `{"document":`+document+`,"expected_version":0}`, "", 200)
	call("POST", base+"/configuration/approve", `{"draft_version":1}`, "", 422)
	call("POST", base+"/configuration/approve", `{"draft_version":1,"structure_revision":0}`, "", 409)
	approvalKey := uuid.NewString()
	call("POST", base+"/configuration/approve", `{"draft_version":1,"structure_revision":1,"reason":null}`, approvalKey, 200)
	replay := call("POST", base+"/configuration/approve", `{"draft_version":1,"structure_revision":1,"reason":null}`, approvalKey, 200)
	if !strings.Contains(replay.Body.String(), `"replayed":true`) {
		t.Fatal(replay.Body.String())
	}
	read := call("GET", base+"/configuration", "", "", 200)
	var envelope struct{ Data communications.Configuration }
	if e = json.Unmarshal(read.Body.Bytes(), &envelope); e != nil || envelope.Data.Approved == nil || envelope.Data.Effective != "approved" || len(envelope.Data.Approved.Document.Methods) != 2 {
		t.Fatal(envelope, e)
	}
	call("GET", base+"/configuration/history", "", "", 200)
	call("GET", "/api/v1/sites/"+site+"/communications/assessments?scope=site", "", "", 200)
	changed := strings.Replace(discovery, `"expected_structure_revision":0`, `"expected_structure_revision":1`, 1)
	changed = strings.Replace(changed, `"method":"GET"`, `"method":"POST"`, 1)
	call("POST", "/api/v1/sites/"+site+"/communications/discovery", changed, "", 200)
	call("GET", strings.Replace(base, key, uuid.NewString(), 1)+"/configuration", "", "", 404)
	for _, tc := range []struct {
		header, value string
		want          int
	}{{"X-Service-Key", "", 401}, {"X-Service-Key", "bad", 401}, {"X-CodexSecure-Actor", "", 403}, {"X-CodexSecure-Actor", "other-admin", 403}, {"Idempotency-Key", "", 422}} {
		req := httptest.NewRequest("POST", base+"/disable", strings.NewReader(`{}`))
		req.Header.Set("X-Service-Key", "test-service-key")
		req.Header.Set("X-CodexSecure-Actor", "admin-1")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		req.Header.Set(tc.header, tc.value)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatal(tc.header, w.Code, w.Body.String())
		}
	}
	call("PUT", base+"/configuration/draft", `{"document":`+document+`,"expected_version":0}`, "", 200)
	call("POST", base+"/configuration/reject", `{"draft_version":2,"reason":"review"}`, "", 200)
	call("POST", base+"/disable", `{}`, "", 200)
	call("POST", base+"/revoke", `{}`, "", 200)

	// Batch lifecycle uses the real SECURITY DEFINER functions through HTTP.
	batches := "/api/v1/sites/" + site + "/communications/discovery/batches"
	siteBatch := uuid.NewString()
	call("POST", batches, `{"scope":"site","expected_pages":["home"]}`, siteBatch, 200)
	pageBatch := uuid.NewString()
	startBody := `{"scope":"page","page":{"key":"home","url":"https://example.test/","zone":"Z1"},"parent_batch_key":"` + siteBatch + `"}`
	call("POST", batches, startBody, pageBatch, 200)
	replay = call("POST", batches, startBody, pageBatch, 200)
	if !strings.Contains(replay.Body.String(), `"replayed":true`) {
		t.Fatal(replay.Body.String())
	}
	call("POST", batches+"/"+pageBatch+"/failures", `{"sequence":1,"outcome":"rejected","code":"CONFIGURATION_INVALID","registration_key":"`+key+`"}`, "", 200)
	// A corrected observation replaces the rejected sequence without changing policy.
	var batched communications.Discovery
	if e = json.Unmarshal([]byte(changed), &batched); e != nil {
		t.Fatal(e)
	}
	batched.Batch = &communications.DiscoveryBatch{Key: pageBatch, Sequence: 1}
	payload, _ := json.Marshal(batched)
	call("POST", "/api/v1/sites/"+site+"/communications/discovery", string(payload), "", 200)
	call("POST", batches+"/"+pageBatch+"/finalize", `{"outcome":"complete","expected_observations":1}`, "", 200)
	call("POST", batches+"/"+siteBatch+"/finalize", `{"outcome":"complete","expected_observations":null}`, "", 200)
	assessment := call("GET", "/api/v1/sites/"+site+"/communications/assessments?scope=site", "", "", 200)
	var covered struct{ Data communications.Assessment }
	if e = json.Unmarshal(assessment.Body.Bytes(), &covered); e != nil || !covered.Data.Complete || covered.Data.Coverage == nil {
		t.Fatal(assessment.Body.String(), e)
	}
	call("POST", batches+"/"+uuid.NewString()+"/finalize", `{"outcome":"complete","expected_observations":0}`, "", 404)
	call("POST", batches, `{"scope":"page","page":{"key":"outside","url":"https://other.test/","zone":"Z1"}}`, "", 422)
	call("POST", batches, `{"scope":"page","unknown":true}`, "", 400)

	// A real repository failure remains non-2xx and carries incomplete scope data.
	runtime.Close()
	failed := call("GET", "/api/v1/sites/"+site+"/communications/assessments?scope=site", "", "", 500)
	var partial struct {
		Data communications.Assessment
		Code string
	}
	if e = json.Unmarshal(failed.Body.Bytes(), &partial); e != nil || partial.Data.Complete || partial.Data.Site != site || partial.Data.Scope.Type != "site" || !strings.Contains(failed.Body.String(), "assessment_processing_failed") {
		t.Fatal(partial, e)
	}
}

func integrationCommunicationRouter(t *testing.T, store communications.Store, site string) *gin.Engine {
	t.Helper()
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		middleware.SetCommunicationPrincipal(c, middleware.CommunicationPrincipal{GatewayAuthenticated: true, Actor: "admin-1", AdminSites: map[string]bool{site: true}, DiscoverySites: map[string]bool{site: true}})
	})
	NewCommunicationHandler(store).RegisterRoutes(r.Group("/api/v1"))
	return r
}

//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/communications"
)

// Real HTTP and PostgreSQL; integrationCommunicationRouter installs a trusted
// test principal. This does not claim the deployed gateway authenticates SDKs.
func TestStage2HTTPRetrieval(t *testing.T) {
	dsn := os.Getenv("COMMUNICATION_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNICATION_TEST_DSN not set")
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid DSN")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" || cfg.ConnConfig.Database != "codexsecure_configuration_test" {
		t.Fatal("requires disposable local database")
	}
	ctx := context.Background()
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	site := "stage2-http-" + uuid.NewString()
	_, e = pool.Exec(ctx, `INSERT INTO sites(site_key,name,base_url) VALUES($1,'Stage 2 HTTP','https://example.test')`, site)
	if e != nil {
		t.Fatal(e)
	}
	_, e = pool.Exec(ctx, `INSERT INTO pages(site_id,page_key,url,zone) SELECT id,'empty','https://example.test/pages/empty','Z1' FROM sites WHERE site_key=$1`, site)
	if e != nil {
		t.Fatal(e)
	}
	store := communications.NewPostgresStore(pool, map[string]bool{"Z1": true})
	server := httptest.NewServer(integrationCommunicationRouter(t, store, site))
	defer server.Close()
	path := server.URL + "/api/v1/sites/" + site + "/communications/assessments?scope=page&page="
	call := func(page, key string, want int) map[string]any {
		t.Helper()
		req, _ := http.NewRequest("GET", path+page, nil)
		if key != "" {
			req.Header.Set("X-Service-Key", key)
		}
		resp, e := server.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		var body map[string]any
		if e = json.NewDecoder(resp.Body).Decode(&body); e != nil {
			t.Fatal(e)
		}
		if resp.StatusCode != want {
			t.Fatalf("got %d want %d: %v", resp.StatusCode, want, body)
		}
		return body
	}
	url := "https%3A%2F%2Fexample.test%2Fpages%2Fempty"
	body := call(url, "test-service-key", 200)
	data := body["data"].(map[string]any)
	if data["page"].(map[string]any)["zone"] != "Z1" || len(data["communications"].([]any)) != 0 {
		t.Fatal(data)
	}
	call("https%3A%2F%2Fexample.test%2Funknown", "test-service-key", 404)
	call(url, "", 401)
	d := communications.Discovery{RegistrationKey: uuid.NewString(), Scope: "page", Page: communications.SourcePage{Key: "empty", URL: "https://example.test/pages/empty", Zone: "Z1"}, Detected: communications.Detected{Destination: "https://example.test/a", Method: "GET", TransportMethod: "GET", Type: "link", Zone: "Z1"}, BrowserIdentifier: "stage2-link"}
	scope := "page"
	item := "link"
	id := "stage2-link"
	revision := int64(0)
	d.Declaration = communications.Declaration{Scope: &scope, Item: &item, ItemIdentifier: &id}
	d.ExpectedStructureRevision = &revision
	if _, e = store.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	body = call(url, "test-service-key", 200)
	if len(body["data"].(map[string]any)["communications"].([]any)) != 1 {
		t.Fatal(body)
	}
	pool.Close()
	call(url, "test-service-key", 500)
}

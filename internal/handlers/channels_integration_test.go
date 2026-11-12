//go:build integration

package handlers

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/channels"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

// Gin → ChannelAuth → CommunicationChannelManager → codex_zones_runtime functions.
// The principal fixture supplies trusted gateway context; it does not emulate
// gateway key verification.
func TestChannelHTTPPostgresIntegration(t *testing.T) {
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
	siteA, siteB, unknown := "http-ch-"+uuid.NewString(), "http-ch-"+uuid.NewString(), "http-ch-"+uuid.NewString()
	for _, site := range []string{siteA, siteB} {
		if _, e = admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'HTTP channel test','https://example.test')`, site); e != nil {
			t.Fatal(e)
		}
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

	// Controlled stand-ins for codexhash_api and the legacy codextime_api, reached
	// through the real clients. They do not prove either production service.
	hash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Data string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		sum := sha512.Sum512([]byte(req.Data))
		_, _ = w.Write([]byte(`{"payload_hash":"` + hex.EncodeToString(sum[:]) + `","algo":"sha512","hash_size":512}`))
	}))
	defer hash.Close()
	tiu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tiu":214612574.69576,"timestamp":"2026-10-28T12:00:00Z","epoch":"2020-01-01 00:00:00","cached":false}`))
	}))
	defer tiu.Close()
	t.Setenv("CODEXHASH_API_URL", hash.URL)
	t.Setenv("CODEXTIME_API_URL", tiu.URL)
	manager := channels.NewManager(runtime, codexhash.NewClient(), codextime.NewClient())

	t.Setenv("SERVICE_API_KEY", "test-service-key")
	gin.SetMode(gin.TestMode)
	router := func(grants ...string) *gin.Engine {
		sites := map[string]bool{}
		for _, site := range grants {
			sites[site] = true
		}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			middleware.SetCommunicationPrincipal(c, middleware.CommunicationPrincipal{GatewayAuthenticated: true, Actor: "app", ChannelSites: sites})
		})
		NewChannelHandler(manager).RegisterRoutes(r.Group("/api/v1"))
		return r
	}
	type envelope struct {
		Data  *channels.Result `json:"data"`
		Code  string           `json:"code"`
		Error string           `json:"error"`
	}
	call := func(r *gin.Engine, method, path, body string, want int) envelope {
		t.Helper()
		w := channelRequest(r, method, path, body)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d body %s", method, path, w.Code, want, w.Body.String())
		}
		var out envelope
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	a := router(siteA, unknown)
	baseA := "/api/v1/sites/" + siteA + "/channels"

	opened := call(a, "POST", baseA, `{"zone":"Z1"}`, 201)
	if opened.Data.Outcome != "opened" || opened.Data.Channel.Status != "pending" || opened.Data.Channel.Protection.Status != "not_protected" {
		t.Fatal(opened)
	}
	id := opened.Data.Channel.ID.String()
	got := call(a, "GET", baseA+"/"+id, "", 200)
	if got.Data.Outcome != "found" || !got.Data.Channel.LastActivityAt.Equal(opened.Data.Channel.LastActivityAt) {
		t.Fatal("GET must not renew", got)
	}
	closed := call(a, "POST", baseA+"/"+id+"/close", "", 200)
	if closed.Data.Outcome != "closed" || closed.Data.Channel.ClosedAt == nil {
		t.Fatal(closed)
	}
	again := call(a, "POST", baseA+"/"+id+"/close", "{}", 409)
	if again.Code != "CHANNEL_CLOSED" || again.Data == nil || again.Data.Channel.Status != "closed" {
		t.Fatal(again)
	}
	if got = call(a, "GET", baseA+"/"+id, "", 200); got.Data.Channel.Status != "closed" {
		t.Fatal(got)
	}

	// Expired: lapsed fixture inserted by the schema owner.
	tx, e := admin.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var lapsed string
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE codex_zones_admin"); e == nil {
		e = tx.QueryRow(ctx, `INSERT INTO public.communication_channels(site_id,initial_zone,opened_at,last_activity_at,expires_at)
 SELECT id,'Z1',now()-interval '31 minutes',now()-interval '31 minutes',now()-interval '1 minute' FROM public.sites WHERE site_key=$1 RETURNING channel_key::text`, siteA).Scan(&lapsed)
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	expired := call(a, "POST", baseA+"/"+lapsed+"/close", "", 409)
	if expired.Code != "CHANNEL_EXPIRED" || expired.Data.Channel.Status != "expired" {
		t.Fatal(expired)
	}
	if got = call(a, "GET", baseA+"/"+lapsed, "", 200); got.Data.Channel.Status != "expired" {
		t.Fatal(got)
	}

	// Errors.
	if out := call(a, "POST", baseA, `{"zone":"Z0"}`, 422); out.Code != "CHANNEL_REQUEST_INVALID" {
		t.Fatal(out)
	}
	if out := call(a, "POST", "/api/v1/sites/"+unknown+"/channels", `{"zone":"Z1"}`, 404); out.Code != "SITE_NOT_FOUND" {
		t.Fatal(out)
	}
	if out := call(a, "GET", baseA+"/"+uuid.NewString(), "", 404); out.Code != "CHANNEL_NOT_FOUND" || out.Data != nil {
		t.Fatal(out)
	}
	if out := call(a, "GET", baseA+"/not-a-uuid", "", 422); out.Code != "CHANNEL_IDENTIFIER_INVALID" {
		t.Fatal(out)
	}

	// Site isolation: no grant for B → 403; with a grant for B, A's channel is not found.
	open2 := call(a, "POST", baseA, `{"zone":"Z2"}`, 201).Data.Channel.ID.String()
	if out := call(a, "GET", "/api/v1/sites/"+siteB+"/channels/"+open2, "", 403); out.Code != "ADMIN_UNAUTHORIZED" {
		t.Fatal(out)
	}
	b := router(siteB)
	call(b, "GET", "/api/v1/sites/"+siteB+"/channels/"+open2, "", 404)
	call(b, "POST", "/api/v1/sites/"+siteB+"/channels/"+open2+"/close", "", 404)
	call(b, "GET", baseA+"/"+open2, "", 403)
	if got = call(a, "GET", baseA+"/"+open2, "", 200); got.Data.Channel.Status != "pending" {
		t.Fatal("foreign close changed the channel", got)
	}

	// Activation: only through an authorized site, never with client protection.
	if out := call(a, "POST", "/api/v1/sites/"+siteB+"/channels/"+open2+"/activate", "", 403); out.Code != "ADMIN_UNAUTHORIZED" {
		t.Fatal(out)
	}
	call(b, "POST", "/api/v1/sites/"+siteB+"/channels/"+open2+"/activate", "", 404)
	call(b, "POST", "/api/v1/sites/"+siteB+"/channels/"+open2+"/renew", "", 404)
	if out := call(a, "POST", baseA+"/"+open2+"/activate", `{"tiu":"1.000000"}`, 400); out.Code != "CHANNEL_REQUEST_INVALID" {
		t.Fatal(out)
	}
	if got = call(a, "GET", baseA+"/"+open2, "", 200); got.Data.Channel.Status != "pending" {
		t.Fatal("refused activation changed the channel", got)
	}
	activated := call(a, "POST", baseA+"/"+open2+"/activate", "", 200)
	p := activated.Data.Channel.Protection
	if activated.Data.Outcome != "activated" || activated.Data.Channel.Status != "active" || p.Status != "protected" || *p.TIU != "214612574.695760" ||
		*p.TIUScheme != "codextime-legacy-seconds-2020/us" || !activated.Data.Channel.ExpiresAt.After(got.Data.Channel.ExpiresAt) {
		t.Fatal(activated)
	}
	// The TIU stays a JSON string on the wire.
	w := channelRequest(a, "GET", baseA+"/"+open2, "")
	if !strings.Contains(w.Body.String(), `"tiu":"214612574.695760"`) {
		t.Fatal("TIU not a string", w.Body.String())
	}
	repeated := call(a, "POST", baseA+"/"+open2+"/activate", "{}", 200)
	if repeated.Data.Outcome != "already_active" || *repeated.Data.Channel.Protection.CodexHash != *p.CodexHash {
		t.Fatal(repeated)
	}
	if out := call(a, "POST", baseA+"/"+id+"/activate", "", 409); out.Code != "CHANNEL_CLOSED" || out.Data.Channel.Protection.Status != "not_protected" {
		t.Fatal(out)
	}
	if out := call(a, "POST", baseA+"/"+lapsed+"/activate", "", 409); out.Code != "CHANNEL_EXPIRED" || out.Data.Channel.Status != "expired" {
		t.Fatal(out)
	}
	call(a, "POST", baseA+"/"+uuid.NewString()+"/activate", "", 404)
	call(a, "POST", baseA+"/not-a-uuid/activate", "", 422)
	if out := call(a, "POST", baseA+"/"+open2+"/close", "", 200); out.Data.Channel.Protection.Status != "protected" {
		t.Fatal("close lost protection", out)
	}

	// No expiration endpoint (renewal has its own route since C10).
	for _, suffix := range []string{"/expire"} {
		w := channelRequest(a, "POST", baseA+"/"+open2+suffix, "")
		if w.Code != 404 || strings.Contains(w.Body.String(), "renewed") {
			t.Fatal(suffix, w.Code, w.Body.String())
		}
	}
}

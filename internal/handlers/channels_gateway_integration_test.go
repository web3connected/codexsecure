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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/channels"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

// Phase C6: Gin → GatewayAssertion → ChannelAuth → CommunicationChannelManager →
// codex_zones_runtime functions, authenticated only by signed C4 assertions. No
// principal is injected.
func TestChannelGatewayAssertionPostgresIntegration(t *testing.T) {
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
	siteA, siteB := "gw-ch-"+uuid.NewString(), "gw-ch-"+uuid.NewString()
	for _, site := range []string{siteA, siteB} {
		if _, e = admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'Gateway channel test','https://example.test')`, site); e != nil {
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

	// Controlled stand-ins for codexhash_api and the legacy codextime_api.
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

	signer, keys := newGatewaySigner(t, "gateway-2026-10")
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	t.Setenv("CODEXSECURE_GATEWAY_ASSERTION_KEYS", keys)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewChannelHandler(manager).RegisterRoutes(r.Group("/api/v1"))

	type envelope struct {
		Data *channels.Result `json:"data"`
		Code string           `json:"code"`
	}
	// call signs for assertedSite and the exact request path, as the gateway does.
	call := func(assertedSite, method, path, body string, want int) envelope {
		t.Helper()
		w := gatewayRequest(r, method, path, body, "test-service-key", signer.sign(t, "web3connected-demo", assertedSite, method, path, time.Now()))
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d body %s", method, path, w.Code, want, w.Body.String())
		}
		var out envelope
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	baseA := "/api/v1/sites/" + siteA + "/channels"

	opened := call(siteA, "POST", baseA, `{"zone":"Z1"}`, 201)
	if opened.Data.Outcome != "opened" || opened.Data.Channel.Status != "pending" || opened.Data.Channel.Site != siteA {
		t.Fatal(opened)
	}
	id := opened.Data.Channel.ID.String()
	pending := call(siteA, "POST", baseA, `{"zone":"Z2"}`, 201).Data.Channel.ID.String()
	if got := call(siteA, "GET", baseA+"/"+id, "", 200); got.Data.Channel.Status != "pending" || got.Data.Channel.Protection.Status != "not_protected" {
		t.Fatal(got)
	}

	// Site isolation: siteB's assertion, on siteB's path, finds nothing; siteA's
	// assertion on siteB's path is not a gateway identity.
	call(siteB, "GET", "/api/v1/sites/"+siteB+"/channels/"+id, "", 404)
	call(siteB, "POST", "/api/v1/sites/"+siteB+"/channels/"+id+"/activate", "", 404)
	call(siteB, "POST", "/api/v1/sites/"+siteB+"/channels/"+id+"/close", "", 404)
	if out := call(siteA, "GET", "/api/v1/sites/"+siteB+"/channels/"+id, "", 403); out.Code != "ADMIN_UNAUTHORIZED" {
		t.Fatal(out)
	}

	activated := call(siteA, "POST", baseA+"/"+id+"/activate", "", 200)
	if activated.Data.Outcome != "activated" || activated.Data.Channel.Status != "active" || activated.Data.Channel.Protection.Status != "protected" {
		t.Fatal(activated)
	}
	if got := call(siteA, "GET", baseA+"/"+id, "", 200); got.Data.Channel.Status != "active" {
		t.Fatal(got)
	}

	// Renewal (C10): the deadline moves to 30 minutes after this communication;
	// identity, zone and protection stay as activation wrote them. Get never renews.
	if out := call(siteA, "POST", baseA+"/"+pending+"/renew", "", 409); out.Code != "CHANNEL_NOT_ACTIVE" || out.Data.Channel.Status != "pending" {
		t.Fatal(out)
	}
	call(siteB, "POST", "/api/v1/sites/"+siteB+"/channels/"+id+"/renew", "", 404)
	call(siteA, "POST", baseA+"/"+uuid.NewString()+"/renew", "", 404)
	time.Sleep(10 * time.Millisecond)
	renewed := call(siteA, "POST", baseA+"/"+id+"/renew", "", 200)
	was, now := activated.Data.Channel, renewed.Data.Channel
	if renewed.Data.Outcome != "renewed" || now.ID != was.ID || now.Status != "active" || now.Zone != was.Zone || !now.OpenedAt.Equal(was.OpenedAt) ||
		!now.LastActivityAt.After(was.LastActivityAt) || now.ExpiresAt.Sub(now.LastActivityAt) != 30*time.Minute ||
		*now.Protection.CodexHash != *was.Protection.CodexHash || *now.Protection.TIU != *was.Protection.TIU || !now.Protection.ProtectedAt.Equal(*was.Protection.ProtectedAt) {
		t.Fatal("renewal must only move the deadline", was, now)
	}
	if got := call(siteA, "GET", baseA+"/"+id, "", 200); !got.Data.Channel.ExpiresAt.Equal(now.ExpiresAt) {
		t.Fatal("Get must not renew", got)
	}
	// Identity binding (C11-H): guest → the account's UUID once, through a signed
	// assertion for exactly this method and path.
	user, other := uuid.NewString(), uuid.NewString()
	bindPath := baseA + "/" + id + "/bind-identity"
	bindBody := func(u string) string { return `{"identity":{"type":"authenticated","user_uuid":"` + u + `"}}` }
	if out := call(siteA, "POST", baseA+"/"+pending+"/bind-identity", bindBody(user), 409); out.Code != "CHANNEL_NOT_ACTIVE" || out.Data.Channel.Identity.State != "guest" {
		t.Fatal(out)
	}
	call(siteB, "POST", "/api/v1/sites/"+siteB+"/channels/"+id+"/bind-identity", bindBody(user), 404)
	// An assertion for another path, method or site, a replayed one or an expired one is no gateway identity.
	signedActivate := signer.sign(t, "web3connected-demo", siteA, "POST", baseA+"/"+id+"/activate", time.Now())
	for name, assertion := range map[string]string{
		"path":    signedActivate,
		"method":  signer.sign(t, "web3connected-demo", siteA, "GET", bindPath, time.Now()),
		"site":    signer.sign(t, "web3connected-demo", siteB, "POST", bindPath, time.Now()),
		"expired": signer.sign(t, "web3connected-demo", siteA, "POST", bindPath, time.Now().Add(-time.Hour)),
	} {
		if w := gatewayRequest(r, "POST", bindPath, bindBody(user), "test-service-key", assertion); w.Code != 403 {
			t.Fatal(name, w.Code, w.Body.String())
		}
	}
	if w := gatewayRequest(r, "POST", bindPath, bindBody(user), "wrong-key", signer.sign(t, "web3connected-demo", siteA, "POST", bindPath, time.Now())); w.Code != 401 {
		t.Fatal("service key", w.Code)
	}
	guest := call(siteA, "GET", baseA+"/"+id, "", 200).Data.Channel
	if guest.Identity.State != "guest" || guest.SiteUUID == uuid.Nil {
		t.Fatal("rejected assertions changed the identity", guest)
	}
	assertion := signer.sign(t, "web3connected-demo", siteA, "POST", bindPath, time.Now())
	w := gatewayRequest(r, "POST", bindPath, bindBody(user), "test-service-key", assertion)
	var bound envelope
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &bound) != nil || bound.Data.Outcome != "bound" {
		t.Fatal(w.Code, w.Body.String())
	}
	b := bound.Data.Channel
	if b.ID != guest.ID || b.SiteUUID != guest.SiteUUID || b.Identity.UserUUID.String() != user || b.Identity.State != "authenticated" || !b.Identity.Final ||
		*b.Protection.CodexHash == *guest.Protection.CodexHash || *b.Protection.TIU != *guest.Protection.TIU || !b.ExpiresAt.Equal(guest.ExpiresAt) {
		t.Fatal("binding", guest, b)
	}
	if replay := gatewayRequest(r, "POST", bindPath, bindBody(user), "test-service-key", assertion); replay.Code != 403 {
		t.Fatal("replayed assertion accepted", replay.Code)
	}
	if again := call(siteA, "POST", bindPath, bindBody(user), 200); again.Data.Outcome != "already_bound" || *again.Data.Channel.Protection.CodexHash != *b.Protection.CodexHash {
		t.Fatal(again)
	}
	if out := call(siteA, "POST", bindPath, bindBody(other), 409); out.Code != "CHANNEL_IDENTITY_CONFLICT" || out.Data.Channel.Identity.UserUUID.String() != user || out.Data.Channel.Status != "active" {
		t.Fatal(out)
	}
	call(siteA, "POST", bindPath, `{"user_uuid":"`+user+`"}`, 400)
	call(siteA, "POST", bindPath, bindBody("someone@example.com"), 422)

	if closed := call(siteA, "POST", baseA+"/"+id+"/close", "", 200); closed.Data.Outcome != "closed" || closed.Data.Channel.ClosedAt == nil ||
		closed.Data.Channel.Identity.UserUUID.String() != user || *closed.Data.Channel.Protection.CodexHash != *b.Protection.CodexHash {
		t.Fatal(closed)
	}
	if got := call(siteA, "GET", baseA+"/"+id, "", 200); got.Data.Channel.Status != "closed" {
		t.Fatal(got)
	}
	if again := call(siteA, "POST", baseA+"/"+id+"/activate", "", 409); again.Code != "CHANNEL_CLOSED" {
		t.Fatal(again)
	}
	if again := call(siteA, "POST", baseA+"/"+id+"/renew", "", 409); again.Code != "CHANNEL_CLOSED" || again.Data == nil || !again.Data.Channel.ExpiresAt.Equal(now.ExpiresAt) {
		t.Fatal(again)
	}
	if again := call(siteA, "POST", bindPath, bindBody(other), 409); again.Code != "CHANNEL_CLOSED" {
		t.Fatal(again)
	}
}

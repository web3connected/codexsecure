//go:build integration

package channels

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

// serviceFixture is a controlled stand-in for codexhash_api POST /api/hash/payload
// and the legacy codextime_api GET /api/time/current-tiu, reached through the real
// Go clients over HTTP. By default it answers as those services do: SHA-512 of the
// UTF-8 data, and an uncached TIU written as a JSON number. It proves the Go and
// database contracts only, not CodexTime's mathematics or either service's
// production availability.
type serviceFixture struct {
	mu         sync.Mutex
	hashData   []string
	tiuCalls   int
	hashReply  func(data string) (int, string)
	tiuReply   func(call int) (int, string)
	beforeHash func()
	hasher     *codexhash.Client
	clock      *codextime.Client
}

func genuineHash(data string) (int, string) {
	sum := sha512.Sum512([]byte(data))
	return 200, `{"payload_hash":"` + hex.EncodeToString(sum[:]) + `","algo":"sha512","hash_size":512}`
}
func legacyReply(wire string) (int, string) {
	return 200, `{"tiu":` + wire + `,"timestamp":"2026-10-28T12:00:00.000001Z","epoch":"2020-01-01 00:00:00","cached":false}`
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	f := &serviceFixture{
		hashReply: genuineHash,
		// Distinct per call; the .5 is written without its trailing zeros, as Python does.
		tiuReply: func(call int) (int, string) { return legacyReply(strconv.Itoa(214612574+call) + ".5") },
	}
	hash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Data     string `json:"data"`
			Algo     string `json:"algo"`
			HashSize int    `json:"hash_size"`
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/hash/payload" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Algo != "sha512" || req.HashSize != 512 {
			http.Error(w, `{"detail":"unexpected request"}`, 422)
			return
		}
		f.mu.Lock()
		f.hashData = append(f.hashData, req.Data)
		reply, before := f.hashReply, f.beforeHash
		f.mu.Unlock()
		if before != nil {
			before()
		}
		status, body := reply(req.Data)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(hash.Close)
	tiu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/time/current-tiu" {
			http.NotFound(w, r)
			return
		}
		f.mu.Lock()
		f.tiuCalls++
		call, reply := f.tiuCalls, f.tiuReply
		f.mu.Unlock()
		status, body := reply(call)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(tiu.Close)
	// Fail-open is on, as in docker-compose: activation must not depend on it being off.
	t.Setenv("CODEXHASH_API_URL", hash.URL)
	t.Setenv("CODEXHASH_FAIL_OPEN", "true")
	t.Setenv("CODEXTIME_API_URL", tiu.URL)
	t.Setenv("CODEXTIME_FAIL_OPEN", "true")
	f.hasher, f.clock = codexhash.NewClient(), codextime.NewClient()
	return f
}
func (f *serviceFixture) calls() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.hashData), f.tiuCalls
}
func (f *serviceFixture) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.hashData...)
}
func (f *serviceFixture) set(update func(*serviceFixture)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	update(f)
}

func activationDatabase(t *testing.T) (context.Context, *pgxpool.Pool, *Manager, *serviceFixture) {
	ctx, admin, m := channelDatabase(t)
	f := newServiceFixture(t) // replaces the fixture channelDatabase started, to observe it
	return ctx, admin, NewManager(m.pool, f.hasher, f.clock), f
}

// expectPending asserts that a failed activation wrote nothing.
func expectPending(t *testing.T, ctx context.Context, m *Manager, site string, id uuid.UUID, before *Channel) {
	t.Helper()
	got, e := m.Get(ctx, site, id)
	expectOutcome(t, got, e, Found, StatusPending)
	c := got.Channel
	if c.Protection.Status != "not_protected" || !c.LastActivityAt.Equal(before.LastActivityAt) || !c.ExpiresAt.Equal(before.ExpiresAt) {
		t.Fatalf("failed activation changed the channel: %+v", c)
	}
}

func TestChannelActivationPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	opened, e := m.Open(ctx, site, "Z3")
	expectOutcome(t, opened, e, Opened, StatusPending)
	id := opened.Channel.ID

	activated, e := m.Activate(ctx, site, id)
	expectOutcome(t, activated, e, Activated, StatusActive)
	c, p := activated.Channel, activated.Channel.Protection
	text, _ := binding(opened.Channel.SiteUUID, opened.Channel.ID, opened.Channel.Identity.UserUUID)
	sum := sha512.Sum512([]byte(text))
	if p.Status != "protected" || *p.Binding != BindingRepresentation || *p.CodexHash != hex.EncodeToString(sum[:]) ||
		*p.TIU != "214612575.500000" || *p.TIUScheme != codextime.LegacySecondsScheme || *p.TIUUnit != "s" {
		t.Fatalf("protection %+v", p)
	}
	if !p.ProtectedAt.Equal(c.LastActivityAt) || c.ExpiresAt.Sub(c.LastActivityAt) != 30*time.Minute ||
		c.LastActivityAt.Before(opened.Channel.LastActivityAt) || !c.OpenedAt.Equal(opened.Channel.OpenedAt) || c.Zone != "Z3" || c.ClosedAt != nil {
		t.Fatalf("activated channel %+v", c)
	}
	// Activation keeps the database-generated guest identity and the Site UUID.
	if c.SiteUUID == uuid.Nil || c.SiteUUID != opened.Channel.SiteUUID || c.Identity.State != IdentityGuest || c.Identity.Final ||
		c.Identity.UserUUID != opened.Channel.Identity.UserUUID || c.Identity.BoundAt != nil {
		t.Fatalf("activation changed the identity %+v", c.Identity)
	}
	// CodexHash received exactly the database's own binding text.
	var database string
	if e = admin.QueryRow(ctx, `SELECT public.codex_channels_binding_v2(s.site_uuid, c.channel_key, c.user_uuid) FROM public.communication_channels c
		JOIN public.sites s ON s.id = c.site_id WHERE s.site_key=$1 AND c.channel_key=$2`, site, id).Scan(&database); e != nil {
		t.Fatal(e)
	}
	if hashes, tius := f.calls(); hashes != 1 || tius != 1 || f.sent()[0] != text || text != database {
		t.Fatalf("services: %d/%d, sent %q, database %q", hashes, tius, f.sent(), database)
	}
	var stored string
	if e = admin.QueryRow(ctx, `SELECT protection_tiu::text FROM public.communication_channels WHERE channel_key=$1`, id).Scan(&stored); e != nil || stored != *p.TIU {
		t.Fatal("stored TIU", stored, e)
	}

	// Persistence, and repetition: the fast path asks no service and changes nothing.
	got, e := m.Get(ctx, site, id)
	expectOutcome(t, got, e, Found, StatusActive)
	again, e := m.Activate(ctx, site, id)
	expectOutcome(t, again, e, AlreadyActive, StatusActive)
	if hashes, tius := f.calls(); hashes != 1 || tius != 1 {
		t.Fatal("repeated activation asked a service", hashes, tius)
	}
	for _, r := range []Result{got, again} {
		if *r.Channel.Protection.TIU != *p.TIU || *r.Channel.Protection.CodexHash != *p.CodexHash || !r.Channel.ExpiresAt.Equal(c.ExpiresAt) {
			t.Fatal("protection or deadline changed", r.Channel)
		}
	}
	// The database path refuses other protection values for an active channel.
	other := codextime.ExactTIU{Value: "999.000000", Scheme: codextime.LegacySecondsScheme, Unit: "s"}
	direct, e := m.activate(ctx, site, id, *p.CodexHash, other)
	expectOutcome(t, direct, e, AlreadyActive, StatusActive)
	if *direct.Channel.Protection.TIU != *p.TIU {
		t.Fatal("protection replaced", direct.Channel.Protection)
	}

	// Activated channels take part in the existing lifecycle.
	renewed, e := m.Renew(ctx, site, id)
	expectOutcome(t, renewed, e, Renewed, StatusActive)
	if *renewed.Channel.Protection.TIU != *p.TIU || !renewed.Channel.Protection.ProtectedAt.Equal(*p.ProtectedAt) {
		t.Fatal("renewal changed protection")
	}
	closed, e := m.Close(ctx, site, id)
	expectOutcome(t, closed, e, Closed, StatusClosed)
	if closed.Channel.Protection.Status != "protected" || *closed.Channel.Protection.CodexHash != *p.CodexHash {
		t.Fatal("closing lost protection", closed.Channel.Protection)
	}
	final, e := m.Activate(ctx, site, id)
	expectOutcome(t, final, e, Closed, StatusClosed)
	if hashes, tius := f.calls(); hashes != 1 || tius != 1 {
		t.Fatal("activation of a closed channel asked a service", hashes, tius)
	}

	// Protection is immutable even for the schema owner, and the runtime has no table access.
	for name, q := range map[string]string{
		"tiu":       `UPDATE public.communication_channels SET protection_tiu = 2 WHERE channel_key=$1`,
		"codexhash": `UPDATE public.communication_channels SET protection_codexhash = repeat('a',128) WHERE channel_key=$1`,
		"scheme":    `UPDATE public.communication_channels SET protection_tiu_scheme = 'codextime-tiu-2020/1' WHERE channel_key=$1`,
		"cleared":   `UPDATE public.communication_channels SET protection_binding=NULL, protection_codexhash=NULL, protection_tiu=NULL, protection_tiu_scheme=NULL, protection_tiu_unit=NULL, protected_at=NULL WHERE channel_key=$1`,
	} {
		tx, e := admin.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = tx.Exec(ctx, "SET LOCAL ROLE codex_zones_admin")
		_, e = tx.Exec(ctx, q, id)
		_ = tx.Rollback(ctx)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || (pg.Code != "CX409" && pg.Code != "23514") {
			t.Fatal(name, "owner change not refused:", e)
		}
	}
	for _, q := range []string{`UPDATE public.communication_channels SET protection_tiu = 2`, `SELECT public.codex_channels_binding(gen_random_uuid(),'x','Z1',now())`,
		`SELECT public.codex_channels_binding_v2(gen_random_uuid(),gen_random_uuid(),gen_random_uuid())`,
		`SELECT public.codex_channels_identity_hash(gen_random_uuid(),gen_random_uuid(),gen_random_uuid())`} {
		var pg *pgconn.PgError
		if _, e := m.pool.Exec(ctx, q); !errors.As(e, &pg) || pg.Code != "42501" {
			t.Fatal("runtime not refused:", q, e)
		}
	}
}

// Activation is the first qualifying activity: it restarts the 30-minute deadline.
func TestChannelActivationDeadlinePostgres(t *testing.T) {
	ctx, admin, m, _ := activationDatabase(t)
	site := newSite(t, ctx, admin)
	id := fixtureChannel(t, ctx, admin, site, StatusPending, 20*time.Minute)
	before, e := m.Get(ctx, site, id)
	expectOutcome(t, before, e, Found, StatusPending)
	activated, e := m.Activate(ctx, site, id)
	expectOutcome(t, activated, e, Activated, StatusActive)
	c := activated.Channel
	if c.LastActivityAt.Sub(before.Channel.LastActivityAt) < 19*time.Minute || c.ExpiresAt.Sub(c.LastActivityAt) != 30*time.Minute ||
		!c.ExpiresAt.After(before.Channel.ExpiresAt.Add(19*time.Minute)) || !c.OpenedAt.Equal(before.Channel.OpenedAt) {
		t.Fatalf("deadline not restarted: before %+v after %+v", before.Channel, c)
	}
}

// A lapsed channel is expired, never activated or revived, including when it
// lapses while CodexHash is answering. A channel closed meanwhile stays closed.
func TestChannelActivationLapsedAndClosedPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)

	lapsed := fixtureChannel(t, ctx, admin, site, StatusPending, 31*time.Minute)
	r, e := m.Activate(ctx, site, lapsed)
	expectOutcome(t, r, e, Expired, StatusExpired)
	if r.Channel.Protection.Status != "not_protected" {
		t.Fatal("expired channel protected")
	}
	if hashes, tius := f.calls(); hashes != 0 || tius != 0 {
		t.Fatal("lapsed channel asked a service")
	}

	lapsing := fixtureChannel(t, ctx, admin, site, StatusPending, 30*time.Minute-1500*time.Millisecond)
	f.set(func(f *serviceFixture) { f.beforeHash = func() { time.Sleep(2500 * time.Millisecond) } })
	r, e = m.Activate(ctx, site, lapsing)
	expectOutcome(t, r, e, Expired, StatusExpired)
	if r.Channel.Protection.Status != "not_protected" {
		t.Fatal("lapsed channel protected")
	}

	opened, e := m.Open(ctx, site, "Z1")
	expectOutcome(t, opened, e, Opened, StatusPending)
	var closed Result
	var closeErr error
	f.set(func(f *serviceFixture) {
		f.beforeHash = func() { closed, closeErr = m.Close(ctx, site, opened.Channel.ID) }
	})
	r, e = m.Activate(ctx, site, opened.Channel.ID)
	expectOutcome(t, closed, closeErr, Closed, StatusClosed)
	expectOutcome(t, r, e, Closed, StatusClosed)
	if r.Channel.Protection.Status != "not_protected" {
		t.Fatal("closed channel protected")
	}
}

func TestChannelActivationServiceFailuresPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	sha256Reply := func(data string) (int, string) {
		sum := sha256.Sum256([]byte(data))
		return 200, `{"payload_hash":"` + hex.EncodeToString(sum[:]) + `","algo":"sha256","hash_size":256}`
	}
	withHash := func(reply func(string) (int, string)) func(*serviceFixture) {
		return func(f *serviceFixture) { f.hashReply = reply }
	}
	withTIU := func(status int, body string) func(*serviceFixture) {
		return func(f *serviceFixture) { f.tiuReply = func(int) (int, string) { return status, body } }
	}
	const fresh = `,"timestamp":"2026-10-28T12:00:00Z","epoch":"2020-01-01 00:00:00","cached":false}`
	for _, tc := range []struct {
		name    string
		fixture func(*serviceFixture)
		env     map[string]string
		code    string
	}{
		{"codexhash 500", withHash(func(string) (int, string) { return 500, `{"detail":"db down"}` }), nil, "CODEXHASH_UNAVAILABLE"},
		{"codexhash 422", withHash(func(string) (int, string) { return 422, `{"detail":"hash_size"}` }), nil, "CODEXHASH_UNAVAILABLE"},
		{"codexhash not json", withHash(func(string) (int, string) { return 200, `<html>` }), nil, "CODEXHASH_UNAVAILABLE"},
		{"codexhash unreachable", nil, map[string]string{"CODEXHASH_API_URL": "http://127.0.0.1:1"}, "CODEXHASH_UNAVAILABLE"},
		{"codexhash disabled", nil, map[string]string{"CODEXHASH_ENABLED": "false"}, "CODEXHASH_UNAVAILABLE"},
		{"codexhash sha256 fallback shape", withHash(sha256Reply), nil, "CODEXHASH_INVALID_RESPONSE"},
		{"codexhash empty object", withHash(func(string) (int, string) { return 200, `{}` }), nil, "CODEXHASH_INVALID_RESPONSE"},
		{"codexhash missing size", withHash(func(d string) (int, string) {
			_, body := genuineHash(d)
			return 200, body[:len(body)-len(`,"hash_size":512}`)] + `}`
		}), nil, "CODEXHASH_INVALID_RESPONSE"},
		{"codexhash uppercase", withHash(func(d string) (int, string) {
			sum := sha512.Sum512([]byte(d))
			return 200, fmt.Sprintf(`{"payload_hash":"%X","algo":"sha512","hash_size":512}`, sum[:])
		}), nil, "CODEXHASH_INVALID_RESPONSE"},
		// Well formed but not this channel's binding: only the database can tell.
		{"codexhash wrong digest", withHash(func(string) (int, string) { return genuineHash("another binding") }), nil, "INTERNAL_ERROR"},
		{"codextime 503", withTIU(503, `{"detail":"unavailable"}`), nil, "CODEXTIME_UNAVAILABLE"},
		{"codextime unreachable", nil, map[string]string{"CODEXTIME_API_URL": "http://127.0.0.1:1"}, "CODEXTIME_UNAVAILABLE"},
		{"codextime disabled", nil, map[string]string{"CODEXTIME_ENABLED": "false"}, "CODEXTIME_UNAVAILABLE"},
		{"codextime cached", withTIU(200, `{"tiu":214612574.695765,"timestamp":"2026-10-28T12:00:00Z","epoch":"2020-01-01 00:00:00","cached":true}`), nil, "CODEXTIME_UNAVAILABLE"},
		{"codextime fail-open shape", withTIU(200, `{"tiu":0,"source":"unavailable"}`), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime zero", withTIU(200, `{"tiu":0`+fresh), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime string", withTIU(200, `{"tiu":"214612574.695765"`+fresh), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime seven decimals", withTIU(200, `{"tiu":214612574.6957651`+fresh), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime exponent", withTIU(200, `{"tiu":2.14612574e8`+fresh), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime missing tiu", withTIU(200, `{"timestamp":"2026-10-28T12:00:00Z","epoch":"2020-01-01 00:00:00","cached":false}`), nil, "CODEXTIME_INVALID_RESPONSE"},
		{"codextime other epoch", withTIU(200, `{"tiu":1.5,"timestamp":"2026-10-28T12:00:00Z","epoch":"1970-01-01 00:00:00","cached":false}`), nil, "CODEXTIME_INVALID_RESPONSE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hashReply func(string) (int, string)
			var tiuReply func(int) (int, string)
			f.set(func(f *serviceFixture) { hashReply, tiuReply = f.hashReply, f.tiuReply })
			defer f.set(func(f *serviceFixture) { f.hashReply, f.tiuReply = hashReply, tiuReply })
			if tc.fixture != nil {
				f.set(tc.fixture)
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			manager := NewManager(m.pool, codexhash.NewClient(), codextime.NewClient())
			opened, e := manager.Open(ctx, site, "Z1")
			expectOutcome(t, opened, e, Opened, StatusPending)
			r, e := manager.Activate(ctx, site, opened.Channel.ID)
			if e == nil || r.Channel != nil {
				t.Fatal("activation did not fail", r)
			}
			expectCode(t, e, tc.code)
			expectPending(t, ctx, manager, site, opened.Channel.ID, opened.Channel)
		})
	}
	t.Run("not configured", func(t *testing.T) {
		for _, manager := range []*Manager{NewManager(m.pool, nil, f.clock), NewManager(m.pool, f.hasher, nil)} {
			opened, e := manager.Open(ctx, site, "Z1")
			expectOutcome(t, opened, e, Opened, StatusPending)
			if _, e = manager.Activate(ctx, site, opened.Channel.ID); e == nil {
				t.Fatal("activation without a service")
			}
			expectPending(t, ctx, manager, site, opened.Channel.ID, opened.Channel)
		}
	})
}

// The TIU reaches PostgreSQL and the API exactly as CodexTime wrote it.
func TestChannelActivationExactTIUPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	check := func(manager *Manager, want codextime.ExactTIU) {
		t.Helper()
		opened, e := manager.Open(ctx, site, "Z5")
		expectOutcome(t, opened, e, Opened, StatusPending)
		r, e := manager.Activate(ctx, site, opened.Channel.ID)
		expectOutcome(t, r, e, Activated, StatusActive)
		p := r.Channel.Protection
		var stored, scheme, unit string
		if e = admin.QueryRow(ctx, `SELECT protection_tiu::text, protection_tiu_scheme, protection_tiu_unit FROM public.communication_channels WHERE channel_key=$1`, opened.Channel.ID).Scan(&stored, &scheme, &unit); e != nil {
			t.Fatal(e)
		}
		if *p.TIU != want.Value || stored != want.Value || *p.TIUScheme != want.Scheme || scheme != want.Scheme || *p.TIUUnit != want.Unit || unit != want.Unit {
			t.Fatalf("got %s/%s (%s %s), want %+v", *p.TIU, stored, scheme, unit, want)
		}
	}
	for wire, want := range map[string]string{
		"214612574.695765":      "214612574.695765",
		"214612574.5":           "214612574.500000",
		"214612574.0":           "214612574.000000",
		"0.000001":              "0.000001",
		"99999999999999.999999": "99999999999999.999999",
		"12345678901234.123457": "12345678901234.123457",
	} {
		f.set(func(f *serviceFixture) { f.tiuReply = func(int) (int, string) { return legacyReply(wire) } })
		check(m, codextime.ExactTIU{Value: want, Scheme: codextime.LegacySecondsScheme, Unit: "s"})
		// The point of the exact path: a float64 would not have kept these.
		if float, _ := strconv.ParseFloat(want, 64); len(want) > 17 && strconv.FormatFloat(float, 'f', 6, 64) == want {
			t.Fatal("fixture does not exercise float64 loss", want)
		}
	}
	// CodexTime V1 integers, which no current client issues, through a TIU source stub.
	for _, v1 := range []codextime.ExactTIU{
		{Value: "214612574695765961", Scheme: "codextime-tiu-2020/1", Unit: "ns"},
		{Value: "3980700000000000000000000000000000000000000000439638", Scheme: "codextime-tiu-2020/1", Unit: "qm"},
		{Value: "1" + fmt.Sprintf("%079d", 7), Scheme: "codextime-tiu-2020/1", Unit: "as"},
	} {
		check(NewManager(m.pool, f.hasher, clockStub{tiu: v1}), v1)
	}
}

// Concurrent activations produce exactly one protection; every caller sees it.
func TestChannelActivationConcurrencyPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	opened, e := m.Open(ctx, site, "Z2")
	expectOutcome(t, opened, e, Opened, StatusPending)
	const n = 16
	// Hold every CodexHash answer until all callers have passed the fast path, so
	// that they all reach codex_channels_activate.
	var arrived sync.WaitGroup
	arrived.Add(n)
	f.set(func(f *serviceFixture) { f.beforeHash = func() { arrived.Done(); arrived.Wait() } })
	results := make([]Result, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = m.Activate(ctx, site, opened.Channel.ID)
		}(i)
	}
	wg.Wait()
	final, e := m.Get(ctx, site, opened.Channel.ID)
	expectOutcome(t, final, e, Found, StatusActive)
	counts := map[string]int{}
	for i, r := range results {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		counts[r.Outcome]++
		p, want := r.Channel.Protection, final.Channel.Protection
		if *p.TIU != *want.TIU || *p.CodexHash != *want.CodexHash || !p.ProtectedAt.Equal(*want.ProtectedAt) {
			t.Fatal("conflicting protection", p, want)
		}
	}
	if hashes, tius := f.calls(); counts[Activated] != 1 || counts[AlreadyActive] != n-1 || hashes != n || tius != n {
		t.Fatal("concurrent activation", counts, hashes, tius)
	}
}

func TestChannelActivationSiteIsolationPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	siteA, siteB := newSite(t, ctx, admin), newSite(t, ctx, admin)
	a, e := m.Open(ctx, siteA, "Z1")
	expectOutcome(t, a, e, Opened, StatusPending)
	b, e := m.Open(ctx, siteB, "Z1")
	expectOutcome(t, b, e, Opened, StatusPending)
	for _, site := range []string{siteB, "missing-" + uuid.NewString()} {
		r, e := m.Activate(ctx, site, a.Channel.ID)
		expectOutcome(t, r, e, NotFound, "")
	}
	if _, e = m.Activate(ctx, siteA, uuid.Nil); !errors.Is(e, ErrIdentifierInvalid) {
		t.Fatal("nil identity", e)
	}
	if hashes, tius := f.calls(); hashes != 0 || tius != 0 {
		t.Fatal("foreign activation asked a service")
	}
	// Directly at the database: A's genuine hash does not open A through site B,
	// and B's genuine hash does not protect A.
	bindA, _ := binding(a.Channel.SiteUUID, a.Channel.ID, a.Channel.Identity.UserUUID)
	bindB, _ := binding(b.Channel.SiteUUID, b.Channel.ID, b.Channel.Identity.UserUUID)
	sumA, sumB := sha512.Sum512([]byte(bindA)), sha512.Sum512([]byte(bindB))
	tiu := codextime.ExactTIU{Value: "214612574.500000", Scheme: codextime.LegacySecondsScheme, Unit: "s"}
	r, e := m.activate(ctx, siteB, a.Channel.ID, hex.EncodeToString(sumA[:]), tiu)
	expectOutcome(t, r, e, NotFound, "")
	r, e = m.activate(ctx, siteA, a.Channel.ID, hex.EncodeToString(sumB[:]), tiu)
	expectOutcome(t, r, e, ProtectionMismatch, StatusPending)
	expectPending(t, ctx, m, siteA, a.Channel.ID, a.Channel)
	expectPending(t, ctx, m, siteB, b.Channel.ID, b.Channel)
	// A Go-built input the database refuses is internal, not a client error.
	if _, e = m.activate(ctx, siteA, a.Channel.ID, "short", tiu); !errors.Is(e, errContract) {
		t.Fatal("CX422 not internal", e)
	}
	expectCode(t, e, "INTERNAL_ERROR")
}

// Go and PostgreSQL build the same binding for a site key with every escaping rule
// (which the v2 binding no longer contains) and for a session that is not in UTC,
// so genuine activation succeeds end to end.
func TestChannelActivationBindingCompatibilityPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	cfg := m.pool.Config()
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, e := c.Exec(ctx, "SET ROLE codex_zones_runtime; SET TimeZone = 'America/New_York'")
		return e
	}
	eastern, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(eastern.Close)
	manager := NewManager(eastern, f.hasher, f.clock)
	site := "s\"\\/<&>é  \b\f\n\r\t\x01\x1f\x7f\U0001F600-" + uuid.NewString()
	if _, e = admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'Escaping test','https://example.test')`, site); e != nil {
		t.Fatal(e)
	}
	opened, e := manager.Open(ctx, site, "Z12")
	expectOutcome(t, opened, e, Opened, StatusPending)
	r, e := manager.Activate(ctx, site, opened.Channel.ID)
	expectOutcome(t, r, e, Activated, StatusActive)
	var database string
	if e = admin.QueryRow(ctx, `SELECT public.codex_channels_binding_v2(s.site_uuid, c.channel_key, c.user_uuid) FROM public.communication_channels c
		JOIN public.sites s ON s.id = c.site_id WHERE s.site_key=$1 AND c.channel_key=$2`, site, opened.Channel.ID).Scan(&database); e != nil {
		t.Fatal(e)
	}
	if sent := f.sent(); sent[len(sent)-1] != database {
		t.Fatalf("Go binding %q\ndatabase   %q", sent[len(sent)-1], database)
	}
}

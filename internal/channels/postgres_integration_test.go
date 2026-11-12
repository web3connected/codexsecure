//go:build integration

package channels

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/communications"
)

// channelDatabase connects to the dedicated disposable database only. admin is
// the disposable cluster administrator (fixtures); runtime connects exactly as the
// Go API does, as codex_zones_runtime, which may only execute lifecycle functions.
func channelDatabase(t *testing.T) (context.Context, *pgxpool.Pool, *Manager) {
	dsn := os.Getenv("COMMUNICATION_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNICATION_TEST_DSN not set")
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid disposable DSN")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" || cfg.ConnConfig.Database != "codexsecure_configuration_test" {
		t.Fatal("requires the dedicated loopback disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(admin.Close)
	runtimeCfg := cfg.Copy()
	runtimeCfg.MaxConns = 32
	runtimeCfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, e := c.Exec(ctx, "SET ROLE codex_zones_runtime")
		return e
	}
	runtime, e := pgxpool.NewWithConfig(ctx, runtimeCfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(runtime.Close)
	services := newServiceFixture(t)
	return ctx, admin, NewManager(runtime, services.hasher, services.clock)
}
func newSite(t *testing.T, ctx context.Context, admin *pgxpool.Pool) string {
	t.Helper()
	site := "channel-" + uuid.NewString()
	if _, e := admin.Exec(ctx, `INSERT INTO public.sites(site_key,name,base_url) VALUES($1,'Channel test','https://example.test')`, site); e != nil {
		t.Fatal(e)
	}
	return site
}

// fixtureChannel inserts a channel as the schema owner with times in the past, the
// only way to observe a lapsed deadline without waiting 30 minutes. Active
// fixtures carry a genuine SHA-512 of their v2 guest binding, computed by the
// database, and an integration-test TIU (1 ns) that no CodexTime service issued.
func fixtureChannel(t *testing.T, ctx context.Context, admin *pgxpool.Pool, site, status string, age time.Duration) uuid.UUID {
	t.Helper()
	return fixtureChannelBinding(t, ctx, admin, site, status, age, BindingRepresentation)
}

// fixtureLegacyChannel is an active channel protected under binding v1 before C11-H.
func fixtureLegacyChannel(t *testing.T, ctx context.Context, admin *pgxpool.Pool, site string, age time.Duration) uuid.UUID {
	t.Helper()
	return fixtureChannelBinding(t, ctx, admin, site, StatusActive, age, LegacyBindingRepresentation)
}

func fixtureChannelBinding(t *testing.T, ctx context.Context, admin *pgxpool.Pool, site, status string, age time.Duration, representation string) uuid.UUID {
	t.Helper()
	tx, e := admin.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE codex_zones_admin"); e != nil {
		t.Fatal(e)
	}
	var id uuid.UUID
	e = tx.QueryRow(ctx, `WITH f AS (SELECT s.id, s.site_key, s.site_uuid, gen_random_uuid() AS channel_key, gen_random_uuid() AS user_uuid,
  now()-$3::float8*interval '1 second' AS at, $2::text='active' AS active, $4::text='codexsecure-channel-binding-v1' AS legacy
  FROM public.sites s WHERE s.site_key=$1)
 INSERT INTO public.communication_channels(channel_key,site_id,status,initial_zone,opened_at,last_activity_at,expires_at,user_uuid,
  protection_binding,protection_codexhash,protection_tiu,protection_tiu_scheme,protection_tiu_unit,protected_at)
 SELECT f.channel_key,f.id,$2::text,'Z1',f.at,f.at,f.at+public.codex_channels_inactivity_timeout(),f.user_uuid,
  CASE WHEN f.active THEN $4::text END,
  CASE WHEN f.active AND f.legacy THEN encode(sha512(convert_to(public.codex_channels_binding(f.channel_key,f.site_key,'Z1',f.at),'UTF8')),'hex')
       WHEN f.active THEN public.codex_channels_identity_hash(f.site_uuid,f.channel_key,f.user_uuid) END,
  CASE WHEN f.active THEN 1 END, CASE WHEN f.active THEN 'codextime-tiu-2020/1' END, CASE WHEN f.active THEN 'ns' END, CASE WHEN f.active THEN f.at END
 FROM f RETURNING channel_key`, site, status, age.Seconds(), representation).Scan(&id)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	return id
}
func expectOutcome(t *testing.T, r Result, e error, outcome, status string) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
	if r.Outcome != outcome || (status != "" && (r.Channel == nil || r.Channel.Status != status)) {
		t.Fatalf("got %s/%v want %s/%s", r.Outcome, r.Channel, outcome, status)
	}
}
func expectCode(t *testing.T, e error, code string) {
	t.Helper()
	if api := MapError(e); api.Code != code {
		t.Fatalf("got %v (%s) want %s", e, api.Code, code)
	}
}

func TestChannelLifecyclePostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	site := newSite(t, ctx, admin)

	opened, e := m.Open(ctx, site, "Z4")
	expectOutcome(t, opened, e, Opened, StatusPending)
	c := opened.Channel
	if c.ID == uuid.Nil || c.Site != site || c.Zone != "Z4" || c.ClosedAt != nil || c.Protection.Status != "not_protected" ||
		!c.LastActivityAt.Equal(c.OpenedAt) || c.ExpiresAt.Sub(c.LastActivityAt) != 30*time.Minute {
		t.Fatalf("opened channel %+v", c)
	}
	second, e := m.Open(ctx, site, "Z4")
	if e != nil || second.Channel.ID == c.ID {
		t.Fatal("open must generate a new identity", e)
	}

	got, e := m.Get(ctx, site, c.ID)
	expectOutcome(t, got, e, Found, StatusPending)
	if !got.Channel.LastActivityAt.Equal(c.LastActivityAt) || !got.Channel.ExpiresAt.Equal(c.ExpiresAt) {
		t.Fatal("Get renewed the channel")
	}
	// Pending channels are never renewed, and the attempt is a failure.
	if _, e = m.Renew(ctx, site, c.ID); e == nil {
		t.Fatal("pending channel renewed")
	}
	expectCode(t, e, "CHANNEL_NOT_ACTIVE")

	notDue, e := m.Expire(ctx, site, c.ID)
	expectOutcome(t, notDue, e, NotDue, StatusPending)

	closed, e := m.Close(ctx, site, c.ID)
	expectOutcome(t, closed, e, Closed, StatusClosed)
	if closed.Channel.ClosedAt == nil || !closed.Channel.LastActivityAt.Equal(c.LastActivityAt) || !closed.Channel.ExpiresAt.Equal(c.ExpiresAt) {
		t.Fatal("close must set closed_at and is not activity", closed.Channel)
	}
	again, e := m.Close(ctx, site, c.ID)
	expectOutcome(t, again, e, AlreadyClosed, StatusClosed)
	got, e = m.Get(ctx, site, c.ID)
	expectOutcome(t, got, e, Found, StatusClosed)
	final, e := m.Expire(ctx, site, c.ID)
	expectOutcome(t, final, e, AlreadyClosed, StatusClosed)
	_, e = m.Renew(ctx, site, c.ID)
	expectCode(t, e, "CHANNEL_CLOSED")
}

func TestChannelExpirationPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	site := newSite(t, ctx, admin)

	// Lazy expiry: the next use of a lapsed channel records and reports expiry.
	lapsed := fixtureChannel(t, ctx, admin, site, StatusPending, 31*time.Minute)
	got, e := m.Get(ctx, site, lapsed)
	expectOutcome(t, got, e, Found, StatusExpired)
	if got.Channel.ClosedAt != nil || !time.Now().After(got.Channel.ExpiresAt) {
		t.Fatal("expired channel keeps its final deadline and no closed_at", got.Channel)
	}
	closed, e := m.Close(ctx, site, lapsed)
	expectOutcome(t, closed, e, Expired, StatusExpired)
	again, e := m.Expire(ctx, site, lapsed)
	expectOutcome(t, again, e, AlreadyExpired, StatusExpired)

	// Lapsed but not yet used: closing it is refused as expired, never revived.
	unused := fixtureChannel(t, ctx, admin, site, StatusPending, 45*time.Minute)
	closed, e = m.Close(ctx, site, unused)
	expectOutcome(t, closed, e, Expired, StatusExpired)

	// Explicit expiry by the manager.
	due := fixtureChannel(t, ctx, admin, site, StatusPending, 30*time.Minute+time.Second)
	expired, e := m.Expire(ctx, site, due)
	expectOutcome(t, expired, e, Expired, StatusExpired)
	expired, e = m.Expire(ctx, site, due)
	expectOutcome(t, expired, e, AlreadyExpired, StatusExpired)

	// A pending channel inside its window is not due, even at 29 minutes.
	live := fixtureChannel(t, ctx, admin, site, StatusPending, 29*time.Minute)
	notDue, e := m.Expire(ctx, site, live)
	expectOutcome(t, notDue, e, NotDue, StatusPending)

	// A lapsed active channel is expired, not renewed.
	active := fixtureChannel(t, ctx, admin, site, StatusActive, 31*time.Minute)
	_, e = m.Renew(ctx, site, active)
	expectCode(t, e, "CHANNEL_EXPIRED")
	got, e = m.Get(ctx, site, active)
	expectOutcome(t, got, e, Found, StatusExpired)
}

// Renewal compatibility with codex_channels_renew. The active channel is a
// schema-owner fixture so that it can be backdated; activation is tested separately.
func TestChannelRenewalPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	site := newSite(t, ctx, admin)
	active := fixtureChannel(t, ctx, admin, site, StatusActive, 10*time.Minute)
	before, e := m.Get(ctx, site, active)
	expectOutcome(t, before, e, Found, StatusActive)
	renewed, e := m.Renew(ctx, site, active)
	expectOutcome(t, renewed, e, Renewed, StatusActive)
	r := renewed.Channel
	if !r.LastActivityAt.After(before.Channel.LastActivityAt) || r.ExpiresAt.Sub(r.LastActivityAt) != 30*time.Minute || !r.OpenedAt.Equal(before.Channel.OpenedAt) {
		t.Fatal("renewal must move activity forward to now + 30 minutes", before.Channel, r)
	}
	if r.Protection.Status != "protected" || r.Protection.TIU == nil {
		t.Fatal("protection lost", r.Protection)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := m.Renew(ctx, site, active); e != nil {
				errs <- e
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal("concurrent renewal failed", e)
	}
	after, e := m.Get(ctx, site, active)
	expectOutcome(t, after, e, Found, StatusActive)
	if after.Channel.LastActivityAt.Before(r.LastActivityAt) {
		t.Fatal("activity moved backwards")
	}
}

func TestChannelSiteIsolationPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	siteA, siteB := newSite(t, ctx, admin), newSite(t, ctx, admin)
	opened, e := m.Open(ctx, siteA, "Z1")
	expectOutcome(t, opened, e, Opened, StatusPending)
	id := opened.Channel.ID
	for name, call := range map[string]func() (Result, error){
		"get":    func() (Result, error) { return m.Get(ctx, siteB, id) },
		"close":  func() (Result, error) { return m.Close(ctx, siteB, id) },
		"expire": func() (Result, error) { return m.Expire(ctx, siteB, id) },
		"unknown site": func() (Result, error) {
			return m.Get(ctx, "missing-"+uuid.NewString(), id)
		},
	} {
		r, e := call()
		if e != nil || r.Outcome != NotFound || r.Channel != nil {
			t.Fatal(name, r, e)
		}
	}
	_, e = m.Renew(ctx, siteB, id)
	expectCode(t, e, "CHANNEL_NOT_FOUND")
	got, e := m.Get(ctx, siteA, id)
	expectOutcome(t, got, e, Found, StatusPending)
	r, e := m.Get(ctx, siteA, uuid.New())
	expectOutcome(t, r, e, NotFound, "")
}

func TestChannelInvalidInputsPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	site := newSite(t, ctx, admin)
	for _, zone := range []string{"", "Z0", "Z13", "z1", "Z01", "Z1 ", "Boundary"} {
		_, e := m.Open(ctx, site, zone)
		expectCode(t, e, "CHANNEL_REQUEST_INVALID")
	}
	_, e := m.Open(ctx, "missing-"+uuid.NewString(), "Z1")
	expectCode(t, e, "SITE_NOT_FOUND")
	for name, call := range map[string]func() (Result, error){
		"get":    func() (Result, error) { return m.Get(ctx, site, uuid.Nil) },
		"close":  func() (Result, error) { return m.Close(ctx, site, uuid.Nil) },
		"expire": func() (Result, error) { return m.Expire(ctx, site, uuid.Nil) },
		"renew":  func() (Result, error) { return m.Renew(ctx, site, uuid.Nil) },
	} {
		if _, e := call(); !errors.Is(e, ErrIdentifierInvalid) {
			t.Fatal(name, e)
		}
	}
}

func TestChannelConcurrencyPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	site := newSite(t, ctx, admin)
	race := func(n int, call func(i int) (Result, error)) map[string]int {
		t.Helper()
		var wg sync.WaitGroup
		var mu sync.Mutex
		counts := map[string]int{}
		var failure error
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				r, e := call(i)
				mu.Lock()
				defer mu.Unlock()
				if e != nil {
					failure = e
					return
				}
				counts[r.Outcome]++
			}(i)
		}
		wg.Wait()
		if failure != nil {
			t.Fatal(failure)
		}
		return counts
	}

	ids := sync.Map{}
	opens := race(20, func(int) (Result, error) {
		r, e := m.Open(ctx, site, "Z2")
		if e == nil {
			if _, dup := ids.LoadOrStore(r.Channel.ID, true); dup {
				t.Error("duplicate channel identity")
			}
		}
		return r, e
	})
	if opens[Opened] != 20 {
		t.Fatal(opens)
	}

	opened, _ := m.Open(ctx, site, "Z2")
	closes := race(12, func(int) (Result, error) { return m.Close(ctx, site, opened.Channel.ID) })
	if closes[Closed] != 1 || closes[AlreadyClosed] != 11 {
		t.Fatal("concurrent close", closes)
	}

	due := fixtureChannel(t, ctx, admin, site, StatusPending, 31*time.Minute)
	expiries := race(12, func(int) (Result, error) { return m.Expire(ctx, site, due) })
	if expiries[Expired] != 1 || expiries[AlreadyExpired] != 11 {
		t.Fatal("concurrent expire", expiries)
	}

	// Close and Expire racing on a lapsed channel: it ends exactly once, as expired.
	mixed := fixtureChannel(t, ctx, admin, site, StatusPending, 31*time.Minute)
	results := race(12, func(i int) (Result, error) {
		if i%2 == 0 {
			return m.Close(ctx, site, mixed)
		}
		return m.Expire(ctx, site, mixed)
	})
	if results[Closed] != 0 || results[AlreadyClosed] != 0 || results[Expired]+results[AlreadyExpired] != 12 {
		t.Fatal("close/expire race", results)
	}
	final, e := m.Get(ctx, site, mixed)
	expectOutcome(t, final, e, Found, StatusExpired)

	// Get, Close and Expire racing on a live channel: exactly one close, no expiry.
	live, _ := m.Open(ctx, site, "Z2")
	results = race(12, func(i int) (Result, error) {
		switch i % 3 {
		case 0:
			return m.Get(ctx, site, live.Channel.ID)
		case 1:
			return m.Close(ctx, site, live.Channel.ID)
		}
		return m.Expire(ctx, site, live.Channel.ID)
	})
	if results[Closed] != 1 || results[Expired] != 0 || results[AlreadyExpired] != 0 {
		t.Fatal("live race", results)
	}
}

// The manager depends on exactly this database contract; drift must fail here.
func TestChannelDatabaseCompatibilityPostgres(t *testing.T) {
	ctx, admin, m := channelDatabase(t)
	var timeout string
	if e := admin.QueryRow(ctx, `SELECT public.codex_channels_inactivity_timeout()::text`).Scan(&timeout); e != nil || timeout != "00:30:00" {
		t.Fatal("inactivity timeout", timeout, e)
	}
	rows, e := admin.Query(ctx, `SELECT p.proname, pg_get_function_identity_arguments(p.oid), p.prorettype::regtype::text, p.prosecdef,
 has_function_privilege('codex_zones_runtime', p.oid, 'EXECUTE')
 FROM pg_proc p WHERE p.pronamespace='public'::regnamespace AND p.proname LIKE 'codex\_channels\_%' ORDER BY 1`)
	if e != nil {
		t.Fatal(e)
	}
	type fn struct {
		args, ret string
		definer   bool
		runtime   bool
	}
	found := map[string]fn{}
	for rows.Next() {
		var name string
		var f fn
		if e = rows.Scan(&name, &f.args, &f.ret, &f.definer, &f.runtime); e != nil {
			t.Fatal(e)
		}
		found[name] = f
	}
	rows.Close()
	want := map[string]fn{
		"codex_channels_open":     {"p_site text, p_zone text", "jsonb", true, true},
		"codex_channels_get":      {"p_site text, p_channel uuid", "jsonb", true, true},
		"codex_channels_renew":    {"p_site text, p_channel uuid", "jsonb", true, true},
		"codex_channels_close":    {"p_site text, p_channel uuid", "jsonb", true, true},
		"codex_channels_expire":   {"p_site text, p_channel uuid", "jsonb", true, true},
		"codex_channels_activate": {"p_site text, p_channel uuid, p_codexhash text, p_tiu text, p_tiu_scheme text, p_tiu_unit text", "jsonb", true, true},
		// C11-H.
		"codex_channels_bind_identity": {"p_site text, p_channel uuid, p_user_uuid uuid, p_codexhash text", "jsonb", true, true},
	}
	runtimeFunctions := 0
	for _, f := range found {
		if f.runtime {
			runtimeFunctions++
		}
	}
	if runtimeFunctions != len(want) {
		t.Fatalf("runtime may execute %d channel functions, want %d", runtimeFunctions, len(want))
	}
	for name, f := range want {
		if found[name] != f {
			t.Fatalf("%s: got %+v want %+v", name, found[name], f)
		}
	}
	for _, internal := range []string{"codex_channels_locate", "codex_channels_view", "codex_channels_guard", "codex_channels_inactivity_timeout",
		"codex_channels_binding", "codex_channels_tiu_valid", "codex_channels_site_key_guard", "codex_channels_binding_v2", "codex_channels_identity_hash"} {
		if found[internal].runtime {
			t.Fatal("runtime may execute internal function", internal)
		}
	}

	// The runtime role cannot bypass the functions, choose a lifetime or an identity.
	site := newSite(t, ctx, admin)
	opened, e := m.Open(ctx, site, "Z1")
	expectOutcome(t, opened, e, Opened, StatusPending)
	for name, q := range map[string]string{
		"select":       `SELECT count(*) FROM public.communication_channels`,
		"insert":       `INSERT INTO public.communication_channels(site_id,initial_zone,expires_at) VALUES(1,'Z1',now())`,
		"update":       `UPDATE public.communication_channels SET status='active'`,
		"delete":       `DELETE FROM public.communication_channels`,
		"locate":       `SELECT public.codex_channels_locate('x', gen_random_uuid())`,
		"lifetime arg": `SELECT public.codex_channels_open('x','Z1',60)`,
		"identity arg": `SELECT public.codex_channels_open('x','Z1',gen_random_uuid())`,
		"sequence":     `SELECT nextval('public.communication_channels_id_seq')`,
	} {
		_, e := m.pool.Exec(ctx, q)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || (pg.Code != "42501" && pg.Code != "42883") {
			t.Fatal(name, "not refused:", e)
		}
	}
	// A malformed UUID would fail the cast (22P02); Go rejects it before calling.
	_, e = m.pool.Exec(ctx, `SELECT public.codex_channels_get($1,$2::uuid)`, site, "not-a-uuid")
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != "22P02" {
		t.Fatal("expected 22P02", e)
	}
	// Raw function output decodes strictly into the Go contract.
	var raw []byte
	if e = m.pool.QueryRow(ctx, `SELECT public.codex_channels_get($1,$2::uuid)`, site, opened.Channel.ID.String()).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var strict Result
	if e = communications.DecodeStrict(raw, &strict); e != nil || strict.Channel == nil || strict.Channel.ID != opened.Channel.ID {
		t.Fatal("strict decode", e, string(raw))
	}
}

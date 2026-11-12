//go:build integration

package channels

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func identityHash(site, channel, user uuid.UUID) string {
	text, _ := binding(site, channel, user)
	sum := sha512.Sum512([]byte(text))
	return hex.EncodeToString(sum[:])
}

// The Site UUID and the guest User UUID come from codex_zones and stay stable
// through Get, Activate, Renew and Close.
func TestChannelIdentityGuestPostgres(t *testing.T) {
	ctx, admin, m, _ := activationDatabase(t)
	site := newSite(t, ctx, admin)
	var siteUUID uuid.UUID
	if e := admin.QueryRow(ctx, `SELECT site_uuid FROM public.sites WHERE site_key=$1`, site).Scan(&siteUUID); e != nil {
		t.Fatal(e)
	}
	a, e := m.Open(ctx, site, "Z1")
	expectOutcome(t, a, e, Opened, StatusPending)
	b, e := m.Open(ctx, site, "Z1")
	expectOutcome(t, b, e, Opened, StatusPending)
	guest := a.Channel.Identity
	if a.Channel.SiteUUID != siteUUID || b.Channel.SiteUUID != siteUUID || guest.State != IdentityGuest || guest.Final || guest.BoundAt != nil ||
		guest.UserUUID.Version() != 4 || guest.UserUUID == b.Channel.Identity.UserUUID || guest.UserUUID == a.Channel.ID {
		t.Fatalf("opened identity %+v / %+v", a.Channel, b.Channel)
	}
	check := func(name string, r Result, e error, outcome, status string) {
		t.Helper()
		expectOutcome(t, r, e, outcome, status)
		if r.Channel.ID != a.Channel.ID || r.Channel.SiteUUID != siteUUID || r.Channel.Identity.UserUUID != guest.UserUUID || r.Channel.Identity.State != IdentityGuest {
			t.Fatal(name, "changed the identity", r.Channel)
		}
	}
	r, e := m.Get(ctx, site, a.Channel.ID)
	check("get", r, e, Found, StatusPending)
	r, e = m.Activate(ctx, site, a.Channel.ID)
	check("activate", r, e, Activated, StatusActive)
	if *r.Channel.Protection.CodexHash != identityHash(siteUUID, a.Channel.ID, guest.UserUUID) || *r.Channel.Protection.Binding != BindingRepresentation {
		t.Fatal("guest channel hash", r.Channel.Protection)
	}
	hash := *r.Channel.Protection.CodexHash
	r, e = m.Renew(ctx, site, a.Channel.ID)
	check("renew", r, e, Renewed, StatusActive)
	if *r.Channel.Protection.CodexHash != hash {
		t.Fatal("renewal changed the hash")
	}
	r, e = m.Close(ctx, site, a.Channel.ID)
	check("close", r, e, Closed, StatusClosed)
	if *r.Channel.Protection.CodexHash != hash {
		t.Fatal("closing changed the hash")
	}
	// The Site UUID is immutable even for the schema owner.
	tx, e := admin.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = tx.Exec(ctx, "SET LOCAL ROLE codex_zones_admin")
	_, e = tx.Exec(ctx, `UPDATE public.sites SET site_uuid = gen_random_uuid() WHERE site_key=$1`, site)
	_ = tx.Rollback(ctx)
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != "CX409" {
		t.Fatal("site uuid change not refused", e)
	}
}

// Guest → authenticated once: same channel, same TIU, zone and status; the hash
// changes once to the account's binding and is then final.
func TestChannelIdentityBindPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	opened, e := m.Open(ctx, site, "Z12")
	expectOutcome(t, opened, e, Opened, StatusPending)
	id := opened.Channel.ID
	active, e := m.Activate(ctx, site, id)
	expectOutcome(t, active, e, Activated, StatusActive)
	before := *active.Channel
	user, other := uuid.New(), uuid.New()
	hashes, _ := f.calls()

	bound, e := m.BindIdentity(ctx, site, id, user)
	expectOutcome(t, bound, e, Bound, StatusActive)
	c, p := bound.Channel, bound.Channel.Protection
	want := identityHash(before.SiteUUID, id, user)
	if c.ID != id || c.SiteUUID != before.SiteUUID || c.Zone != "Z12" || !c.OpenedAt.Equal(before.OpenedAt) ||
		c.Identity.State != IdentityAuthenticated || c.Identity.UserUUID != user || !c.Identity.Final || c.Identity.BoundAt == nil ||
		*p.CodexHash != want || *p.CodexHash == *before.Protection.CodexHash || *p.Binding != BindingRepresentation ||
		*p.TIU != *before.Protection.TIU || *p.TIUScheme != *before.Protection.TIUScheme || !p.ProtectedAt.Equal(*before.Protection.ProtectedAt) ||
		!c.LastActivityAt.Equal(before.LastActivityAt) || !c.ExpiresAt.Equal(before.ExpiresAt) {
		t.Fatalf("bound channel %+v\nbefore %+v", c, before)
	}
	if sent := f.sent(); len(sent) != hashes+1 || sent[len(sent)-1] != mustBinding(before.SiteUUID, id, user) {
		t.Fatal("CodexHash was not asked for the account binding", sent)
	}
	var database string
	if e = admin.QueryRow(ctx, `SELECT public.codex_channels_identity_hash(s.site_uuid, c.channel_key, c.user_uuid) FROM public.communication_channels c
		JOIN public.sites s ON s.id = c.site_id WHERE c.channel_key=$1`, id).Scan(&database); e != nil || database != want {
		t.Fatal("database hash", database, e)
	}

	// Repetition and conflict answer from the stored identity without hashing.
	hashes, _ = f.calls()
	again, e := m.BindIdentity(ctx, site, id, user)
	expectOutcome(t, again, e, AlreadyBound, StatusActive)
	conflict, e := m.BindIdentity(ctx, site, id, other)
	expectOutcome(t, conflict, e, IdentityConflict, StatusActive)
	if now, _ := f.calls(); now != hashes {
		t.Fatal("repeated binding asked CodexHash")
	}
	// The database path decides the same for a caller that lost the fast path.
	direct, e := m.bindIdentity(ctx, site, id, user, want)
	expectOutcome(t, direct, e, AlreadyBound, StatusActive)
	direct, e = m.bindIdentity(ctx, site, id, other, identityHash(before.SiteUUID, id, other))
	expectOutcome(t, direct, e, IdentityConflict, StatusActive)
	for _, r := range []Result{again, conflict, direct} {
		if r.Channel.Identity.UserUUID != user || *r.Channel.Protection.CodexHash != want {
			t.Fatal("refusal changed the identity", r.Channel)
		}
	}

	// Renew and Close keep the final identity and hash; nothing rebinds a closed channel.
	renewed, e := m.Renew(ctx, site, id)
	expectOutcome(t, renewed, e, Renewed, StatusActive)
	closed, e := m.Close(ctx, site, id)
	expectOutcome(t, closed, e, Closed, StatusClosed)
	for _, r := range []Result{renewed, closed} {
		if r.Channel.Identity.UserUUID != user || r.Channel.Identity.State != IdentityAuthenticated || *r.Channel.Protection.CodexHash != want ||
			!r.Channel.Identity.BoundAt.Equal(*c.Identity.BoundAt) {
			t.Fatal("lifecycle changed the identity", r.Channel)
		}
	}
	after, e := m.BindIdentity(ctx, site, id, other)
	expectOutcome(t, after, e, Closed, StatusClosed)

	// The owner cannot revert or replace the binding.
	for name, q := range map[string]string{
		"revert":  `UPDATE public.communication_channels SET identity_state='guest', identity_bound_at=NULL WHERE channel_key=$1`,
		"replace": `UPDATE public.communication_channels SET user_uuid=gen_random_uuid() WHERE channel_key=$1`,
		"rehash":  `UPDATE public.communication_channels SET protection_codexhash=repeat('a',128) WHERE channel_key=$1`,
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
			t.Fatal(name, "not refused", e)
		}
	}
}

func mustBinding(site, channel, user uuid.UUID) string {
	text, _ := binding(site, channel, user)
	return text
}

// Only an active v2 channel of the requesting site is bound; every refusal leaves
// the channel as it was and never closes it.
func TestChannelIdentityRefusalsPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site, otherSite := newSite(t, ctx, admin), newSite(t, ctx, admin)
	user := uuid.New()

	if _, e := m.BindIdentity(ctx, site, uuid.New(), uuid.Nil); !errors.Is(e, ErrUserIdentifierInvalid) {
		t.Fatal("nil user", e)
	}
	if _, e := m.BindIdentity(ctx, site, uuid.Nil, user); !errors.Is(e, ErrIdentifierInvalid) {
		t.Fatal("nil channel", e)
	}
	pending, e := m.Open(ctx, site, "Z1")
	expectOutcome(t, pending, e, Opened, StatusPending)
	r, e := m.BindIdentity(ctx, site, pending.Channel.ID, user)
	expectOutcome(t, r, e, Pending, StatusPending)
	expectCode(t, OutcomeError(r.Outcome), "CHANNEL_NOT_ACTIVE")

	active := fixtureChannel(t, ctx, admin, site, StatusActive, 0)
	for _, s := range []string{otherSite, "missing-" + uuid.NewString()} {
		r, e = m.BindIdentity(ctx, s, active, user)
		expectOutcome(t, r, e, NotFound, "")
	}
	r, e = m.BindIdentity(ctx, site, uuid.New(), user)
	expectOutcome(t, r, e, NotFound, "")

	lapsed := fixtureChannel(t, ctx, admin, site, StatusActive, 31*time.Minute)
	r, e = m.BindIdentity(ctx, site, lapsed, user)
	expectOutcome(t, r, e, Expired, StatusExpired)
	closedID := fixtureChannel(t, ctx, admin, site, StatusActive, 0)
	_, _ = m.Close(ctx, site, closedID)
	r, e = m.BindIdentity(ctx, site, closedID, user)
	expectOutcome(t, r, e, Closed, StatusClosed)

	// A channel protected before C11-H keeps its v1 hash and is never bound.
	legacy := fixtureLegacyChannel(t, ctx, admin, site, 0)
	got, e := m.Get(ctx, site, legacy)
	expectOutcome(t, got, e, Found, StatusActive)
	if *got.Channel.Protection.Binding != LegacyBindingRepresentation || !got.Channel.Identity.Final || got.Channel.Identity.State != IdentityGuest {
		t.Fatal("legacy channel", got.Channel)
	}
	r, e = m.BindIdentity(ctx, site, legacy, user)
	expectOutcome(t, r, e, BindingUnsupported, StatusActive)
	direct, e := m.bindIdentity(ctx, site, legacy, user, identityHash(got.Channel.SiteUUID, legacy, user))
	expectOutcome(t, direct, e, BindingUnsupported, StatusActive)
	renewed, e := m.Renew(ctx, site, legacy)
	expectOutcome(t, renewed, e, Renewed, StatusActive)
	if *renewed.Channel.Protection.CodexHash != *got.Channel.Protection.CodexHash {
		t.Fatal("legacy hash changed")
	}
	if hashes, _ := f.calls(); hashes != 0 {
		t.Fatal("a refused binding asked CodexHash", hashes)
	}

	// A hash that is not this channel's account binding is refused by the database.
	current, _ := m.Get(ctx, site, active)
	// The guest hash, another user's, and another site's.
	for _, hash := range []string{*current.Channel.Protection.CodexHash, identityHash(current.Channel.SiteUUID, active, uuid.New()), identityHash(uuid.New(), active, user)} {
		r, e = m.bindIdentity(ctx, site, active, user, hash)
		expectOutcome(t, r, e, ProtectionMismatch, StatusActive)
	}
	// Through BindIdentity a database refusal is internal, never retried.
	f.set(func(f *serviceFixture) {
		f.hashReply = func(string) (int, string) { return genuineHash("another binding") }
	})
	if _, e = m.BindIdentity(ctx, site, active, user); !errors.Is(e, errProtectionMismatch) {
		t.Fatal("mismatch", e)
	}
	expectCode(t, e, "INTERNAL_ERROR")
	// CodexHash unavailable: nothing is written.
	f.set(func(f *serviceFixture) { f.hashReply = func(string) (int, string) { return 503, `{"detail":"down"}` } })
	_, e = m.BindIdentity(ctx, site, active, user)
	expectCode(t, e, "CODEXHASH_UNAVAILABLE")
	// A Go-built input the database refuses is internal, not a client error.
	if _, e = m.bindIdentity(ctx, site, active, user, "short"); !errors.Is(e, errContract) {
		t.Fatal("CX422 not internal", e)
	}
	still, e := m.Get(ctx, site, active)
	expectOutcome(t, still, e, Found, StatusActive)
	if still.Channel.Identity != current.Channel.Identity || *still.Channel.Protection.CodexHash != *current.Channel.Protection.CodexHash {
		t.Fatal("a refused binding changed the channel", still.Channel)
	}
}

// Concurrent bindings for different accounts: exactly one succeeds.
func TestChannelIdentityConcurrencyPostgres(t *testing.T) {
	ctx, admin, m, f := activationDatabase(t)
	site := newSite(t, ctx, admin)
	id := fixtureChannel(t, ctx, admin, site, StatusActive, 0)
	const n = 16
	users := make([]uuid.UUID, n)
	for i := range users {
		users[i] = uuid.New()
	}
	users[n-1] = users[0] // one account submitted twice
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
			results[i], errs[i] = m.BindIdentity(ctx, site, id, users[i])
		}(i)
	}
	wg.Wait()
	final, e := m.Get(ctx, site, id)
	expectOutcome(t, final, e, Found, StatusActive)
	winner := final.Channel.Identity.UserUUID
	counts := map[string]int{}
	for i, r := range results {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		counts[r.Outcome]++
		want := IdentityConflict
		if users[i] == winner {
			want = AlreadyBound
		}
		if r.Outcome != Bound && r.Outcome != want {
			t.Fatal(i, r.Outcome, want)
		}
		if r.Channel.Identity.UserUUID != winner || *r.Channel.Protection.CodexHash != *final.Channel.Protection.CodexHash {
			t.Fatal("conflicting identity", r.Channel.Identity, final.Channel.Identity)
		}
	}
	if counts[Bound] != 1 || final.Channel.Identity.State != IdentityAuthenticated {
		t.Fatal("concurrent binding", counts)
	}
}

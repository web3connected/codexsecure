package channels

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

const fixtureID = "3f0e1c2a-8b4d-4e6f-9a1b-2c3d4e5f6a7b"

const (
	fixtureSiteUUID = "3f25c4a0-7e21-48a6-bb18-4d92e61a7f10"
	fixtureUserUUID = "6a27c391-3e42-4d80-97b1-18c25f60a913"
)

// The C2A specification's example representation, with the C3A protection keys and
// the C11-H Site UUID and identity.
const pendingFixture = `{"result":"found","channel":{"channel":"` + fixtureID + `","site":"web3codex-local","site_uuid":"` + fixtureSiteUUID + `","status":"pending","zone":"Z1",
 "opened_at":"2026-10-27T14:00:00.000000+00:00","last_activity_at":"2026-10-27T14:00:00.000000+00:00",
 "expires_at":"2026-10-27T14:30:00.000000+00:00","closed_at":null,
 "protection":{"status":"not_protected","binding":null,"codexhash":null,"tiu":null,"tiu_scheme":null,"tiu_unit":null,"protected_at":null},
 "identity":{"state":"guest","user_uuid":"` + fixtureUserUUID + `","bound_at":null,"final":false}}}`

func decodeFixture(t *testing.T, raw string) Result {
	t.Helper()
	var result Result
	if e := communications.DecodeStrict([]byte(raw), &result); e != nil {
		t.Fatal(e)
	}
	return result
}

func TestValidateContract(t *testing.T) {
	id := uuid.MustParse(fixtureID)
	ok := decodeFixture(t, pendingFixture)
	if e := validate(ok, "web3codex-local", id, []string{Found, NotFound}); e != nil {
		t.Fatal(e)
	}
	if ok.Channel.ExpiresAt.Sub(ok.Channel.LastActivityAt).Minutes() != 30 || ok.Channel.OpenedAt.Location().String() != "UTC" {
		t.Fatal(ok.Channel)
	}
	notFound := decodeFixture(t, `{"result":"not_found","channel":null}`)
	if e := validate(notFound, "web3codex-local", id, []string{Found, NotFound}); e != nil {
		t.Fatal(e)
	}
	mutate := func(f func(*Result)) Result {
		r := decodeFixture(t, pendingFixture)
		f(&r)
		return r
	}
	hash, tiu, scheme, unit, representation := strings.Repeat("a", 128), "1.500000", codextime.LegacySecondsScheme, "s", BindingRepresentation
	protect := func(r *Result) {
		r.Channel.Status = StatusActive
		r.Channel.Protection = Protection{Status: "protected", Binding: &representation, CodexHash: &hash, TIU: &tiu, TIUScheme: &scheme, TIUUnit: &unit, ProtectedAt: &r.Channel.OpenedAt}
	}
	bad := func(f func(*Protection)) Result {
		return mutate(func(r *Result) { protect(r); f(&r.Channel.Protection) })
	}
	authenticate := func(r *Result) {
		protect(r)
		r.Channel.Identity = Identity{State: IdentityAuthenticated, UserUUID: uuid.New(), BoundAt: &r.Channel.OpenedAt, Final: true}
	}
	legacy := LegacyBindingRepresentation
	protectV1 := func(r *Result) {
		protect(r)
		r.Channel.Protection.Binding = &legacy
		r.Channel.Identity.Final = true
	}
	other, upper, float := "codexsecure-protect-payload-v1", strings.Repeat("A", 128), "1.5"
	for name, bad := range map[string]Result{
		"unexpected result":   mutate(func(r *Result) { r.Outcome = Renewed }),
		"other site":          mutate(func(r *Result) { r.Channel.Site = "other" }),
		"other channel":       mutate(func(r *Result) { r.Channel.ID = uuid.New() }),
		"not found with data": mutate(func(r *Result) { r.Outcome = NotFound }),
		"found without data":  {Outcome: Found},
		"unknown status":      mutate(func(r *Result) { r.Channel.Status = "open" }),
		"pending protected": mutate(func(r *Result) {
			protect(r)
			r.Channel.Status = StatusPending
		}),
		"missing scheme":      bad(func(p *Protection) { p.TIUScheme = nil }),
		"missing binding":     bad(func(p *Protection) { p.Binding = nil }),
		"other binding":       bad(func(p *Protection) { p.Binding = &other }),
		"uppercase hash":      bad(func(p *Protection) { p.CodexHash = &upper }),
		"non-canonical tiu":   bad(func(p *Protection) { p.TIU = &float }),
		"unit not for scheme": bad(func(p *Protection) { p.TIUUnit = &other }),
		"protected label":     mutate(func(r *Result) { r.Channel.Protection.Status = "protected" }),
		"partial protection":  mutate(func(r *Result) { r.Channel.Protection.CodexHash = &hash }),
		"active unprotected":  mutate(func(r *Result) { r.Channel.Status = StatusActive }),
		"closed without time": mutate(func(r *Result) { r.Channel.Status = StatusClosed }),
		"pending with close":  mutate(func(r *Result) { r.Channel.ClosedAt = &r.Channel.OpenedAt }),
		"result contradicts":  mutate(func(r *Result) { r.Outcome = Expired }),
		// C11-H identity.
		"nil site uuid":           mutate(func(r *Result) { r.Channel.SiteUUID = uuid.Nil }),
		"nil user uuid":           mutate(func(r *Result) { r.Channel.Identity.UserUUID = uuid.Nil }),
		"missing identity":        mutate(func(r *Result) { r.Channel.Identity = Identity{} }),
		"unknown identity state":  mutate(func(r *Result) { r.Channel.Identity.State = "admin" }),
		"guest with bound_at":     mutate(func(r *Result) { r.Channel.Identity.BoundAt = &r.Channel.OpenedAt }),
		"guest final":             mutate(func(r *Result) { r.Channel.Identity.Final = true }),
		"authenticated unbound":   mutate(func(r *Result) { authenticate(r); r.Channel.Identity.BoundAt = nil }),
		"authenticated not final": mutate(func(r *Result) { authenticate(r); r.Channel.Identity.Final = false }),
		"authenticated pending": mutate(func(r *Result) {
			r.Channel.Identity = Identity{State: IdentityAuthenticated, UserUUID: uuid.New(), BoundAt: &r.Channel.OpenedAt, Final: true}
		}),
		"authenticated v1":  mutate(func(r *Result) { authenticate(r); r.Channel.Protection.Binding = &legacy }),
		"v1 not final":      mutate(func(r *Result) { protectV1(r); r.Channel.Identity.Final = false }),
		"bound but pending": mutate(func(r *Result) { r.Outcome = Bound }),
	} {
		outcomes := []string{Found, NotFound, Expired, Bound}
		if e := validate(bad, "web3codex-local", id, outcomes); e == nil {
			t.Fatal(name, "accepted")
		}
	}
	for name, good := range map[string]func(*Result){"guest v2": protect, "authenticated v2": authenticate, "legacy v1": protectV1} {
		if e := validate(mutate(good), "web3codex-local", id, []string{Found}); e != nil {
			t.Fatal(name, "rejected", e)
		}
	}
	if r := mutate(authenticate); validate(r, "web3codex-local", id, []string{Found}) != nil || r.Channel.Identity.BoundAt.Location() != time.UTC {
		t.Fatal("bound_at is not normalized to UTC")
	}
	// A response from a codex_zones database without C11-H has no identity.
	preC11H := strings.Replace(strings.Replace(pendingFixture, `"site_uuid":"`+fixtureSiteUUID+`",`, "", 1),
		`,
 "identity":{"state":"guest","user_uuid":"`+fixtureUserUUID+`","bound_at":null,"final":false}`, "", 1)
	if r := decodeFixture(t, preC11H); validate(r, "web3codex-local", id, []string{Found}) == nil {
		t.Fatal("representation without identity accepted")
	}
	var r Result
	if e := communications.DecodeStrict([]byte(`{"result":"found","channel":null,"extra":1}`), &r); e == nil {
		t.Fatal("unknown response field accepted")
	}
}

func TestMapErrorAndOutcomes(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{ErrIdentifierInvalid, 422, "CHANNEL_IDENTIFIER_INVALID"},
		{&pgconn.PgError{Code: "CX404", Message: "Site not found"}, 404, "SITE_NOT_FOUND"},
		{&pgconn.PgError{Code: "CX422"}, 422, "CHANNEL_REQUEST_INVALID"},
		{&pgconn.PgError{Code: "CX409"}, 500, "INTERNAL_ERROR"},
		{&pgconn.PgError{Code: "23514"}, 500, "INTERNAL_ERROR"},
		{&pgconn.PgError{Code: "22P02"}, 500, "INTERNAL_ERROR"},
		{errContract, 500, "INTERNAL_ERROR"},
		{errors.New("boom"), 500, "INTERNAL_ERROR"},
		{OutcomeError(NotFound), 404, "CHANNEL_NOT_FOUND"},
		{OutcomeError(Pending), 409, "CHANNEL_NOT_ACTIVE"},
		{OutcomeError(Closed), 409, "CHANNEL_CLOSED"},
		{OutcomeError(AlreadyClosed), 409, "CHANNEL_CLOSED"},
		{OutcomeError(Expired), 409, "CHANNEL_EXPIRED"},
		{OutcomeError(AlreadyExpired), 409, "CHANNEL_EXPIRED"},
		{OutcomeError(IdentityConflict), 409, "CHANNEL_IDENTITY_CONFLICT"},
		{OutcomeError(BindingUnsupported), 409, "CHANNEL_BINDING_UNSUPPORTED"},
		{ErrUserIdentifierInvalid, 422, "CHANNEL_REQUEST_INVALID"},
		{ErrCodexHashUnavailable, 503, "CODEXHASH_UNAVAILABLE"},
		{ErrCodexHashInvalid, 502, "CODEXHASH_INVALID_RESPONSE"},
		{ErrCodexTimeUnavailable, 503, "CODEXTIME_UNAVAILABLE"},
		{ErrCodexTimeInvalid, 502, "CODEXTIME_INVALID_RESPONSE"},
		{errProtectionMismatch, 500, "INTERNAL_ERROR"},
	} {
		if api := MapError(tc.err); api.Status != tc.status || api.Code != tc.code {
			t.Fatal(tc.err, api)
		}
	}
	for _, outcome := range []string{Opened, Found, Renewed, NotDue, Activated, AlreadyActive, Bound, AlreadyBound} {
		if OutcomeError(outcome) != nil {
			t.Fatal(outcome)
		}
	}
}

// Golden vectors of codexsecure-channel-binding-v2. The first is the canonical text
// asserted by the Web3Codex C11-H PostgreSQL tests (CommunicationChannelIdentityTest)
// against codex_channels_binding_v2; the digests were computed independently.
func TestBindingGoldenVectors(t *testing.T) {
	for _, tc := range []struct {
		site, channel, user string
		text, sha512        string
	}{
		{"3f25c4a0-7e21-48a6-bb18-4d92e61a7f10", "9b41d802-6e73-4d15-a840-72f8c16e9b20", "6a27c391-3e42-4d80-97b1-18c25f60a913",
			`{"channel_uuid":"9b41d802-6e73-4d15-a840-72f8c16e9b20","representation":"codexsecure-channel-binding-v2","site_uuid":"3f25c4a0-7e21-48a6-bb18-4d92e61a7f10","user_uuid":"6a27c391-3e42-4d80-97b1-18c25f60a913"}`,
			"cde46785fdfc64692c8fd336537c3a6e62a03288fbd71952161cc3b5f24454e8fc3cae66f76c8863c7c4d741aa406480d7c88a93e0699134bc6036ee503bbc0d"},
		// Uppercase and braced input spellings yield the same canonical lowercase text.
		{"{00000000-0000-4000-8000-000000000001}", "3F0E1C2A-8B4D-4E6F-9A1B-2C3D4E5F6A7B", "urn:uuid:7c81b7f2-9f54-4c12-8a5b-6d203e7f4190",
			`{"channel_uuid":"3f0e1c2a-8b4d-4e6f-9a1b-2c3d4e5f6a7b","representation":"codexsecure-channel-binding-v2","site_uuid":"00000000-0000-4000-8000-000000000001","user_uuid":"7c81b7f2-9f54-4c12-8a5b-6d203e7f4190"}`,
			"dd1915532bbdee16c34223868c49da5b126a672e10257399a2cee214b341d965aaa5e28768ce0257a368275e73dbf4242419ce40e7ecb39bd74f32a34931ba02"},
	} {
		text, e := binding(uuid.MustParse(tc.site), uuid.MustParse(tc.channel), uuid.MustParse(tc.user))
		if e != nil || text != tc.text {
			t.Fatalf("binding %q, %v\nwant    %q", text, e, tc.text)
		}
		if got := sha512Hash(text); got != tc.sha512 {
			t.Fatal("sha512", got)
		}
	}
	// Each UUID has its own field: swapping two changes the hash.
	a, b, u := uuid.New(), uuid.New(), uuid.New()
	x, _ := binding(a, b, u)
	y, _ := binding(b, a, u)
	if sha512Hash(x) == sha512Hash(y) {
		t.Fatal("swapped site and channel produce the same hash")
	}
}

func TestBoundUnchanged(t *testing.T) {
	hash, bound := strings.Repeat("b", 128), strings.Repeat("c", 128)
	tiu, scheme, unit, representation := "1.500000", codextime.LegacySecondsScheme, "s", BindingRepresentation
	user := uuid.New()
	before := *decodeFixture(t, pendingFixture).Channel
	before.Status = StatusActive
	before.Protection = Protection{Status: "protected", Binding: &representation, CodexHash: &hash, TIU: &tiu, TIUScheme: &scheme, TIUUnit: &unit, ProtectedAt: &before.OpenedAt}
	after := func(f func(*Channel)) Channel {
		c := before
		p := c.Protection
		h, t, at := bound, *p.TIU, *p.ProtectedAt
		p.CodexHash, p.TIU, p.ProtectedAt = &h, &t, &at
		c.Protection = p
		now := before.OpenedAt.Add(time.Minute)
		c.Identity = Identity{State: IdentityAuthenticated, UserUUID: user, BoundAt: &now, Final: true}
		f(&c)
		return c
	}
	if e := boundUnchanged(before, after(func(*Channel) {}), user, bound); e != nil {
		t.Fatal(e)
	}
	renewed := after(func(c *Channel) {
		c.LastActivityAt, c.ExpiresAt = c.LastActivityAt.Add(time.Minute), c.ExpiresAt.Add(time.Minute)
	})
	if e := boundUnchanged(before, renewed, user, bound); e != nil {
		t.Fatal("a concurrent renewal was refused", e)
	}
	other := "2.000000"
	for name, c := range map[string]Channel{
		"other user":    after(func(c *Channel) { c.Identity.UserUUID = uuid.New() }),
		"still guest":   after(func(c *Channel) { c.Identity = before.Identity }),
		"other hash":    after(func(c *Channel) { c.Protection.CodexHash = &hash }),
		"tiu changed":   after(func(c *Channel) { c.Protection.TIU = &other }),
		"protected_at":  after(func(c *Channel) { t := c.OpenedAt.Add(time.Second); c.Protection.ProtectedAt = &t }),
		"site uuid":     after(func(c *Channel) { c.SiteUUID = uuid.New() }),
		"zone":          after(func(c *Channel) { c.Zone = "Z2" }),
		"activity back": after(func(c *Channel) { c.ExpiresAt = c.ExpiresAt.Add(-time.Second) }),
	} {
		if boundUnchanged(before, c, user, bound) == nil {
			t.Fatal(name, "accepted")
		}
	}
}

func TestValidTIU(t *testing.T) {
	legacy, v1 := codextime.LegacySecondsScheme, "codextime-tiu-2020/1"
	for _, ok := range [][3]string{
		{"214612574.695765", legacy, "s"}, {"214612574.500000", legacy, "s"}, {"0.000001", legacy, "s"}, {"99999999999999.999999", legacy, "s"},
		{"214612574695765961", v1, "ns"}, {"1", v1, "qm"}, {"3980700000000000000000000000000000000000000000000439638", v1, "qm"}, {"1" + strings.Repeat("0", 79), v1, "as"},
	} {
		if !validTIU(ok[0], ok[1], ok[2]) {
			t.Fatal("rejected", ok)
		}
	}
	for _, bad := range [][3]string{
		{"214612574.5", legacy, "s"}, {"214612574.6957651", legacy, "s"}, {"0.000000", legacy, "s"}, {"01.000000", legacy, "s"}, {"-1.000000", legacy, "s"},
		{"1.000000", legacy, "ms"}, {"214612574.695765", legacy, ""}, {"1e3", v1, "ns"}, {"0", v1, "ns"}, {"01", v1, "ns"}, {"1", v1, "NS"},
		{"1", v1, "s "}, {"1" + strings.Repeat("0", 80), v1, "ns"}, {"1.000000", "codextime-tiu-2020/2", "s"}, {"", legacy, "s"},
	} {
		if validTIU(bad[0], bad[1], bad[2]) {
			t.Fatal("accepted", bad)
		}
	}
}

type hasherStub struct {
	result codexhash.PayloadHashResult
	err    error
	data   []string
}

func (h *hasherStub) PayloadHash(_ context.Context, data, algo string, size int) (codexhash.PayloadHashResult, error) {
	if algo != "sha512" || size != 512 {
		return codexhash.PayloadHashResult{}, errors.New("unexpected request")
	}
	h.data = append(h.data, data)
	return h.result, h.err
}

type clockStub struct {
	tiu codextime.ExactTIU
	err error
}

func (c clockStub) CurrentTIUExact(context.Context) (codextime.ExactTIU, error) { return c.tiu, c.err }

func TestProtectionHashValidation(t *testing.T) {
	good := sha512Hash("x")
	if h, e := (&Manager{hasher: &hasherStub{result: codexhash.PayloadHashResult{PayloadHash: good, Algo: "sha512", HashSize: 512}}}).protectionHash(context.Background(), "x"); e != nil || h != good {
		t.Fatal(h, e)
	}
	for name, tc := range map[string]struct {
		hasher PayloadHasher
		want   error
	}{
		"not configured": {nil, ErrCodexHashUnavailable},
		"unavailable":    {&hasherStub{err: codexhash.ErrUnavailable}, ErrCodexHashUnavailable},
		"empty result":   {&hasherStub{}, ErrCodexHashInvalid},
		"sha256":         {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: good[:64], Algo: "sha256", HashSize: 256}}, ErrCodexHashInvalid},
		"wrong algo":     {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: good, Algo: "sha256", HashSize: 512}}, ErrCodexHashInvalid},
		"wrong size":     {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: good, Algo: "sha512", HashSize: 256}}, ErrCodexHashInvalid},
		"uppercase":      {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: strings.ToUpper(good), Algo: "sha512", HashSize: 512}}, ErrCodexHashInvalid},
		"short":          {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: good[:127], Algo: "sha512", HashSize: 512}}, ErrCodexHashInvalid},
		"padded":         {&hasherStub{result: codexhash.PayloadHashResult{PayloadHash: " " + good, Algo: "sha512", HashSize: 512}}, ErrCodexHashInvalid},
	} {
		if h, e := (&Manager{hasher: tc.hasher}).protectionHash(context.Background(), "x"); !errors.Is(e, tc.want) || h != "" {
			t.Fatal(name, h, e)
		}
	}
}

func TestProtectionTIUValidation(t *testing.T) {
	good := codextime.ExactTIU{Value: "214612574.695765", Scheme: codextime.LegacySecondsScheme, Unit: "s"}
	if got, e := (&Manager{clock: clockStub{tiu: good}}).protectionTIU(context.Background()); e != nil || got != good {
		t.Fatal(got, e)
	}
	for name, tc := range map[string]struct {
		clock TIUSource
		want  error
	}{
		"not configured": {nil, ErrCodexTimeUnavailable},
		"unavailable":    {clockStub{err: codextime.ErrTIUUnavailable}, ErrCodexTimeUnavailable},
		"cached":         {clockStub{err: codextime.ErrTIUCached}, ErrCodexTimeUnavailable},
		"invalid":        {clockStub{err: codextime.ErrTIUInvalid}, ErrCodexTimeInvalid},
		"zero value":     {clockStub{}, ErrCodexTimeInvalid},
		"float text":     {clockStub{tiu: codextime.ExactTIU{Value: "214612574.5", Scheme: codextime.LegacySecondsScheme, Unit: "s"}}, ErrCodexTimeInvalid},
		"no scheme":      {clockStub{tiu: codextime.ExactTIU{Value: "214612574.695765", Unit: "s"}}, ErrCodexTimeInvalid},
		"wrong unit":     {clockStub{tiu: codextime.ExactTIU{Value: "214612574.695765", Scheme: codextime.LegacySecondsScheme, Unit: "us"}}, ErrCodexTimeInvalid},
		// A value with an error is never used, even if it looks valid.
		"value and error": {clockStub{tiu: good, err: codextime.ErrTIUUnavailable}, ErrCodexTimeUnavailable},
	} {
		if got, e := (&Manager{clock: tc.clock}).protectionTIU(context.Background()); !errors.Is(e, tc.want) || got != (codextime.ExactTIU{}) {
			t.Fatal(name, got, e)
		}
	}
}

func sha512Hash(text string) string {
	sum := sha512.Sum512([]byte(text))
	return hex.EncodeToString(sum[:])
}

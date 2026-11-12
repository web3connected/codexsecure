// Package channels is the CommunicationChannelManager. The codex_zones schema
// (Web3Codex) owns channel identity, lifecycle rules, the 30-minute inactivity
// timeout and all times; the manager only calls its SECURITY DEFINER functions as
// codex_zones_runtime, which has no table privileges.
package channels

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3connected/codexsecure-api/internal/communications"
)

type Manager struct {
	pool   *pgxpool.Pool
	hasher PayloadHasher
	clock  TIUSource
}

// NewManager takes the CodexHash and CodexTime operations used by Activate. A nil
// hasher or clock leaves activation unavailable; it never substitutes a value.
func NewManager(pool *pgxpool.Pool, hasher PayloadHasher, clock TIUSource) *Manager {
	return &Manager{pool: pool, hasher: hasher, clock: clock}
}

// Open creates a pending, unprotected channel. The database generates its
// identity and first deadline; the caller supplies only the site and zone.
func (m *Manager) Open(ctx context.Context, site, zone string) (Result, error) {
	result, e := m.call(ctx, "codex_channels_open", site, zone, uuid.Nil, Opened)
	if e != nil {
		return result, e
	}
	if result.Channel.Status != StatusPending || result.Channel.Zone != zone || result.Channel.Protection.Status != "not_protected" {
		return Result{}, errContract
	}
	return result, nil
}

// Get reads a channel. It never renews; a lapsed channel is reported as expired.
func (m *Manager) Get(ctx context.Context, site string, channel uuid.UUID) (Result, error) {
	return m.call(ctx, "codex_channels_get", site, "", channel, Found, NotFound)
}

// Close ends a pending or active channel. Closing is not communication activity.
func (m *Manager) Close(ctx context.Context, site string, channel uuid.UUID) (Result, error) {
	return m.call(ctx, "codex_channels_close", site, "", channel, Closed, AlreadyClosed, Expired, NotFound)
}

// Expire records a channel whose deadline has passed. It never ends a channel
// early (not_due). Manager-internal: there is no HTTP expiration endpoint.
func (m *Manager) Expire(ctx context.Context, site string, channel uuid.UUID) (Result, error) {
	return m.call(ctx, "codex_channels_expire", site, "", channel, Expired, AlreadyExpired, AlreadyClosed, NotDue, NotFound)
}

// Renew moves an active channel's deadline to 30 minutes after now, for a visitor
// communication that completed the Seven-Fold Process on this channel (Phase C10:
// reported by the site through POST …/channels/{channel}/renew). Open, Get, Close,
// heartbeats, polling and failed communications never call it. Identity and
// protection are unchanged; codex_channels_renew never revives a lapsed channel.
// Any outcome other than renewed is returned with its result and as an error, so
// the communication fails instead of continuing.
func (m *Manager) Renew(ctx context.Context, site string, channel uuid.UUID) (Result, error) {
	result, e := m.call(ctx, "codex_channels_renew", site, "", channel, Renewed, Pending, Closed, Expired, NotFound)
	if e != nil {
		return result, e
	}
	if result.Outcome != Renewed {
		return result, OutcomeError(result.Outcome)
	}
	return result, nil
}

// call runs one lifecycle function as a single autocommit statement, so every
// decision is its own short transaction using database time, and validates the
// response against the C2A/C3A/C11-H contract before anything is returned.
func (m *Manager) call(ctx context.Context, function, site, zone string, channel uuid.UUID, outcomes ...string) (Result, error) {
	var raw []byte
	var e error
	if function == "codex_channels_open" {
		e = m.pool.QueryRow(ctx, `SELECT public.codex_channels_open($1::text,$2::text)`, site, zone).Scan(&raw)
	} else {
		if channel == uuid.Nil {
			return Result{}, ErrIdentifierInvalid
		}
		// function is one of the fixed names above, never caller input.
		e = m.pool.QueryRow(ctx, `SELECT public.`+function+`($1::text,$2::uuid)`, site, channel.String()).Scan(&raw)
	}
	if e != nil {
		return Result{}, e
	}
	return decode(function, raw, site, channel, outcomes)
}

func decode(function string, raw []byte, site string, channel uuid.UUID, outcomes []string) (Result, error) {
	var result Result
	if e := communications.DecodeStrict(raw, &result); e != nil {
		return Result{}, fmt.Errorf("%w: %s: %v", errContract, function, e)
	}
	if e := validate(result, site, channel, outcomes); e != nil {
		return Result{}, fmt.Errorf("%w: %s: %v", errContract, function, e)
	}
	return result, nil
}

func validate(result Result, site string, channel uuid.UUID, outcomes []string) error {
	known := false
	for _, outcome := range outcomes {
		known = known || result.Outcome == outcome
	}
	if !known {
		return fmt.Errorf("unexpected result %q", result.Outcome)
	}
	if (result.Outcome == NotFound) != (result.Channel == nil) {
		return fmt.Errorf("channel presence does not match result %q", result.Outcome)
	}
	c := result.Channel
	if c == nil {
		return nil
	}
	if c.Site != site || c.ID == uuid.Nil || (channel != uuid.Nil && c.ID != channel) {
		return fmt.Errorf("channel identity does not match the request")
	}
	switch c.Status {
	case StatusPending, StatusActive, StatusClosed, StatusExpired:
	default:
		return fmt.Errorf("unknown status %q", c.Status)
	}
	p := c.Protection
	protected := p.Binding != nil && (*p.Binding == BindingRepresentation || *p.Binding == LegacyBindingRepresentation) && p.CodexHash != nil && sha512Hex.MatchString(*p.CodexHash) &&
		p.TIU != nil && p.TIUScheme != nil && p.TIUUnit != nil && validTIU(*p.TIU, *p.TIUScheme, *p.TIUUnit) && p.ProtectedAt != nil
	unprotected := p.Binding == nil && p.CodexHash == nil && p.TIU == nil && p.TIUScheme == nil && p.TIUUnit == nil && p.ProtectedAt == nil
	if !(protected && p.Status == "protected" || unprotected && p.Status == "not_protected") {
		return fmt.Errorf("inconsistent protection")
	}
	if c.Status == StatusPending && !unprotected || c.Status == StatusActive && !protected {
		return fmt.Errorf("status %q contradicts protection", c.Status)
	}
	if (c.Status == StatusClosed) != (c.ClosedAt != nil) {
		return fmt.Errorf("closed_at contradicts status %q", c.Status)
	}
	if !strings.HasPrefix(c.Zone, "Z") {
		return fmt.Errorf("invalid zone")
	}
	if e := validateIdentity(c); e != nil {
		return e
	}
	// Results that name a final state must carry that state. protection_mismatch is
	// absent: activation reports it on a pending channel, binding on an active one.
	if want := map[string]string{Closed: StatusClosed, AlreadyClosed: StatusClosed, Expired: StatusExpired, AlreadyExpired: StatusExpired, Pending: StatusPending, Renewed: StatusActive,
		Activated: StatusActive, AlreadyActive: StatusActive, Bound: StatusActive, AlreadyBound: StatusActive, IdentityConflict: StatusActive, BindingUnsupported: StatusActive}[result.Outcome]; want != "" && c.Status != want {
		return fmt.Errorf("result %q with status %q", result.Outcome, c.Status)
	}
	toUTC(c)
	return nil
}

// validateIdentity checks the C11-H Site UUID and identity: authenticated exactly
// when bound, only on a non-pending v2 channel, and final when authenticated or v1.
func validateIdentity(c *Channel) error {
	id := c.Identity
	if c.SiteUUID == uuid.Nil || id.UserUUID == uuid.Nil {
		return fmt.Errorf("missing site or user uuid")
	}
	if id.State != IdentityGuest && id.State != IdentityAuthenticated {
		return fmt.Errorf("unknown identity state %q", id.State)
	}
	authenticated := id.State == IdentityAuthenticated
	legacy := c.Protection.Binding != nil && *c.Protection.Binding == LegacyBindingRepresentation
	if authenticated != (id.BoundAt != nil) || id.Final != (authenticated || legacy) {
		return fmt.Errorf("inconsistent identity")
	}
	if authenticated && (c.Status == StatusPending || legacy) {
		return fmt.Errorf("identity bound on a channel that cannot be bound")
	}
	return nil
}

func toUTC(c *Channel) {
	c.OpenedAt, c.LastActivityAt, c.ExpiresAt = c.OpenedAt.UTC(), c.LastActivityAt.UTC(), c.ExpiresAt.UTC()
	if c.ClosedAt != nil {
		t := c.ClosedAt.UTC()
		c.ClosedAt = &t
	}
	if c.Protection.ProtectedAt != nil {
		t := c.Protection.ProtectedAt.UTC()
		c.Protection.ProtectedAt = &t
	}
	if c.Identity.BoundAt != nil {
		t := c.Identity.BoundAt.UTC()
		c.Identity.BoundAt = &t
	}
}

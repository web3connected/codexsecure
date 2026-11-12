package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
	"github.com/web3connected/codexsecure-api/pkg/codextime"
)

// BindingRepresentation names the canonical channel binding hashed by CodexHash
// (codex_zones C11-H): {"channel_uuid","representation","site_uuid","user_uuid"},
// each UUID in its canonical lowercase text, under the codexsecure-protect-payload-v1
// rules. Values come only from the database row (and, for the identity binding, the
// trusted account UUID), never from a browser.
const BindingRepresentation = "codexsecure-channel-binding-v2"

// LegacyBindingRepresentation is the C3A binding {channel, opened_at, representation,
// site, zone}. It is never issued again; channels protected under it keep their hash,
// are reported unchanged and cannot be bound to an identity.
const LegacyBindingRepresentation = "codexsecure-channel-binding-v1"

const (
	bindingHashAlgo = "sha512"
	bindingHashSize = 512
)

// PayloadHasher is the CodexHash operation used for protection (codexhash.Client).
// Hash and QuickHash are deliberately absent: they can fall back to a local hash.
type PayloadHasher interface {
	PayloadHash(ctx context.Context, data string, algo string, hashSize int) (codexhash.PayloadHashResult, error)
}

// TIUSource is the fail-closed, lossless CodexTime read (codextime.Client).
type TIUSource interface {
	CurrentTIUExact(ctx context.Context) (codextime.ExactTIU, error)
}

var (
	sha512Hex = regexp.MustCompile(`^[0-9a-f]{128}$`)
	// The canonical TIU text of each scheme, as codex_channels_tiu_valid admits it.
	v1TIU     = regexp.MustCompile(`^[1-9][0-9]{0,79}$`)
	legacyTIU = regexp.MustCompile(`^(0|[1-9][0-9]{0,13})\.[0-9]{6}$`)
	zeroTIU   = regexp.MustCompile(`^0\.0+$`)
	v1Units   = map[string]bool{"s": true, "ms": true, "us": true, "ns": true, "ps": true, "fs": true, "as": true, "qm": true}
)

func validTIU(value, scheme, unit string) bool {
	switch scheme {
	case "codextime-tiu-2020/1":
		return v1Units[unit] && v1TIU.MatchString(value)
	case codextime.LegacySecondsScheme:
		return unit == codextime.LegacySecondsUnit && legacyTIU.MatchString(value) && !zeroTIU.MatchString(value)
	}
	return false
}

// binding returns the canonical codexsecure-channel-binding-v2 text of a site,
// channel and user.
func binding(site, channel, user uuid.UUID) (string, error) {
	raw, e := json.Marshal(map[string]string{
		"channel_uuid":   channel.String(),
		"representation": BindingRepresentation,
		"site_uuid":      site.String(),
		"user_uuid":      user.String(),
	})
	if e != nil {
		return "", e
	}
	return protect.CanonicalPayload(raw)
}

// Activate protects a pending channel with a CodexHash of its binding and a
// CodexTime TIU, and makes it active through codex_channels_activate, which
// rebuilds and checks the binding itself and restarts the inactivity deadline.
// Both services are asked before the database call and outside any transaction;
// if either fails, nothing is written and the channel stays pending. A channel
// that is already active keeps its first protection (already_active).
func (m *Manager) Activate(ctx context.Context, site string, channel uuid.UUID) (Result, error) {
	current, e := m.Get(ctx, site, channel)
	if e != nil || current.Outcome == NotFound {
		return current, e
	}
	// A fast path only: the database decides again under its row locks.
	switch current.Channel.Status {
	case StatusActive:
		return Result{Outcome: AlreadyActive, Channel: current.Channel}, nil
	case StatusClosed:
		return Result{Outcome: Closed, Channel: current.Channel}, nil
	case StatusExpired:
		return Result{Outcome: Expired, Channel: current.Channel}, nil
	}
	// The guest binding: the database generated the user UUID at Open.
	text, e := binding(current.Channel.SiteUUID, current.Channel.ID, current.Channel.Identity.UserUUID)
	if e != nil {
		return Result{}, fmt.Errorf("%w: binding: %v", errContract, e)
	}
	hash, e := m.protectionHash(ctx, text)
	if e != nil {
		activationWarning(site, channel, e, "channel protection hash unavailable")
		return Result{}, e
	}
	tiu, e := m.protectionTIU(ctx)
	if e != nil {
		activationWarning(site, channel, e, "channel protection TIU unavailable")
		return Result{}, e
	}
	result, e := m.activate(ctx, site, channel, hash, tiu)
	if e != nil {
		return Result{}, e
	}
	switch result.Outcome {
	case ProtectionMismatch:
		// Go and the database disagree on the binding, or the hash was altered.
		// Never retried with another hash.
		log.Error().Str("operation", "channels.activate").Str("site", site).Str("channel", channel.String()).Msg("database refused the channel protection hash")
		return Result{}, errProtectionMismatch
	case Activated:
		p, c := result.Channel.Protection, result.Channel
		if *p.Binding != BindingRepresentation || *p.CodexHash != hash || *p.TIU != tiu.Value || *p.TIUScheme != tiu.Scheme || *p.TIUUnit != tiu.Unit ||
			!p.ProtectedAt.Equal(c.LastActivityAt) || !c.ExpiresAt.After(c.LastActivityAt) ||
			c.SiteUUID != current.Channel.SiteUUID || c.Identity.State != IdentityGuest || c.Identity.UserUUID != current.Channel.Identity.UserUUID {
			return Result{}, fmt.Errorf("%w: codex_channels_activate: stored protection differs from the supplied values", errContract)
		}
	}
	return result, nil
}

// protectionHash asks CodexHash for the SHA-512 payload hash of the binding. There
// is no local fallback; the database independently recomputes the hash.
func (m *Manager) protectionHash(ctx context.Context, text string) (string, error) {
	if m.hasher == nil {
		return "", fmt.Errorf("%w: not configured", ErrCodexHashUnavailable)
	}
	r, e := m.hasher.PayloadHash(ctx, text, bindingHashAlgo, bindingHashSize)
	if e != nil {
		return "", fmt.Errorf("%w: %v", ErrCodexHashUnavailable, e)
	}
	if r.Algo != bindingHashAlgo || r.HashSize != bindingHashSize || !sha512Hex.MatchString(r.PayloadHash) {
		return "", fmt.Errorf("%w: algo %q, size %d", ErrCodexHashInvalid, r.Algo, r.HashSize)
	}
	return r.PayloadHash, nil
}

// protectionTIU obtains a TIU exactly as CodexTime issued it, in a scheme the
// database accepts. It is never cached, fabricated or converted through float64.
func (m *Manager) protectionTIU(ctx context.Context) (codextime.ExactTIU, error) {
	if m.clock == nil {
		return codextime.ExactTIU{}, fmt.Errorf("%w: not configured", ErrCodexTimeUnavailable)
	}
	t, e := m.clock.CurrentTIUExact(ctx)
	switch {
	case errors.Is(e, codextime.ErrTIUInvalid):
		return codextime.ExactTIU{}, fmt.Errorf("%w: %v", ErrCodexTimeInvalid, e)
	case e != nil:
		return codextime.ExactTIU{}, fmt.Errorf("%w: %v", ErrCodexTimeUnavailable, e)
	case !validTIU(t.Value, t.Scheme, t.Unit):
		return codextime.ExactTIU{}, fmt.Errorf("%w: unsupported scheme %q or unit %q", ErrCodexTimeInvalid, t.Scheme, t.Unit)
	}
	return t, nil
}

// activate calls codex_channels_activate once, as its own autocommit statement.
func (m *Manager) activate(ctx context.Context, site string, channel uuid.UUID, hash string, tiu codextime.ExactTIU) (Result, error) {
	if channel == uuid.Nil {
		return Result{}, ErrIdentifierInvalid
	}
	var raw []byte
	e := m.pool.QueryRow(ctx, `SELECT public.codex_channels_activate($1::text,$2::uuid,$3::text,$4::text,$5::text,$6::text)`,
		site, channel.String(), hash, tiu.Value, tiu.Scheme, tiu.Unit).Scan(&raw)
	var database *pgconn.PgError
	if errors.As(e, &database) && database.Code == "CX422" {
		// Go validated every value it built, so this is not a client error.
		return Result{}, fmt.Errorf("%w: codex_channels_activate refused its input", errContract)
	}
	if e != nil {
		return Result{}, e
	}
	return decode("codex_channels_activate", raw, site, channel, []string{Activated, AlreadyActive, ProtectionMismatch, Closed, Expired, NotFound})
}

func activationWarning(site string, channel uuid.UUID, e error, message string) {
	log.Warn().Err(e).Str("operation", "channels.activate").Str("site", site).Str("channel", channel.String()).Msg(message)
}

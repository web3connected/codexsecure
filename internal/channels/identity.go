package channels

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

// BindIdentity links an active guest channel to an authenticated account's
// permanent User UUID, once (C11-H). The trusted site application supplies the
// UUID after its own successful login; Go never reads credentials or sessions.
// CodexHash hashes the v2 binding for that UUID before the database call and
// outside any transaction; codex_channels_bind_identity then rebuilds the binding,
// decides under its row locks and replaces user_uuid and the CodexHash together.
// The same UUID again is already_bound and writes nothing; another UUID is
// identity_conflict. The channel UUID, zone, status, lifecycle times, TIU and
// protected_at never change, and a refusal never closes the channel.
func (m *Manager) BindIdentity(ctx context.Context, site string, channel, user uuid.UUID) (Result, error) {
	if user == uuid.Nil {
		return Result{}, ErrUserIdentifierInvalid
	}
	current, e := m.Get(ctx, site, channel)
	if e != nil || current.Outcome == NotFound {
		return current, e
	}
	// A fast path only, without hashing: the database decides again under its locks.
	c := current.Channel
	switch {
	case c.Status != StatusActive:
		return Result{Outcome: c.Status, Channel: c}, nil
	case *c.Protection.Binding != BindingRepresentation:
		return Result{Outcome: BindingUnsupported, Channel: c}, nil
	case c.Identity.State == IdentityAuthenticated && c.Identity.UserUUID == user:
		return Result{Outcome: AlreadyBound, Channel: c}, nil
	case c.Identity.State == IdentityAuthenticated:
		return Result{Outcome: IdentityConflict, Channel: c}, nil
	}
	text, e := binding(c.SiteUUID, c.ID, user)
	if e != nil {
		return Result{}, fmt.Errorf("%w: binding: %v", errContract, e)
	}
	hash, e := m.protectionHash(ctx, text)
	if e != nil {
		log.Warn().Err(e).Str("operation", "channels.bind_identity").Str("site", site).Str("channel", channel.String()).Msg("channel identity hash unavailable")
		return Result{}, e
	}
	result, e := m.bindIdentity(ctx, site, channel, user, hash)
	if e != nil {
		return Result{}, e
	}
	switch result.Outcome {
	case ProtectionMismatch:
		// Go and the database disagree on the binding. Never retried with another hash.
		log.Error().Str("operation", "channels.bind_identity").Str("site", site).Str("channel", channel.String()).Msg("database refused the channel identity hash")
		return Result{}, errProtectionMismatch
	case Bound:
		if e := boundUnchanged(*c, *result.Channel, user, hash); e != nil {
			return Result{}, fmt.Errorf("%w: codex_channels_bind_identity: %v", errContract, e)
		}
	}
	return result, nil
}

// boundUnchanged checks that the binding replaced only the identity and the CodexHash.
// Activity may only have moved forward: a concurrent renewal is legitimate.
func boundUnchanged(before, after Channel, user uuid.UUID, hash string) error {
	p, q := before.Protection, after.Protection
	switch {
	case after.Identity.State != IdentityAuthenticated || after.Identity.UserUUID != user || !after.Identity.Final || after.Identity.BoundAt == nil:
		return errors.New("identity was not bound to the supplied user")
	case *q.CodexHash != hash || *q.Binding != BindingRepresentation:
		return errors.New("stored hash differs from the supplied hash")
	case after.SiteUUID != before.SiteUUID || after.Zone != before.Zone || !after.OpenedAt.Equal(before.OpenedAt):
		return errors.New("channel identity changed")
	case *q.TIU != *p.TIU || *q.TIUScheme != *p.TIUScheme || *q.TIUUnit != *p.TIUUnit || !q.ProtectedAt.Equal(*p.ProtectedAt):
		return errors.New("activation protection changed")
	case after.LastActivityAt.Before(before.LastActivityAt) || after.ExpiresAt.Before(before.ExpiresAt):
		return errors.New("channel activity moved backwards")
	}
	return nil
}

// bindIdentity calls codex_channels_bind_identity once, as its own autocommit statement.
func (m *Manager) bindIdentity(ctx context.Context, site string, channel, user uuid.UUID, hash string) (Result, error) {
	var raw []byte
	e := m.pool.QueryRow(ctx, `SELECT public.codex_channels_bind_identity($1::text,$2::uuid,$3::uuid,$4::text)`,
		site, channel.String(), user.String(), hash).Scan(&raw)
	var database *pgconn.PgError
	if errors.As(e, &database) && database.Code == "CX422" {
		// Go validated every value it built, so this is not a client error.
		return Result{}, fmt.Errorf("%w: codex_channels_bind_identity refused its input", errContract)
	}
	if e != nil {
		return Result{}, e
	}
	return decode("codex_channels_bind_identity", raw, site, channel,
		[]string{Bound, AlreadyBound, IdentityConflict, BindingUnsupported, ProtectionMismatch, Pending, Closed, Expired, NotFound})
}

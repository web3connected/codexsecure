package channels

import (
	"time"

	"github.com/google/uuid"
)

// Lifecycle results returned by the codex_zones channel functions.
const (
	Opened         = "opened"
	Found          = "found"
	NotFound       = "not_found"
	Renewed        = "renewed"
	Closed         = "closed"
	AlreadyClosed  = "already_closed"
	Expired        = "expired"
	AlreadyExpired = "already_expired"
	NotDue         = "not_due"
	Pending        = "pending"
	// Activation (codex_channels_activate, C3A).
	Activated          = "activated"
	AlreadyActive      = "already_active"
	ProtectionMismatch = "protection_mismatch"
	// Identity binding (codex_channels_bind_identity, C11-H).
	Bound              = "bound"
	AlreadyBound       = "already_bound"
	IdentityConflict   = "identity_conflict"
	BindingUnsupported = "binding_unsupported"
)

// Identity states (communication_channels.identity_state).
const (
	IdentityGuest         = "guest"
	IdentityAuthenticated = "authenticated"
)

// Channel statuses (communication_channels.status).
const (
	StatusPending = "pending"
	StatusActive  = "active"
	StatusClosed  = "closed"
	StatusExpired = "expired"
)

// Protection holds the genuine CodexHash and CodexTime values written once by
// activation. The database reports not_protected until then; Go never fills these
// fields itself. TIU is the exact decimal text the database returns as a JSON
// string, so it is never read or re-encoded as a float.
type Protection struct {
	Status      string     `json:"status"`
	Binding     *string    `json:"binding"`
	CodexHash   *string    `json:"codexhash"`
	TIU         *string    `json:"tiu"`
	TIUScheme   *string    `json:"tiu_scheme"`
	TIUUnit     *string    `json:"tiu_unit"`
	ProtectedAt *time.Time `json:"protected_at"`
}

// Identity is the channel's participant (C11-H). The database generates the guest
// UUID at Open; the one authenticated binding replaces it, and then it is final.
// A channel protected under binding v1 names no identity and is final too.
type Identity struct {
	State    string     `json:"state"`
	UserUUID uuid.UUID  `json:"user_uuid"`
	BoundAt  *time.Time `json:"bound_at"`
	Final    bool       `json:"final"`
}

// Channel is the codex_channels_view representation. All times are database times.
type Channel struct {
	ID             uuid.UUID  `json:"channel"`
	Site           string     `json:"site"`
	SiteUUID       uuid.UUID  `json:"site_uuid"`
	Status         string     `json:"status"`
	Zone           string     `json:"zone"`
	OpenedAt       time.Time  `json:"opened_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	ClosedAt       *time.Time `json:"closed_at"`
	Protection     Protection `json:"protection"`
	Identity       Identity   `json:"identity"`
}

// Result is the {result, channel} object of every lifecycle function.
type Result struct {
	Outcome string   `json:"result"`
	Channel *Channel `json:"channel"`
}

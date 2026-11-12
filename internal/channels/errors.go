package channels

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/web3connected/codexsecure-api/internal/communications"
)

// ErrIdentifierInvalid is returned before any database call for a nil channel identity.
var ErrIdentifierInvalid = errors.New("channel identifier is invalid")

// ErrUserIdentifierInvalid is returned before any call for a nil account UUID.
var ErrUserIdentifierInvalid = errors.New("user identifier is invalid")

// errContract reports a function response that does not match the C2A/C3A
// contract. It is never shown to clients.
var errContract = errors.New("channel function response violates the contract")

// Activation failures. The channel stays pending in every case.
var (
	ErrCodexHashUnavailable = errors.New("codexhash could not hash the channel binding")
	ErrCodexHashInvalid     = errors.New("codexhash returned an invalid channel binding hash")
	ErrCodexTimeUnavailable = errors.New("codextime could not provide a tiu")
	ErrCodexTimeInvalid     = errors.New("codextime returned an invalid tiu")
	// errProtectionMismatch: the database rebuilt the binding and refused the hash.
	errProtectionMismatch = errors.New("channel protection does not match the channel")
)

// MapError returns the existing {error,code} envelope without database detail.
// Channels need their own mapping: communications.MapError maps "Site not found"
// to a discovery-batch code.
func MapError(err error) communications.APIError {
	var api communications.APIError
	if errors.As(err, &api) {
		return api
	}
	switch {
	case errors.Is(err, ErrIdentifierInvalid):
		return communications.APIError{Status: 422, Message: "channel identifier is invalid", Code: "CHANNEL_IDENTIFIER_INVALID"}
	case errors.Is(err, ErrUserIdentifierInvalid):
		return communications.APIError{Status: 422, Message: "user identifier is invalid", Code: "CHANNEL_REQUEST_INVALID"}
	case errors.Is(err, ErrCodexHashUnavailable):
		return communications.APIError{Status: 503, Message: "CodexHash could not protect the channel", Code: "CODEXHASH_UNAVAILABLE"}
	case errors.Is(err, ErrCodexHashInvalid):
		return communications.APIError{Status: 502, Message: "CodexHash returned an unexpected result", Code: "CODEXHASH_INVALID_RESPONSE"}
	case errors.Is(err, ErrCodexTimeUnavailable):
		return communications.APIError{Status: 503, Message: "CodexTime could not provide a TIU", Code: "CODEXTIME_UNAVAILABLE"}
	case errors.Is(err, ErrCodexTimeInvalid):
		return communications.APIError{Status: 502, Message: "CodexTime returned an unexpected result", Code: "CODEXTIME_INVALID_RESPONSE"}
	}
	var database *pgconn.PgError
	if errors.As(err, &database) {
		switch database.Code {
		case "CX404":
			return communications.APIError{Status: 404, Message: "site not found", Code: "SITE_NOT_FOUND"}
		case "CX422":
			return communications.APIError{Status: 422, Message: "channel request is invalid", Code: "CHANNEL_REQUEST_INVALID"}
		}
	}
	return communications.APIError{Status: http.StatusInternalServerError, Message: "channel service failed", Code: "INTERNAL_ERROR"}
}

// OutcomeError maps a non-successful lifecycle result to its API error, or nil.
// Renewal outcomes use the same codes, so a communication on a channel that is not
// active fails instead of silently continuing.
func OutcomeError(outcome string) error {
	switch outcome {
	case NotFound:
		return communications.APIError{Status: 404, Message: "channel not found", Code: "CHANNEL_NOT_FOUND"}
	case Pending:
		return communications.APIError{Status: 409, Message: "channel is not active", Code: "CHANNEL_NOT_ACTIVE"}
	case Closed, AlreadyClosed:
		return communications.APIError{Status: 409, Message: "channel is closed", Code: "CHANNEL_CLOSED"}
	case Expired, AlreadyExpired:
		return communications.APIError{Status: 409, Message: "channel is expired", Code: "CHANNEL_EXPIRED"}
	case IdentityConflict:
		return communications.APIError{Status: 409, Message: "channel is bound to another identity", Code: "CHANNEL_IDENTITY_CONFLICT"}
	case BindingUnsupported:
		return communications.APIError{Status: 409, Message: "channel cannot be bound to an identity", Code: "CHANNEL_BINDING_UNSUPPORTED"}
	}
	return nil
}

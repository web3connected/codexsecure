package communications

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// APIError deliberately excludes database messages, SQL, and connection details.
// It is suitable for the existing {error,code,errors?} HTTP error envelope.
type APIError struct {
	Status  int              `json:"-"`
	Message string           `json:"error"`
	Code    string           `json:"code"`
	Errors  ValidationErrors `json:"errors,omitempty"`
}

func (e APIError) Error() string { return e.Message }
func MapError(err error) APIError {
	var recording DiscoveryFailureError
	if errors.As(err, &recording) {
		original := MapError(recording.Original)
		fields := ValidationErrors{}
		for key, value := range original.Errors {
			fields[key] = value
		}
		fields["batch.failure_recording"] = []string{"sequence failure could not be recorded; retry or reconcile the discovery batch"}
		original.Errors = fields
		return original
	}
	var api APIError
	if errors.As(err, &api) {
		return api
	}
	var validation ValidationErrors
	if errors.As(err, &validation) {
		return APIError{422, "configuration is invalid", "CONFIGURATION_INVALID", validation}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return APIError{404, "communication not found", "COMMUNICATION_NOT_FOUND", nil}
	}
	var database *pgconn.PgError
	if errors.As(err, &database) {
		switch database.Code {
		case "23514":
			if database.Message == "Observation source page does not own this page-scoped communication" {
				return APIError{409, "source page does not own communication", "REGISTRATION_IDENTITY_CONFLICT", nil}
			}
		case "CX404":
			if database.Message == "Discovery batch not found" || database.Message == "Site not found" || database.Message == "Parent site generation not found" {
				return APIError{404, "discovery batch not found", "DISCOVERY_BATCH_NOT_FOUND", nil}
			}
			return APIError{404, "communication not found", "COMMUNICATION_NOT_FOUND", nil}
		case "CX409":
			if strings.HasPrefix(database.Message, "Discovery batch") {
				return APIError{409, "discovery batch is closed", "DISCOVERY_BATCH_CLOSED", nil}
			}
			return APIError{409, "draft or structure revision is stale", "STALE_REVISION", nil}
		case "CX410":
			if strings.HasPrefix(database.Message, "Replay conflict") {
				return APIError{409, "replay request conflicts with the original request", "REPLAY_CONFLICT", nil}
			}
			return APIError{409, "idempotency key already processed", "ALREADY_PROCESSED", nil}
		case "CX422":
			return APIError{422, "configuration is invalid", "CONFIGURATION_INVALID", ValidationErrors{"document": {"configuration or replay key is invalid"}}}
		case "CX423":
			return APIError{422, "structure is not eligible for approval", "STRUCTURE_INELIGIBLE", nil}
		case "23505":
			// Concurrent requests on different objects can race on the global key.
			// Other uniqueness failures are not administrator replay decisions.
			if database.ConstraintName == "communication_approval_events_idempotency_key_key" {
				return APIError{409, "idempotency key already processed", "ALREADY_PROCESSED", nil}
			}
		}
	}
	return APIError{http.StatusInternalServerError, "configuration service failed", "INTERNAL_ERROR", nil}
}

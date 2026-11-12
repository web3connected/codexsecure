package communications

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
)

func TestDatabaseErrorContract(t *testing.T) {
	for _, tc := range []struct {
		sql    string
		status int
		code   string
	}{{"CX404", 404, "COMMUNICATION_NOT_FOUND"}, {"CX409", 409, "STALE_REVISION"}, {"CX410", 409, "ALREADY_PROCESSED"}, {"CX422", 422, "CONFIGURATION_INVALID"}, {"CX423", 422, "STRUCTURE_INELIGIBLE"}, {"42501", 500, "INTERNAL_ERROR"}, {"23505", 500, "INTERNAL_ERROR"}} {
		t.Run(tc.sql, func(t *testing.T) {
			got := MapError(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: tc.sql, Message: "private database details"}))
			if got.Status != tc.status || got.Code != tc.code || got.Message == "private database details" {
				t.Fatal(got)
			}
		})
	}
	if got := MapError(errors.New("private connection details")); got.Status != 500 {
		t.Fatal(got)
	}
	if got := MapError(ValidationErrors{"zone": {"unknown zone"}}); got.Code != "CONFIGURATION_INVALID" || len(got.Errors) == 0 {
		t.Fatal(got)
	}
	if got := MapError(&pgconn.PgError{Code: "23505", ConstraintName: "communication_approval_events_idempotency_key_key"}); got.Code != "ALREADY_PROCESSED" {
		t.Fatal(got)
	}
}

func TestCoverageErrorMapping(t *testing.T) {
	for _, tc := range []struct{ state, message, code string }{
		{"CX404", "Discovery batch not found", "DISCOVERY_BATCH_NOT_FOUND"},
		{"CX404", "Site not found", "DISCOVERY_BATCH_NOT_FOUND"},
		{"CX404", "Parent site generation not found", "DISCOVERY_BATCH_NOT_FOUND"},
		{"CX409", "Discovery batch sequence 1 was already processed", "DISCOVERY_BATCH_CLOSED"},
		{"CX409", "Discovery batch was superseded by a newer generation", "DISCOVERY_BATCH_CLOSED"},
		{"CX409", "Stale draft", "STALE_REVISION"},
		{"CX410", "Replay conflict: different request", "REPLAY_CONFLICT"},
		{"CX410", "Replay conflict: another administrator", "REPLAY_CONFLICT"},
		{"CX410", "Replay conflict: predates request binding", "REPLAY_CONFLICT"},
		{"CX410", "Idempotency key already used for a different decision", "ALREADY_PROCESSED"},
		{"23514", "Observation source page does not own this page-scoped communication", "REGISTRATION_IDENTITY_CONFLICT"},
	} {
		t.Run(tc.message, func(t *testing.T) {
			if got := MapError(&pgconn.PgError{Code: tc.state, Message: tc.message}); got.Code != tc.code {
				t.Fatal(got)
			}
		})
	}
}
